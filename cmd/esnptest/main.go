package main

import (
	"fmt"
	"time"
	"tifo/espn"
)

func main() {
	svc := espn.NewService()
	
	t := time.Date(2026, 6, 25, 1, 0, 0, 0, time.UTC)
	data, err := svc.FetchMatch(0, "World Cup Grp. A", t, "Czechia", "Mexico")
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	fmt.Printf("EventID: %s\n", data.EventID)
	fmt.Printf("Home: %s\n", data.HomeTeam)
	fmt.Printf("Away: %s\n", data.AwayTeam)
	
	s := data.Summary
	fmt.Printf("Venue: %s\n", s.GameInfo.Venue.FullName)
	fmt.Printf("Attendance: %d\n", s.GameInfo.Attendance)
	for _, off := range s.GameInfo.Officials {
		fmt.Printf("Official: %s (%s)\n", off.DisplayName, off.Position.DisplayName)
	}
	fmt.Printf("KeyEvents: %d\n", len(s.KeyEvents))
	
	for i, ke := range s.KeyEvents {
		if i >= 5 {
			break
		}
		fmt.Printf("  KE %d: %s %s %s\n", i, ke.Clock.DisplayValue, ke.Type.Text, ke.Text)
	}
}
