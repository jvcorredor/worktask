// Package task is the in-memory model and on-disk codec for a single task
// file.
//
// On-disk format. A task file is YAML-style frontmatter delimited by lines
// containing only "---", followed by a free-form markdown body. The
// frontmatter carries `id`, `created`, optional `completed`, optional
// `last_researched`, and optional `last_research_log`; all timestamps are
// RFC 3339. The body is everything after the trailing delimiter and is
// preserved verbatim. The user-visible specification of this format,
// including which fields are stable and how they interact with the CLI,
// lives on the docs site at
// <https://jvcorredor.github.io/bytheway/storage/>; this package
// implements that contract and does not duplicate it.
//
// [Encode] and [Decode] are the only entry points; the rest of the codec
// is internal.
package task

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/jvcorredor/bytheway/internal/tag"
)

// Task is the in-memory representation of one task file. The zero value
// of Completed and LastResearched means "not set" and is encoded by
// omitting the corresponding frontmatter field.
type Task struct {
	ID              string
	Created         time.Time
	Completed       time.Time
	LastResearched  time.Time
	LastResearchLog string
	Tags            []string
	Body            string
}

const delim = "---\n"

// Encode serialises t as the bytes of a task file. The output is the
// frontmatter described in the package documentation followed by t.Body
// verbatim. Zero-valued Completed and LastResearched fields are omitted
// from the frontmatter; an empty LastResearchLog is omitted as well.
// An empty Tags slice is omitted; a non-empty Tags slice is written as
// a YAML flow-style list. Encode never returns an error in the current
// implementation, but the signature reserves room for future
// format-validation failures.
func Encode(t Task) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(delim)
	fmt.Fprintf(&b, "id: %s\n", t.ID)
	fmt.Fprintf(&b, "created: %s\n", t.Created.Format(time.RFC3339))
	if !t.Completed.IsZero() {
		fmt.Fprintf(&b, "completed: %s\n", t.Completed.Format(time.RFC3339))
	}
	if !t.LastResearched.IsZero() {
		fmt.Fprintf(&b, "last_researched: %s\n", t.LastResearched.Format(time.RFC3339))
	}
	if t.LastResearchLog != "" {
		fmt.Fprintf(&b, "last_research_log: %s\n", t.LastResearchLog)
	}
	if len(t.Tags) > 0 {
		fmt.Fprintf(&b, "tags: [%s]\n", strings.Join(t.Tags, ", "))
	}
	b.WriteString(delim)
	b.WriteString(t.Body)
	return b.Bytes(), nil
}

// Decode parses the bytes of a task file into a [Task]. It is the
// inverse of [Encode] for any value Encode produces. Decode requires the
// leading and trailing "---" frontmatter delimiters and rejects malformed
// frontmatter lines or unparseable RFC 3339 timestamps. Unknown
// frontmatter keys are tolerated and ignored so older binaries can read
// task files written by newer ones. The body is returned verbatim,
// without trimming.
func Decode(data []byte) (Task, error) {
	s := string(data)
	if !strings.HasPrefix(s, delim) {
		return Task{}, fmt.Errorf("task: missing leading frontmatter delimiter")
	}
	rest := s[len(delim):]
	end := strings.Index(rest, delim)
	if end < 0 {
		return Task{}, fmt.Errorf("task: missing trailing frontmatter delimiter")
	}
	header := rest[:end]
	body := rest[end+len(delim):]

	var t Task
	for _, line := range strings.Split(strings.TrimRight(header, "\n"), "\n") {
		key, value, ok := strings.Cut(line, ": ")
		if !ok {
			return Task{}, fmt.Errorf("task: malformed frontmatter line %q", line)
		}
		switch key {
		case "id":
			t.ID = value
		case "created":
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return Task{}, fmt.Errorf("task: parse created: %w", err)
			}
			t.Created = parsed
		case "completed":
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return Task{}, fmt.Errorf("task: parse completed: %w", err)
			}
			t.Completed = parsed
		case "last_researched":
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return Task{}, fmt.Errorf("task: parse last_researched: %w", err)
			}
			t.LastResearched = parsed
		case "last_research_log":
			t.LastResearchLog = value
		case "tags":
			t.Tags = parseTags(value)
		}
	}
	t.Body = body
	return t, nil
}

func parseTags(value string) []string {
	value = strings.Trim(value, "[]")
	if value == "" {
		return nil
	}
	raw := strings.Split(value, ",")
	var out []string
	for _, r := range raw {
		n := tag.Normalize(strings.TrimSpace(r))
		if tag.Validate(n) != nil {
			continue
		}
		out = append(out, n)
	}
	return tag.Deduplicate(out)
}
