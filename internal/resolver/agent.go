package resolver

import "github.com/kryptamine/herdr-auto-title/internal/state"

// agentField is a source whose reading is one field a pane's agent reports
// about itself. Every such source shares one gate — a pane without an agent
// reports nothing, whatever a leftover field still says — and one reading of
// the field, so the gate and the reading live here rather than in each source.
type agentField struct {
	name       string
	confidence int
	field      func(*state.PaneState) string
}

var _ Source = agentField{}

func (s agentField) Name() string    { return s.name }
func (s agentField) Confidence() int { return s.confidence }

func (s agentField) Resolve(pane *state.PaneState) (Parts, bool) {
	if !pane.HasAgent() {
		return Parts{}, false
	}

	return activityFrom(pane, s.field(pane))
}

// NewAgent derives the activity from what the pane's agent says it is working
// on, which outranks every other source. Many agents leave the field empty and
// fall through to TerminalTitle instead.
func NewAgent() Source {
	return agentField{
		name:       "agent",
		confidence: ConfidenceAgent,
		field:      func(pane *state.PaneState) string { return pane.AgentTitle },
	}
}

// NewTranscript derives the activity from what the agent's own session says it
// is about. It sits below TerminalTitle because an agent that titles its window
// has already said the same thing, sooner; this answers when it has not.
func NewTranscript() Source {
	return agentField{
		name:       "transcript",
		confidence: ConfidenceTranscript,
		field:      func(pane *state.PaneState) string { return pane.AgentTopic },
	}
}
