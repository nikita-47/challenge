package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ─── Task phases and step statuses ──────────────────────────────────────────

const (
	PhasePlanning   = "planning"
	PhaseExecuting  = "executing"
	PhaseValidating = "validating"
	PhaseDone       = "done"
	PhasePaused     = "paused"
)

const (
	StepPending    = "pending"
	StepInProgress = "in_progress"
	StepCompleted  = "completed"
	StepFailed     = "failed"
)

// ─── Data types ─────────────────────────────────────────────────────────────

type TaskStep struct {
	Index       int    `json:"index"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Result      string `json:"result,omitempty"`
}

type TaskState struct {
	Goal           string     `json:"goal"`
	Phase          string     `json:"phase"`
	Steps          []TaskStep `json:"steps"`
	CurrentStep    int        `json:"current_step"`
	ExpectedAction string     `json:"expected_action,omitempty"`
	PausedAtPhase  string     `json:"paused_at_phase,omitempty"`
	Error          string     `json:"error,omitempty"`
}

// ─── Transition validation ──────────────────────────────────────────────────

var validTransitions = map[string][]string{
	PhasePlanning:   {PhaseExecuting, PhasePaused},
	PhaseExecuting:  {PhaseValidating, PhasePaused},
	PhaseValidating: {PhaseDone, PhaseExecuting, PhasePaused},
	PhasePaused:     {PhasePlanning, PhaseExecuting, PhaseValidating},
}

func (ts *TaskState) canTransition(to string) bool {
	allowed, ok := validTransitions[ts.Phase]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == to {
			return true
		}
	}
	return false
}

// ─── Task state update actions ──────────────────────────────────────────────

type taskStateAction struct {
	Action      string     `json:"action"`
	Steps       []TaskStep `json:"steps,omitempty"`
	StepIndex   int        `json:"step_index,omitempty"`
	Result      string     `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
	Expected    string     `json:"expected_action,omitempty"`
}

func (ts *TaskState) applyAction(raw json.RawMessage) (string, error) {
	var act taskStateAction
	if err := json.Unmarshal(raw, &act); err != nil {
		return "", fmt.Errorf("invalid action input: %w", err)
	}

	switch act.Action {
	case "set_plan":
		if ts.Phase != PhasePlanning {
			return "", fmt.Errorf("set_plan only allowed in planning phase, current: %s", ts.Phase)
		}
		if len(act.Steps) == 0 {
			return "", fmt.Errorf("set_plan requires at least one step")
		}
		ts.Steps = make([]TaskStep, len(act.Steps))
		for i, s := range act.Steps {
			ts.Steps[i] = TaskStep{
				Index:       i,
				Description: s.Description,
				Status:      StepPending,
			}
		}
		ts.CurrentStep = 0
		ts.Phase = PhaseExecuting
		ts.ExpectedAction = act.Expected
		return fmt.Sprintf("Plan set with %d steps. Transitioned to executing phase.", len(ts.Steps)), nil

	case "start_step":
		if ts.Phase != PhaseExecuting {
			return "", fmt.Errorf("start_step only allowed in executing phase, current: %s", ts.Phase)
		}
		idx := act.StepIndex
		if idx < 0 || idx >= len(ts.Steps) {
			return "", fmt.Errorf("step index %d out of range (0-%d)", idx, len(ts.Steps)-1)
		}
		ts.Steps[idx].Status = StepInProgress
		ts.CurrentStep = idx
		ts.ExpectedAction = act.Expected
		return fmt.Sprintf("Started step %d: %s", idx, ts.Steps[idx].Description), nil

	case "complete_step":
		if ts.Phase != PhaseExecuting {
			return "", fmt.Errorf("complete_step only allowed in executing phase, current: %s", ts.Phase)
		}
		idx := act.StepIndex
		if idx < 0 || idx >= len(ts.Steps) {
			return "", fmt.Errorf("step index %d out of range (0-%d)", idx, len(ts.Steps)-1)
		}
		ts.Steps[idx].Status = StepCompleted
		ts.Steps[idx].Result = act.Result
		ts.ExpectedAction = act.Expected
		return fmt.Sprintf("Completed step %d: %s", idx, ts.Steps[idx].Description), nil

	case "fail_step":
		if ts.Phase != PhaseExecuting {
			return "", fmt.Errorf("fail_step only allowed in executing phase, current: %s", ts.Phase)
		}
		idx := act.StepIndex
		if idx < 0 || idx >= len(ts.Steps) {
			return "", fmt.Errorf("step index %d out of range (0-%d)", idx, len(ts.Steps)-1)
		}
		ts.Steps[idx].Status = StepFailed
		ts.Steps[idx].Result = act.Error
		ts.Error = act.Error
		ts.ExpectedAction = act.Expected
		return fmt.Sprintf("Step %d failed: %s", idx, act.Error), nil

	case "validate":
		if !ts.canTransition(PhaseValidating) {
			return "", fmt.Errorf("cannot transition to validating from %s", ts.Phase)
		}
		ts.Phase = PhaseValidating
		ts.ExpectedAction = act.Expected
		return "Transitioned to validating phase. Review results and confirm completion.", nil

	case "done":
		if !ts.canTransition(PhaseDone) {
			return "", fmt.Errorf("cannot transition to done from %s", ts.Phase)
		}
		ts.Phase = PhaseDone
		ts.ExpectedAction = ""
		ts.Error = ""
		return "Task completed successfully.", nil

	case "pause":
		if !ts.canTransition(PhasePaused) {
			return "", fmt.Errorf("cannot pause from %s", ts.Phase)
		}
		ts.PausedAtPhase = ts.Phase
		ts.Phase = PhasePaused
		ts.ExpectedAction = ""
		return fmt.Sprintf("Task paused at %s phase. Use resume to continue.", ts.PausedAtPhase), nil

	case "resume":
		if ts.Phase != PhasePaused {
			return "", fmt.Errorf("resume only allowed in paused phase, current: %s", ts.Phase)
		}
		if ts.PausedAtPhase == "" {
			ts.PausedAtPhase = PhaseExecuting
		}
		ts.Phase = ts.PausedAtPhase
		ts.PausedAtPhase = ""
		return fmt.Sprintf("Resumed task at %s phase.", ts.Phase), nil

	default:
		return "", fmt.Errorf("unknown action: %s", act.Action)
	}
}

