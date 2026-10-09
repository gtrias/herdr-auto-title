// Package session reads the title an agent gives its own session from the
// session file its Herdr integration points at. omp and pi report that file by
// path, and both keep the current name in it as an append-only log:
//
//	omp: {"type":"title","title":…} heads the file, {"type":"title_change","title":…} renames it
//	pi:  {"type":"session_info","name":…} names and renames it
//
// The terminal title says the same thing, but behind a status glyph and a
// spinner frame that change several times a second.
package session

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"
)

// Agents are the labels Herdr gives the agents whose session files this
// package understands.
var Agents = []string{"omp", "pi"}

// maxScan bounds one read. The file is append-only, so this bounds the
// catch-up on a session first seen mid-flight, not the steady state. A rename
// buried further back than this is not seen, which a first prompt then covers.
const maxScan = 4 << 20

// maxHead bounds the read of a file's first line, where omp writes its title
// header and pads it so the header can be rewritten in place.
const maxHead = 64 << 10

// Reader reads session titles, remembering how far it has read each file.
type Reader struct {
	mu    sync.Mutex
	files map[string]*file
}

type file struct {
	offset int64
	title  string
	// head is the title omp keeps on the first line, rewritten in place rather
	// than appended to, so it is reread whenever the file grows.
	head string
}

func NewReader() *Reader {
	return &Reader{files: make(map[string]*file)}
}

// Title reports the session's current title, or "" when the file names none
// or cannot be read.
func (r *Reader) Title(path string) string {
	if path == "" {
		return ""
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	f, known := r.files[path]
	if !known {
		f = &file{}
		r.files[path] = f
	}

	f.read(path)

	if f.title != "" {
		return f.title
	}

	return f.head
}

// Retain forgets every file but these, so a pane that closed takes its
// session with it.
func (r *Reader) Retain(paths []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	kept := make(map[string]*file, len(paths))
	for _, path := range paths {
		if f, known := r.files[path]; known {
			kept[path] = f
		}
	}

	r.files = kept
}

func (f *file) read(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	size := info.Size()
	switch {
	case size == f.offset:
		return
	case size < f.offset:
		*f = file{}
	}

	handle, err := os.Open(path)
	if err != nil {
		return
	}
	// A read-only file has nothing to report on close.
	defer func() { _ = handle.Close() }()

	f.head = headTitle(handle)

	from, fromTail := f.offset, size-f.offset > maxScan
	if fromTail {
		from = size - maxScan
	}

	if _, err := handle.Seek(from, io.SeekStart); err != nil {
		return
	}

	content, err := io.ReadAll(io.LimitReader(handle, maxScan))
	if err != nil {
		return
	}

	// Only whole lines: the agent may be writing one while this runs.
	end := bytes.LastIndexByte(content, '\n')
	if end < 0 {
		return
	}

	f.offset = from + int64(end) + 1

	complete := string(content[:end])
	if fromTail {
		// A tail read starts mid-line, and that fragment is not JSON.
		_, complete, _ = strings.Cut(complete, "\n")
	}

	f.absorb(complete)
}

// entry is the part of a session line that can name the session.
type entry struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Name  string `json:"name"`
}

// absorb takes the last rename in lines; a session is renamed as it goes.
func (f *file) absorb(lines string) {
	for line := range strings.SplitSeq(lines, "\n") {
		if title, ok := renamed(line); ok {
			f.title = title
		}
	}
}

func renamed(line string) (string, bool) {
	// Most lines are conversation, and decoding each one would cost a poll far
	// more than the title is worth.
	if !strings.Contains(line, `"title_change"`) && !strings.Contains(line, `"session_info"`) {
		return "", false
	}

	var read entry
	if json.Unmarshal([]byte(line), &read) != nil {
		return "", false
	}

	switch read.Type {
	case "title_change":
		return clean(read.Title)
	case "session_info":
		return clean(read.Name)
	}

	return "", false
}

// headTitle reads omp's title header, the file's first line.
func headTitle(handle *os.File) string {
	buf := make([]byte, maxHead)

	n, err := handle.ReadAt(buf, 0)
	if err != nil && err != io.EOF {
		return ""
	}

	line, _, _ := bytes.Cut(buf[:n], []byte{'\n'})

	var read entry
	if json.Unmarshal(line, &read) != nil || read.Type != "title" {
		return ""
	}

	title, _ := clean(read.Title)

	return title
}

func clean(title string) (string, bool) {
	title = strings.TrimSpace(title)

	return title, title != ""
}
