package cloze

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// re matches {{cN::answer}} and {{cN::answer::hint}} patterns.
var re = regexp.MustCompile(`\{\{c(\d+)::([^}]*?)(?:::([^}]*?))?\}\}`)

// Deletion represents a single cloze deletion extracted from a cloze text.
type Deletion struct {
	Index  int
	Answer string
	Hint   string // empty if no hint was specified
}

// Parse extracts all cloze deletions from text in the order they appear.
func Parse(text string) []Deletion {
	matches := re.FindAllStringSubmatch(text, -1)
	out := make([]Deletion, 0, len(matches))
	for _, m := range matches {
		idx, _ := strconv.Atoi(m[1])
		out = append(out, Deletion{Index: idx, Answer: m[2], Hint: m[3]})
	}
	return out
}

// Indices returns the sorted unique cloze numbers found in text.
func Indices(text string) []int {
	seen := map[int]bool{}
	for _, d := range Parse(text) {
		seen[d.Index] = true
	}
	out := make([]int, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// Render produces the front and back strings for a cloze card identified by
// activeIndex (1-based, matching the cN number).
//
// Front: active cloze replaced with "[...]" or "[hint]"; inactive clozes show
// plain answer text.
//
// Back: active cloze replaced with "<b>answer</b>" (highlighted); inactive
// clozes show plain answer text.
func Render(text string, activeIndex int) (front, back string) {
	front = re.ReplaceAllStringFunc(text, func(match string) string {
		m := re.FindStringSubmatch(match)
		idx, _ := strconv.Atoi(m[1])
		if idx == activeIndex {
			if m[3] != "" {
				return "[" + m[3] + "]"
			}
			return "[...]"
		}
		return m[2]
	})
	back = re.ReplaceAllStringFunc(text, func(match string) string {
		m := re.FindStringSubmatch(match)
		idx, _ := strconv.Atoi(m[1])
		if idx == activeIndex {
			return "<b>" + m[2] + "</b>"
		}
		return m[2]
	})
	// Trim surrounding whitespace that may be left after replacements.
	front = strings.TrimSpace(front)
	back = strings.TrimSpace(back)
	return front, back
}
