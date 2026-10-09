package session

import (
	"os"
	"path/filepath"
	"testing"
)

func appendTo(t *testing.T, path, text string) {
	t.Helper()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.WriteString(text); err != nil {
		t.Fatal(err)
	}

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestOmpHeaderNamesASessionAndTheLastRenameWins(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "omp.jsonl")
	appendTo(t, path, `{"type":"title","v":1,"title":"🗂️ HERDR: First name","source":"auto","pad":"   "}`+"\n"+
		`{"type":"session","id":"x","cwd":"/tmp"}`+"\n"+
		`{"type":"message","message":{"role":"user","content":"I want tab titles"}}`+"\n")

	r := NewReader()
	if got, want := r.Title(path), "🗂️ HERDR: First name"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}

	appendTo(t, path, `{"type":"title_change","id":"a","title":"Second name","source":"auto"}`+"\n")
	if got, want := r.Title(path), "Second name"; got != want {
		t.Fatalf("after rename title = %q, want %q", got, want)
	}

	// A line still being written is not read until it is whole.
	appendTo(t, path, `{"type":"title_change","title":"Third`)
	if got, want := r.Title(path), "Second name"; got != want {
		t.Fatalf("mid-write title = %q, want %q", got, want)
	}

	appendTo(t, path, ` name"}`+"\n")
	if got, want := r.Title(path), "Third name"; got != want {
		t.Fatalf("completed title = %q, want %q", got, want)
	}
}

func TestPiSessionInfoNamesASession(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "pi.jsonl")
	appendTo(t, path, `{"type":"session","version":3,"id":"x","cwd":"/tmp"}`+"\n"+
		`{"type":"session_info","id":"a","name":"Define a skill"}`+"\n")

	if got, want := NewReader().Title(path), "Define a skill"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
}

func TestAnUntitledOrMissingSessionHasNoTitle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "untitled.jsonl")
	// A message that merely mentions a rename is conversation, not a rename.
	appendTo(t, path, `{"type":"session","id":"x"}`+"\n"+
		`{"type":"message","message":{"content":"the \"title_change\" record"}}`+"\n")

	r := NewReader()
	if got := r.Title(path); got != "" {
		t.Fatalf("untitled = %q, want empty", got)
	}

	if got := r.Title(filepath.Join(dir, "missing.jsonl")); got != "" {
		t.Fatalf("missing = %q, want empty", got)
	}
}