// ─── System prompt section ──────────────────────────────────────────────────

func (ts *TaskState) SystemPromptSection() string {
	var b strings.Builder
	b.WriteString("\n\n## Task State Machine\n\n")
	b.WriteString("You are operating in TASK MODE with a structured state machine.\n")
	b.WriteString("Use the `update_task_state` tool to manage task phases and steps.\n\n")
	b.WriteString("### FSM Rules\n")
	b.WriteString("1. Start in `planning` phase. Call `set_plan` with a list of concrete steps.\n")
	b.WriteString("2. After planning, transition to `executing`. Call `start_step` before working on each step.\n")
	b.WriteString("3. After executing each step (using run_shell/read_file), call `complete_step` or `fail_step`.\n")
	b.WriteString("4. When all steps are done, call `validate` to enter validation phase.\n")
	b.WriteString("5. In validation, verify results. If OK, call `done`. If not, transition back to `executing`.\n")
	b.WriteString("6. You can call `pause` at any phase to save state for later resume.\n\n")
	b.WriteString("### Actions\n")
	b.WriteString("- `set_plan` — set steps array (planning only)\n")
	b.WriteString("- `start_step` — mark step as in_progress\n")
	b.WriteString("- `complete_step` — mark step as completed with result\n")
	b.WriteString("- `fail_step` — mark step as failed with error\n")
	b.WriteString("- `validate` — transition to validation phase\n")
	b.WriteString("- `done` — mark task as completed\n")
	b.WriteString("- `pause` — pause task for later resume\n\n")

	b.WriteString("### Current State\n")
	b.WriteString(fmt.Sprintf("- **Goal**: %s\n", ts.Goal))
	b.WriteString(fmt.Sprintf("- **Phase**: %s\n", ts.Phase))
	if len(ts.Steps) > 0 {
		b.WriteString(fmt.Sprintf("- **Progress**: step %d/%d\n", ts.completedCount(), len(ts.Steps)))
		b.WriteString("- **Steps**:\n")
		for _, s := range ts.Steps {
			icon := "[ ]"
			switch s.Status {
			case StepInProgress:
				icon = "[>]"
			case StepCompleted:
				icon = "[x]"
			case StepFailed:
				icon = "[!]"
			}
			line := fmt.Sprintf("  %s %d. %s", icon, s.Index, s.Description)
			if s.Result != "" {
				line += fmt.Sprintf(" — %s", s.Result)
			}
			b.WriteString(line + "\n")
		}
	}
	if ts.ExpectedAction != "" {
		b.WriteString(fmt.Sprintf("- **Expected next action**: %s\n", ts.ExpectedAction))
	}
	if ts.Error != "" {
		b.WriteString(fmt.Sprintf("- **Error**: %s\n", ts.Error))
	}

	return b.String()
}

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
	b.WriteString(fmt.Sprintf("Phase: %s\n", ts.Phase))
	if len(ts.Steps) > 0 {
		b.WriteString(fmt.Sprintf("Steps: %d/%d completed\n", ts.completedCount(), len(ts.Steps)))
		for _, s := range ts.Steps {
			icon := "[ ]"
			switch s.Status {
			case StepInProgress:
				icon = "[>]"
			case StepCompleted:
				icon = "[x]"
			case StepFailed:
				icon = "[!]"
			}
			line := fmt.Sprintf("  %s %d. %s", icon, s.Index, s.Description)
			if s.Result != "" {
				line += fmt.Sprintf(" — %s", truncate(s.Result, 80))
			}
			b.WriteString(line + "\n")
		}
	}
	if ts.ExpectedAction != "" {
		b.WriteString(fmt.Sprintf("Next:  %s\n", ts.ExpectedAction))
	}
	if ts.Error != "" {
		b.WriteString(fmt.Sprintf("Error: %s\n", ts.Error))
	}
	return b.String()
}
