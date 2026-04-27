package script

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Beat is a single trackable moment within a phase.
type Beat struct {
	ID          string
	Label       string
	Description string
	Tags        []string
}

// HasTag reports whether the beat carries the given tag.
func (b *Beat) HasTag(tag string) bool {
	for _, t := range b.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Phase is a timed section of the talk containing an ordered list of beats.
type Phase struct {
	ID                     int
	Label                  string
	PlannedDurationSeconds int
	Beats                  []Beat
}

// Script is the fully-parsed talk plan.
type Script struct {
	TalkID               string
	TotalDurationSeconds int
	Phases               []Phase
}

// AllBeats returns every beat across all phases in order.
func (s *Script) AllBeats() []Beat {
	var out []Beat
	for _, p := range s.Phases {
		out = append(out, p.Beats...)
	}
	return out
}

// BeatByID returns the beat with the given ID, or nil.
func (s *Script) BeatByID(id string) *Beat {
	for i := range s.Phases {
		for j := range s.Phases[i].Beats {
			if s.Phases[i].Beats[j].ID == id {
				return &s.Phases[i].Beats[j]
			}
		}
	}
	return nil
}

// PhaseByID returns the phase with the given ID, or nil.
func (s *Script) PhaseByID(id int) *Phase {
	for i := range s.Phases {
		if s.Phases[i].ID == id {
			return &s.Phases[i]
		}
	}
	return nil
}

// PhaseForBeat returns the phase that owns the given beat ID, or nil.
func (s *Script) PhaseForBeat(beatID string) *Phase {
	for i := range s.Phases {
		for _, b := range s.Phases[i].Beats {
			if b.ID == beatID {
				return &s.Phases[i]
			}
		}
	}
	return nil
}

var (
	rePhaseHeader  = regexp.MustCompile(`^##\s+Phase\s+\d+`)
	reBeatHeader   = regexp.MustCompile(`^###\s+Beat:\s+(.+)`)
	rePhaseComment = regexp.MustCompile(`<!--\s*phase_id:\s*(\d+),\s*planned_duration_seconds:\s*(\d+)\s*-->`)
	reBeatComment  = regexp.MustCompile(`<!--\s*beat_id:\s*([^,>\s]+)(?:,\s*tags:\s*\[([^\]]*)\])?\s*-->`)
)

// Parse reads and parses the script file at the given path.
func Parse(path string) (*Script, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("script: open %q: %w", path, err)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("script: read %q: %w", path, err)
	}

	return parseLines(lines)
}

func parseLines(lines []string) (*Script, error) {
	s := &Script{}

	// Frontmatter between the first pair of "---" delimiters.
	lines, err := parseFrontmatter(lines, s)
	if err != nil {
		return nil, err
	}

	// Collect the line index where each phase header starts.
	var phaseStarts []int
	for i, l := range lines {
		if rePhaseHeader.MatchString(l) {
			phaseStarts = append(phaseStarts, i)
		}
	}

	for pi, start := range phaseStarts {
		end := len(lines)
		if pi+1 < len(phaseStarts) {
			end = phaseStarts[pi+1]
		}
		phase, err := parsePhase(lines[start:end])
		if err != nil {
			return nil, err
		}
		s.Phases = append(s.Phases, *phase)
	}

	return s, nil
}

func parseFrontmatter(lines []string, s *Script) ([]string, error) {
	if len(lines) == 0 || lines[0] != "---" {
		return lines, nil
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
		if k, v, ok := splitKV(lines[i]); ok {
			switch k {
			case "talk_id":
				s.TalkID = v
			case "total_duration_seconds":
				s.TotalDurationSeconds, _ = strconv.Atoi(v)
			}
		}
	}
	if end < 0 {
		return lines, nil
	}
	return lines[end+1:], nil
}

func parsePhase(lines []string) (*Phase, error) {
	p := &Phase{}

	// Label from the ## header: "## Phase N: Label · Xs"
	header := strings.TrimPrefix(lines[0], "## ")
	if idx := strings.Index(header, ":"); idx >= 0 {
		rest := strings.TrimSpace(header[idx+1:])
		if dot := strings.Index(rest, "·"); dot >= 0 {
			p.Label = strings.TrimSpace(rest[:dot])
		} else {
			p.Label = rest
		}
	}

	// Metadata from the HTML comment.
	for _, l := range lines {
		if m := rePhaseComment.FindStringSubmatch(l); m != nil {
			p.ID, _ = strconv.Atoi(m[1])
			p.PlannedDurationSeconds, _ = strconv.Atoi(m[2])
			break
		}
	}
	if p.ID == 0 {
		return nil, fmt.Errorf("script: phase %q missing phase_id comment", p.Label)
	}

	// Collect beat block start indices.
	var beatStarts []int
	for i, l := range lines {
		if reBeatHeader.MatchString(l) {
			beatStarts = append(beatStarts, i)
		}
	}
	for bi, start := range beatStarts {
		end := len(lines)
		if bi+1 < len(beatStarts) {
			end = beatStarts[bi+1]
		}
		beat, err := parseBeat(lines[start:end])
		if err != nil {
			return nil, err
		}
		p.Beats = append(p.Beats, *beat)
	}

	return p, nil
}

func parseBeat(lines []string) (*Beat, error) {
	b := &Beat{}

	if m := reBeatHeader.FindStringSubmatch(lines[0]); m != nil {
		b.Label = strings.TrimSpace(m[1])
	}

	var descLines []string
	for _, l := range lines[1:] {
		if m := reBeatComment.FindStringSubmatch(l); m != nil {
			b.ID = strings.TrimSpace(m[1])
			if m[2] != "" {
				for _, tag := range strings.Split(m[2], ",") {
					if t := strings.TrimSpace(tag); t != "" {
						b.Tags = append(b.Tags, t)
					}
				}
			}
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(l), "<!--") {
			continue
		}
		if t := strings.TrimSpace(l); t != "" {
			descLines = append(descLines, t)
		}
	}
	b.Description = strings.Join(descLines, " ")

	if b.ID == "" {
		return nil, fmt.Errorf("script: beat %q missing beat_id comment", b.Label)
	}
	return b, nil
}

func splitKV(line string) (key, value string, ok bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}
