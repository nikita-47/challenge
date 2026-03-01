package main

import (
	"fmt"
	"time"
)

// branch represents a dialog branch forked from the main message history.
type branch struct {
	Name      string    `json:"name"`
	ForkIndex int       `json:"fork_index"` // index in cw.Messages where this branch was created
	Messages  []message `json:"messages"`   // messages after the fork point
	CreatedAt time.Time `json:"created_at"`
}

// createBranch creates a new branch from the current position.
func createBranch(cw *contextWindow, name string) error {
	if name == "" || name == "main" {
		return fmt.Errorf("invalid branch name %q", name)
	}
	for _, b := range cw.Branches {
		if b.Name == name {
			return fmt.Errorf("branch %q already exists", name)
		}
	}

	forkIndex := len(cw.Messages)

	// If currently on a branch, fork from the branch's view of messages.
	if cw.ActiveBranch != "" && cw.ActiveBranch != "main" {
		for _, b := range cw.Branches {
			if b.Name == cw.ActiveBranch {
				forkIndex = b.ForkIndex
				break
			}
		}
	}

	// Copy current branch messages (after fork) into the new branch.
	var branchMsgs []message
	if cw.ActiveBranch != "" && cw.ActiveBranch != "main" {
		for _, b := range cw.Branches {
			if b.Name == cw.ActiveBranch {
				branchMsgs = make([]message, len(b.Messages))
				copy(branchMsgs, b.Messages)
				break
			}
		}
	} else if forkIndex < len(cw.Messages) {
		// Forking from main with some messages beyond fork point — shouldn't happen
		// since forkIndex = len(cw.Messages), but handle defensively.
		branchMsgs = make([]message, len(cw.Messages)-forkIndex)
		copy(branchMsgs, cw.Messages[forkIndex:])
	}

	cw.Branches = append(cw.Branches, branch{
		Name:      name,
		ForkIndex: forkIndex,
		Messages:  branchMsgs,
		CreatedAt: time.Now().UTC(),
	})
	cw.ActiveBranch = name
	return nil
}

// switchBranch switches to an existing branch.
func switchBranch(cw *contextWindow, name string) error {
	if name == "main" {
		cw.ActiveBranch = "main"
		return nil
	}
	for _, b := range cw.Branches {
		if b.Name == name {
			cw.ActiveBranch = name
			return nil
		}
	}
	return fmt.Errorf("branch %q not found", name)
}

// listBranches returns all branch names including "main".
func listBranches(cw *contextWindow) []string {
	names := []string{"main"}
	for _, b := range cw.Branches {
		names = append(names, b.Name)
	}
	return names
}

// deleteBranch removes a branch by name (cannot delete "main").
func deleteBranch(cw *contextWindow, name string) error {
	if name == "main" {
		return fmt.Errorf("cannot delete main branch")
	}
	for i, b := range cw.Branches {
		if b.Name == name {
			cw.Branches = append(cw.Branches[:i], cw.Branches[i+1:]...)
			if cw.ActiveBranch == name {
				cw.ActiveBranch = "main"
			}
			return nil
		}
	}
	return fmt.Errorf("branch %q not found", name)
}

// buildBranchMessages returns filtered messages for the active branch.
// Uses the default summary strategy within the branch context.
func buildBranchMessages(cw *contextWindow) []message {
	msgs := filterAPIMessages(activeMessages(cw))
	return msgs
}
