package script

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Point is a single trackable moment within a phase.
type Point struct {
	ID          string
	Label       string
	Description string
	Tags        []string
}

// HasTag reports whether the point carries the given tag.
func (p *Point) HasTag(tag string) bool {
	for _, t := range p.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Phase is a timed section of the talk containing an ordered list of points.
type Phase struct {
	ID                     int
	Label                  string
	PlannedDurationSeconds int
	Points                 []Point
}

// Script is the fully-parsed talk plan.
type Script struct {
	TalkID               string
	TotalDurationSeconds int
	Phases               []Phase
}

// AllPoints returns every point across all phases in order.
func (s *Script) AllPoints() []Point {
	var out []Point
	for _, p := range s.Phases {
		out = append(out, p.Points...)
	}
	return out
}

// PointByID returns the point with the given ID, or nil.
func (s *Script) PointByID(id string) *Point {
	for i := range s.Phases {
		for j := range s.Phases[i].Points {
			if s.Phases[i].Points[j].ID == id {
				return &s.Phases[i].Points[j]
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

// PhaseForPoint returns the phase that owns the given point ID, or nil.
func (s *Script) PhaseForPoint(pointID string) *Phase {
	for i := range s.Phases {
		for _, p := range s.Phases[i].Points {
			if p.ID == pointID {
				return &s.Phases[i]
			}
		}
	}
	return nil
}

var (
	rePhaseHeader   = regexp.MustCompile(`^##\s+Phase\s+\d+`)
	rePointHeader   = regexp.MustCompile(`^###\s+Point:\s+(.+)`)
	rePhaseComment  = regexp.MustCompile(`<!--\s*phase_id:\s*(\d+),\s*planned_duration_seconds:\s*(\d+)\s*-->`)
	rePointComment  = regexp.MustCompile(`<!--\s*point_id:\s*([^,>\s]+)(?:,\s*tags:\s*\[([^\]]*)\])?\s*-->`)
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

	// Collect point block start indices.
	var pointStarts []int
	for i, l := range lines {
		if rePointHeader.MatchString(l) {
			pointStarts = append(pointStarts, i)
		}
	}
	for pi, start := range pointStarts {
		end := len(lines)
		if pi+1 < len(pointStarts) {
			end = pointStarts[pi+1]
		}
		point, err := parsePoint(lines[start:end])
		if err != nil {
			return nil, err
		}
		p.Points = append(p.Points, *point)
	}

	return p, nil
}

func parsePoint(lines []string) (*Point, error) {
	pt := &Point{}

	if m := rePointHeader.FindStringSubmatch(lines[0]); m != nil {
		pt.Label = strings.TrimSpace(m[1])
	}

	var descLines []string
	for _, l := range lines[1:] {
		if m := rePointComment.FindStringSubmatch(l); m != nil {
			pt.ID = strings.TrimSpace(m[1])
			if m[2] != "" {
				for _, tag := range strings.Split(m[2], ",") {
					if t := strings.TrimSpace(tag); t != "" {
						pt.Tags = append(pt.Tags, t)
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
	pt.Description = strings.Join(descLines, " ")

	if pt.ID == "" {
		return nil, fmt.Errorf("script: point %q missing point_id comment", pt.Label)
	}
	return pt, nil
}

func splitKV(line string) (key, value string, ok bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}
