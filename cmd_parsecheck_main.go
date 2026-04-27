package main

import (
	"fmt"
	"os"

	"github.com/afeef-razick/manintheear/internal/script"
)

func main() {
	s, err := script.Parse(os.Args[1])
	if err != nil {
		fmt.Println("PARSE ERROR:", err)
		os.Exit(1)
	}
	fmt.Printf("talk_id=%s  total=%ds  phases=%d\n", s.TalkID, s.TotalDurationSeconds, len(s.Phases))
	totalPlanned := 0
	totalBeats := 0
	seenIDs := map[string]bool{}
	dupes := []string{}
	for _, p := range s.Phases {
		fmt.Printf("  phase %d %-30q  %ds  beats=%d\n", p.ID, p.Label, p.PlannedDurationSeconds, len(p.Beats))
		totalPlanned += p.PlannedDurationSeconds
		totalBeats += len(p.Beats)
		for _, b := range p.Beats {
			if seenIDs[b.ID] {
				dupes = append(dupes, b.ID)
			}
			seenIDs[b.ID] = true
			tags := ""
			if len(b.Tags) > 0 {
				tags = fmt.Sprintf(" %v", b.Tags)
			}
			fmt.Printf("    %-22s%s\n", b.ID, tags)
		}
	}
	fmt.Printf("\ntotals: planned=%ds (%dm)  beats=%d  unique_ids=%d\n", totalPlanned, totalPlanned/60, totalBeats, len(seenIDs))
	if len(dupes) > 0 {
		fmt.Println("DUPLICATE BEAT IDs:", dupes)
		os.Exit(1)
	}
	fmt.Println("OK")
}
