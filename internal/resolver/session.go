package resolver

import "github.com/kryptamine/herdr-auto-title/internal/state"

// Session derives the activity from the title the agent gave its own session,
// read from its session file. It outranks Agent because Herdr fills an omp or
// pi pane's title with the first prompt, while the session title is the one
// the agent keeps current.
type Session struct{}

var _ Source = Session{}

func NewSession() Session { return Session{} }

func (Session) Name() string    { return "session" }
func (Session) Confidence() int { return ConfidenceSession }

func (Session) Resolve(pane *state.PaneState) (Parts, bool) {
	if !pane.HasAgent() {
		return Parts{}, false
	}

	return activityFrom(pane, pane.SessionTitle)
}
