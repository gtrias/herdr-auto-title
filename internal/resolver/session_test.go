package resolver

import (
	"testing"

	"github.com/kryptamine/herdr-auto-title/internal/herdr"
	"github.com/kryptamine/herdr-auto-title/internal/state"
)

func TestASessionTitleOutranksTheFirstPromptHerdrReports(t *testing.T) {
	t.Parallel()

	// Herdr fills an omp pane's title with its first prompt, and the terminal
	// title carries a spinner; the session file says what the session is now.
	got := defaultChain().Resolve(tabWithPane(&state.PaneState{
		Dir:           dashboard,
		TerminalTitle: "π ⠴ Sync herdr tab titles",
		Agent:         "omp",
		AgentStatus:   herdr.AgentStatusWorking,
		AgentTitle:    "I want omp, pi, claude sessions to",
		SessionTitle:  "Sync herdr tab titles",
	}))

	if want := "dashboard › omp › Sync herdr tab titles"; got.Name != want {
		t.Errorf("name = %q, want %q", got.Name, want)
	}

	if got.Reason != "session" {
		t.Errorf("reason = %q, want session", got.Reason)
	}
}

func TestAnUntitledSessionFallsThroughToTheAgentTitle(t *testing.T) {
	t.Parallel()

	got := defaultChain().Resolve(tabWithPane(&state.PaneState{
		Dir:         dashboard,
		Agent:       "omp",
		AgentStatus: herdr.AgentStatusWorking,
		AgentTitle:  "Fix OAuth redirect",
	}))

	if got.Reason != "agent" {
		t.Errorf("reason = %q, want agent", got.Reason)
	}
}
