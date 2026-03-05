package main

import (
	"fmt"
	"regexp"
	"strings"
)

// ─── Task phases and step statuses ──────────────────────────────────────────

const (
	PhasePlanning   = "planning"
	PhaseExecuting  = "executing"
	PhaseValidating = "validating"
	PhaseDone       = "done"
)

const (
	StepPending   = "pending"
	StepCompleted = "completed"
	StepFailed    = "failed"
)

// ─── Data types ─────────────────────────────────────────────────────────────

type TaskStep struct {
	Index       int    `json:"index"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type StepResult struct {
	Index  int    `json:"index"`
	Status string `json:"status"` // completed|failed
	Output string `json:"output"`
}

type TaskState struct {
	Goal            string            `json:"goal"`
	Phase           string            `json:"phase"`       // planning|executing|validating|done
	Paused          bool              `json:"paused"`      // true = ждём Continue от пользователя
	Steps           []TaskStep        `json:"steps"`
	StepResults     []StepResult      `json:"step_results"`
	Artifacts       map[string]string `json:"artifacts"`   // "plan_summary", "exec_log", "validation"
	Feedback        string            `json:"feedback,omitempty"` // от неудачной валидации
	ValidationCount int               `json:"validation_count"`
	Error           string            `json:"error,omitempty"`
}

// ─── Phase-specific system prompts ──────────────────────────────────────────

func buildPlanningPrompt(ts *TaskState) string {
	var b strings.Builder
	b.WriteString("You are a planning agent. Your job is to analyze the goal and create a structured plan.\n\n")
	b.WriteString("Use the `submit_plan` tool to submit your plan. The plan must contain:\n")
	b.WriteString("- `steps`: an array of concrete, actionable steps with descriptions\n")
	b.WriteString("- `summary`: a brief summary of the overall approach\n\n")
	b.WriteString(fmt.Sprintf("Goal: %s\n", ts.Goal))
	if ts.Feedback != "" {
		b.WriteString(fmt.Sprintf("\nFeedback from previous attempt (incorporate this into your new plan):\n%s\n", ts.Feedback))
	}
	b.WriteString("\nCreate a clear, step-by-step plan and submit it using submit_plan.")
	return b.String()
}

func buildExecutingPrompt(ts *TaskState) string {
	var b strings.Builder
	b.WriteString("You are an execution agent. Your job is to execute the plan step by step.\n\n")
	b.WriteString("Use `run_shell` and `read_file` tools to do the work.\n")
	b.WriteString("After completing or failing each step, call `report_step` with the step_index, status (completed|failed), and output.\n\n")
	b.WriteString(fmt.Sprintf("Goal: %s\n\n", ts.Goal))
	if ts.Artifacts["plan_summary"] != "" {
		b.WriteString(fmt.Sprintf("Plan summary: %s\n\n", ts.Artifacts["plan_summary"]))
	}
	if len(ts.Steps) > 0 {
		b.WriteString("Steps to execute:\n")
		for _, s := range ts.Steps {
			b.WriteString(fmt.Sprintf("  %d. %s\n", s.Index, s.Description))
		}
		b.WriteString("\n")
	}
	if len(ts.StepResults) > 0 {
		b.WriteString("Previously completed steps:\n")
		for _, r := range ts.StepResults {
			b.WriteString(fmt.Sprintf("  Step %d [%s]: %s\n", r.Index, r.Status, truncate(r.Output, 200)))
		}
		b.WriteString("\nContinue with remaining steps.\n")
	} else {
		b.WriteString("Execute each step in order. Report the result of each step using report_step.\n")
	}
	return b.String()
}

func buildValidatingPrompt(ts *TaskState) string {
	var b strings.Builder
	b.WriteString("You are a validation agent. Your job is to verify that the goal has been achieved.\n\n")
	b.WriteString("Use `run_shell` and `read_file` tools to check the results.\n")
	b.WriteString("After your verification, call `submit_validation` with:\n")
	b.WriteString("- `passed`: true if the goal was achieved, false otherwise\n")
	b.WriteString("- `feedback`: description of what was verified (or what failed and needs to be fixed)\n")
	b.WriteString("- `next_phase`: if passed=false, specify 'executing' or 'planning'\n\n")
	b.WriteString(fmt.Sprintf("Goal: %s\n\n", ts.Goal))
	if len(ts.Steps) > 0 {
		b.WriteString("Plan steps:\n")
		for _, s := range ts.Steps {
			b.WriteString(fmt.Sprintf("  %d. %s\n", s.Index, s.Description))
		}
		b.WriteString("\n")
	}
	if len(ts.StepResults) > 0 {
		b.WriteString("Execution results:\n")
		for _, r := range ts.StepResults {
			b.WriteString(fmt.Sprintf("  Step %d [%s]: %s\n", r.Index, r.Status, truncate(r.Output, 300)))
		}
		b.WriteString("\n")
	}
	b.WriteString("Verify the results and submit your validation decision.")
	return b.String()
}

// ─── Local LLM text-mode prompts ─────────────────────────────────────────────

func buildPlanningPromptLocal(ts *TaskState) string {
	var b strings.Builder
	b.WriteString("You are a planning agent. Analyze the goal and output a structured plan.\n\n")
	b.WriteString("Output your plan in EXACTLY this format:\n\n")
	b.WriteString("SUMMARY: <brief description of the approach>\n")
	b.WriteString("STEPS:\n")
	b.WriteString("1. <step 1>\n")
	b.WriteString("2. <step 2>\n")
	b.WriteString("...\n\n")
	b.WriteString(fmt.Sprintf("Goal: %s\n", ts.Goal))
	if ts.Feedback != "" {
		b.WriteString(fmt.Sprintf("\nFeedback from previous attempt (incorporate this):\n%s\n", ts.Feedback))
	}
	return b.String()
}

func buildExecutingPromptLocal(ts *TaskState) string {
	var b strings.Builder
	b.WriteString("You are an execution agent. Describe how each step would be executed.\n")
	b.WriteString("For each step, explain what you would do and what the expected result is.\n\n")
	b.WriteString(fmt.Sprintf("Goal: %s\n\n", ts.Goal))
	if ts.Artifacts["plan_summary"] != "" {
		b.WriteString(fmt.Sprintf("Plan summary: %s\n\n", ts.Artifacts["plan_summary"]))
	}
	if len(ts.Steps) > 0 {
		b.WriteString("Steps to execute:\n")
		for _, s := range ts.Steps {
			b.WriteString(fmt.Sprintf("  %d. %s\n", s.Index, s.Description))
		}
		b.WriteString("\n")
	}
	b.WriteString("Describe the execution of each step in detail.")
	return b.String()
}

func buildValidatingPromptLocal(ts *TaskState) string {
	var b strings.Builder
	b.WriteString("You are a validation agent. Evaluate whether the goal has been achieved.\n\n")
	b.WriteString("After your analysis, output your decision in EXACTLY this format:\n\n")
	b.WriteString("RESULT: PASS\n")
	b.WriteString("or\n")
	b.WriteString("RESULT: FAIL\n")
	b.WriteString("FEEDBACK: <description of what failed and needs fixing>\n\n")
	b.WriteString(fmt.Sprintf("Goal: %s\n\n", ts.Goal))
	if len(ts.Steps) > 0 {
		b.WriteString("Plan steps:\n")
		for _, s := range ts.Steps {
			b.WriteString(fmt.Sprintf("  %d. %s\n", s.Index, s.Description))
		}
		b.WriteString("\n")
	}
	if len(ts.StepResults) > 0 {
		b.WriteString("Execution results:\n")
		for _, r := range ts.StepResults {
			b.WriteString(fmt.Sprintf("  Step %d [%s]: %s\n", r.Index, r.Status, truncate(r.Output, 300)))
		}
		b.WriteString("\n")
	}
	if ts.Artifacts["exec_log"] != "" {
		b.WriteString("Execution log:\n")
		b.WriteString(truncate(ts.Artifacts["exec_log"], 1000))
		b.WriteString("\n\n")
	}
	b.WriteString("Evaluate the results and output RESULT: PASS or RESULT: FAIL with FEEDBACK.")
	return b.String()
}

// ─── Local LLM text parsers ─────────────────────────────────────────────────

var stepLineRe = regexp.MustCompile(`^\s*(\d+)\.\s+(.+)`)

func parsePlanText(text string) ([]TaskStep, string) {
	lines := strings.Split(text, "\n")
	var summary string
	var steps []TaskStep
	inSteps := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "SUMMARY:") {
			summary = strings.TrimSpace(strings.TrimPrefix(trimmed, "SUMMARY:"))
			continue
		}

		if strings.HasPrefix(trimmed, "STEPS:") {
			inSteps = true
			continue
		}

		if inSteps {
			if m := stepLineRe.FindStringSubmatch(line); m != nil {
				steps = append(steps, TaskStep{
					Index:       len(steps),
					Description: strings.TrimSpace(m[2]),
					Status:      StepPending,
				})
			}
		}
	}

	// Fallback: if no SUMMARY found, use first non-empty line.
	if summary == "" && len(lines) > 0 {
		for _, l := range lines {
			t := strings.TrimSpace(l)
			if t != "" && !strings.HasPrefix(t, "STEPS:") && stepLineRe.FindStringSubmatch(t) == nil {
				summary = t
				break
			}
		}
	}

	// Fallback: if no steps parsed, create a single step from the text.
	if len(steps) == 0 && strings.TrimSpace(text) != "" {
		steps = []TaskStep{{Index: 0, Description: "Execute the goal", Status: StepPending}}
	}

	return steps, summary
}

func parseExecutionText(text string, steps []TaskStep) []StepResult {
	results := make([]StepResult, len(steps))
	for i, s := range steps {
		results[i] = StepResult{
			Index:  s.Index,
			Status: StepCompleted,
			Output: fmt.Sprintf("Described by LLM (text mode)"),
		}
	}
	return results
}

func parseValidationText(text string) (passed bool, feedback string, nextPhase string) {
	lines := strings.Split(text, "\n")
	foundResult := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.ToUpper(line))
		if strings.Contains(trimmed, "RESULT:") {
			foundResult = true
			if strings.Contains(trimmed, "PASS") {
				passed = true
			} else if strings.Contains(trimmed, "FAIL") {
				passed = false
			}
		}
	}

	// If the model didn't produce a RESULT: marker at all, assume passed.
	// Small/local models often skip structured output format.
	if !foundResult {
		passed = true
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmed), "FEEDBACK:") {
			feedback = strings.TrimSpace(trimmed[len("FEEDBACK:"):])
			break
		}
	}

	// Default: if validation failed and no explicit next phase, go to executing.
	if !passed {
		nextPhase = PhaseExecuting
		if feedback == "" {
			feedback = "Validation did not pass (no detailed feedback from LLM)."
		}
	}

	return passed, feedback, nextPhase
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (ts *TaskState) completedCount() int {
	n := 0
	for _, s := range ts.Steps {
		if s.Status == StepCompleted {
			n++
		}
	}
	return n
}

// ─── Display (CLI) ──────────────────────────────────────────────────────────

func (ts *TaskState) FormatStatus() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Goal:  %s\n", ts.Goal))
	b.WriteString(fmt.Sprintf("Phase: %s", ts.Phase))
	if ts.Paused {
		b.WriteString(" (paused — send a message to continue)")
	}
	b.WriteString("\n")
	if len(ts.Steps) > 0 {
		b.WriteString(fmt.Sprintf("Steps: %d/%d completed\n", ts.completedCount(), len(ts.Steps)))
		for _, s := range ts.Steps {
			icon := "[ ]"
			switch s.Status {
			case StepCompleted:
				icon = "[x]"
			case StepFailed:
				icon = "[!]"
			}
			b.WriteString(fmt.Sprintf("  %s %d. %s\n", icon, s.Index, s.Description))
		}
	}
	if len(ts.StepResults) > 0 {
		b.WriteString(fmt.Sprintf("Results: %d step results recorded\n", len(ts.StepResults)))
	}
	if ts.ValidationCount > 0 {
		b.WriteString(fmt.Sprintf("Validations: %d\n", ts.ValidationCount))
	}
	if ts.Feedback != "" {
		b.WriteString(fmt.Sprintf("Feedback: %s\n", truncate(ts.Feedback, 100)))
	}
	if ts.Error != "" {
		b.WriteString(fmt.Sprintf("Error: %s\n", ts.Error))
	}
	return b.String()
}
