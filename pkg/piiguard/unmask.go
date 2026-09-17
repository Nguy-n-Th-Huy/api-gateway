package piiguard

import (
	"io"
	"sort"
	"strings"
)

// unmaskReader restores fake values in an upstream response while it streams.
//
// The reader works on bytes rather than on a parsed payload because the same
// path carries SSE chunks, plain JSON and passthrough bodies. A chunk boundary
// can fall inside a fake, so the reader holds back the tail that is still a
// prefix of a known fake and emits it once the next chunk resolves it.
type unmaskReader struct {
	source io.Reader

	// table maps a fake to the real value it stands for.
	table map[string]string
	// fakes are the table keys ordered by length, so the longest match wins.
	fakes []string
	// maxFakeLength caps the hold-back window.
	maxFakeLength int

	pending []byte
	// ready holds unmasked bytes that did not fit the caller's buffer.
	ready   []byte
	drained bool
}

func newUnmaskReader(source io.Reader, table map[string]string, maxFakeLength int) io.Reader {
	fakes := make([]string, 0, len(table))
	for fake := range table {
		if fake == "" {
			continue
		}
		fakes = append(fakes, fake)
		if len(fake) > maxFakeLength {
			maxFakeLength = len(fake)
		}
	}
	sort.Slice(fakes, func(left, right int) bool {
		return len(fakes[left]) > len(fakes[right])
	})
	return &unmaskReader{
		source:        source,
		table:         table,
		fakes:         fakes,
		maxFakeLength: maxFakeLength,
	}
}

func (r *unmaskReader) Read(buffer []byte) (int, error) {
	for {
		if len(r.ready) > 0 {
			written := copy(buffer, r.ready)
			r.ready = r.ready[written:]
			return written, nil
		}
		if r.drained {
			if len(r.pending) == 0 {
				return 0, io.EOF
			}
			r.ready = []byte(applyUnmaskTableSorted(string(r.pending), r.table, r.fakes))
			r.pending = nil
			continue
		}
		chunk := make([]byte, 32*1024)
		read, err := r.source.Read(chunk)
		if read > 0 {
			r.pending = append(r.pending, chunk[:read]...)
			r.queueSafePrefix()
		}
		if err == io.EOF {
			r.drained = true
		} else if err != nil {
			r.drained = true
			if len(r.pending) == 0 {
				return 0, err
			}
		}
	}
}

// queueSafePrefix unmarshals the part of pending that cannot still turn into a
// fake and moves it into ready.
func (r *unmaskReader) queueSafePrefix() {
	masked := string(r.pending)
	safe := r.safePrefixLength(masked)
	if safe <= 0 {
		return
	}
	r.ready = append(r.ready, applyUnmaskTableSorted(masked[:safe], r.table, r.fakes)...)
	r.pending = append(r.pending[:0], r.pending[safe:]...)
}

// safePrefixLength returns how many bytes of masked are safe to emit without
// risking a fake split across chunks.
func (r *unmaskReader) safePrefixLength(masked string) int {
	limit := r.maxFakeLength
	if limit > len(masked) {
		limit = len(masked)
	}
	// Walk back from the end: the longest tail that is a prefix of a fake, and
	// is shorter than that fake, determines how much has to be held back. A tail
	// equal to a whole fake is not a risk — it can never grow into a longer
	// fake, so holding it back would deadlock the read when no more bytes come.
	for tail := limit; tail >= 1; tail-- {
		if r.isPartialFakePrefix(masked[len(masked)-tail:]) {
			return len(masked) - tail
		}
	}
	return len(masked)
}

// isPartialFakePrefix reports whether candidate is the beginning of a known fake
// without already being a complete fake.
func (r *unmaskReader) isPartialFakePrefix(candidate string) bool {
	for _, fake := range r.fakes {
		if len(fake) > len(candidate) && strings.HasPrefix(fake, candidate) {
			return true
		}
	}
	return false
}

// applyUnmaskTable replaces every fake that appears in text with its real
// value, scanning left to right so a fake is never examined twice.
func applyUnmaskTable(text string, table map[string]string) string {
	return applyUnmaskTableSorted(text, table, sortedFakes(table))
}

// applyUnmaskTableSorted is applyUnmaskTable with the key list supplied by the
// caller, which keeps the sort out of the streaming read path.
func applyUnmaskTableSorted(text string, table map[string]string, fakes []string) string {
	if len(table) == 0 || text == "" {
		return text
	}
	var builder strings.Builder
	builder.Grow(len(text))
	cursor := 0
	for cursor < len(text) {
		matched := ""
		for _, fake := range fakes {
			if len(fake) > len(text)-cursor {
				continue
			}
			if text[cursor:cursor+len(fake)] == fake {
				matched = fake
				break
			}
		}
		if matched == "" {
			builder.WriteByte(text[cursor])
			cursor++
			continue
		}
		builder.WriteString(table[matched])
		cursor += len(matched)
	}
	return builder.String()
}

// sortedFakes returns the replacement table keys ordered by length, longest
// first, so the longest fake always wins at a given position.
func sortedFakes(table map[string]string) []string {
	fakes := make([]string, 0, len(table))
	for fake := range table {
		if fake != "" {
			fakes = append(fakes, fake)
		}
	}
	sort.Slice(fakes, func(left, right int) bool {
		return len(fakes[left]) > len(fakes[right])
	})
	return fakes
}
