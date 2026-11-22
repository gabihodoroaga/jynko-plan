package scion

import "testing"

func TestParseSlashCommand(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCmd   bool
		command   string
		instr     string
		agent     string
		newAgent  bool
		harness   string
		repo      string
		branch    string
		stopAll   bool
	}{
		{
			name:    "plain human comment",
			body:    "Looks good to me, let's ship it.",
			wantCmd: false,
		},
		{
			name:    "unknown slash command",
			body:    "/deploy production now",
			wantCmd: false,
		},
		{
			name:    "slash command not at start",
			body:    "Please run /implement later",
			wantCmd: false,
		},
		{
			name:     "implement with inline flags and multiline instructions",
			body:     "/implement --harness=antigravity --repo=https://github.com/org/repo.git --branch=main --new focus on RBAC\nAdd unit tests too.",
			wantCmd:  true,
			command:  CmdImplement,
			harness:  "antigravity",
			repo:     "https://github.com/org/repo.git",
			branch:   "main",
			newAgent: true,
			instr:    "focus on RBAC\nAdd unit tests too.",
		},
		{
			name:    "review with space-separated --agent flag",
			body:    "/review --agent agent-rev-1 check error handling",
			wantCmd: true,
			command: CmdReview,
			agent:   "agent-rev-1",
			instr:   "check error handling",
		},
		{
			name:    "stop --all",
			body:    "/stop --all",
			wantCmd: true,
			command: CmdStop,
			stopAll: true,
		},
		{
			name:    "case-insensitive /ASK",
			body:    "  /ASK what is the status of the PR?  ",
			wantCmd: true,
			command: CmdAsk,
			instr:   "what is the status of the PR?",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseSlashCommand(tc.body)
			if got.IsCommand != tc.wantCmd {
				t.Fatalf("IsCommand = %v, want %v", got.IsCommand, tc.wantCmd)
			}
			if !tc.wantCmd {
				return
			}
			if got.Command != tc.command {
				t.Errorf("Command = %q, want %q", got.Command, tc.command)
			}
			if got.Instructions != tc.instr {
				t.Errorf("Instructions = %q, want %q", got.Instructions, tc.instr)
			}
			if got.TargetAgent != tc.agent {
				t.Errorf("TargetAgent = %q, want %q", got.TargetAgent, tc.agent)
			}
			if got.NewAgent != tc.newAgent {
				t.Errorf("NewAgent = %v, want %v", got.NewAgent, tc.newAgent)
			}
			if got.Harness != tc.harness {
				t.Errorf("Harness = %q, want %q", got.Harness, tc.harness)
			}
			if got.RepoURL != tc.repo {
				t.Errorf("RepoURL = %q, want %q", got.RepoURL, tc.repo)
			}
			if got.BaseBranch != tc.branch {
				t.Errorf("BaseBranch = %q, want %q", got.BaseBranch, tc.branch)
			}
			if got.StopAll != tc.stopAll {
				t.Errorf("StopAll = %v, want %v", got.StopAll, tc.stopAll)
			}
		})
	}
}
