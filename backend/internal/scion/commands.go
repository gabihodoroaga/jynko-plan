package scion

import (
	"strings"
)

// Supported slash commands in work item comments.
const (
	CmdImplement = "implement"
	CmdReview    = "review"
	CmdFix       = "fix"
	CmdPlan      = "plan"
	CmdAsk       = "ask"
	CmdStop      = "stop"
)

// ParsedCommand represents the result of parsing a work item comment for a leading Scion slash command.
type ParsedCommand struct {
	IsCommand    bool   `json:"is_command"`
	Command      string `json:"command,omitempty"`
	Instructions string `json:"instructions,omitempty"`
	TargetAgent  string `json:"target_agent,omitempty"`
	NewAgent     bool   `json:"new_agent,omitempty"`
	Harness      string `json:"harness,omitempty"`
	RepoURL      string `json:"repo_url,omitempty"`
	BaseBranch   string `json:"base_branch,omitempty"`
	StopAll      bool   `json:"stop_all,omitempty"`
}

var supportedCommands = map[string]bool{
	CmdImplement: true,
	CmdReview:    true,
	CmdFix:       true,
	CmdPlan:      true,
	CmdAsk:       true,
	CmdStop:      true,
}

// ParseSlashCommand inspects a comment body for a leading Scion slash command (/implement, /review, /fix, /plan, /ask, /stop).
// Comments not starting with '/' or starting with an unknown slash token return IsCommand=false.
func ParseSlashCommand(body string) ParsedCommand {
	trimmed := strings.TrimSpace(body)
	if !strings.HasPrefix(trimmed, "/") {
		return ParsedCommand{IsCommand: false}
	}

	// Split into first line and remaining lines so multi-line instructions stay intact.
	firstLine := trimmed
	restLines := ""
	if idx := strings.IndexByte(trimmed, '\n'); idx != -1 {
		firstLine = trimmed[:idx]
		restLines = strings.TrimSpace(trimmed[idx+1:])
	}

	fields := strings.Fields(firstLine)
	if len(fields) == 0 {
		return ParsedCommand{IsCommand: false}
	}

	cmdToken := strings.ToLower(strings.TrimPrefix(fields[0], "/"))
	if !supportedCommands[cmdToken] {
		return ParsedCommand{IsCommand: false}
	}

	out := ParsedCommand{
		IsCommand: true,
		Command:   cmdToken,
	}

	var instructionTokens []string
	args := fields[1:]
	for i := 0; i < len(args); i++ {
		tok := args[i]
		switch {
		case tok == "--new":
			out.NewAgent = true
		case tok == "--all":
			out.StopAll = true
		case strings.HasPrefix(tok, "--agent="):
			out.TargetAgent = strings.TrimSpace(strings.TrimPrefix(tok, "--agent="))
		case tok == "--agent" && i+1 < len(args):
			i++
			out.TargetAgent = strings.TrimSpace(args[i])
		case strings.HasPrefix(tok, "--harness="):
			out.Harness = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(tok, "--harness=")))
		case tok == "--harness" && i+1 < len(args):
			i++
			out.Harness = strings.ToLower(strings.TrimSpace(args[i]))
		case strings.HasPrefix(tok, "--repo="):
			out.RepoURL = strings.TrimSpace(strings.TrimPrefix(tok, "--repo="))
		case tok == "--repo" && i+1 < len(args):
			i++
			out.RepoURL = strings.TrimSpace(args[i])
		case strings.HasPrefix(tok, "--branch="):
			out.BaseBranch = strings.TrimSpace(strings.TrimPrefix(tok, "--branch="))
		case tok == "--branch" && i+1 < len(args):
			i++
			out.BaseBranch = strings.TrimSpace(args[i])
		default:
			instructionTokens = append(instructionTokens, tok)
		}
	}

	inlineInstr := strings.Join(instructionTokens, " ")
	switch {
	case inlineInstr != "" && restLines != "":
		out.Instructions = inlineInstr + "\n" + restLines
	case inlineInstr != "":
		out.Instructions = inlineInstr
	default:
		out.Instructions = restLines
	}

	return out
}
