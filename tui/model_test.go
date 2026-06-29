package tui

import "testing"

func TestPosAbbr(t *testing.T) {
	tests := []struct {
		name    string
		posID   int
		posName string
		want    string
	}{
		{name: "returns posName as-is", posName: "POR", want: "POR"},
		{name: "with abbreviation", posName: "DC", want: "DC"},
		{name: "falls back to posID when name is empty", posID: 2, posName: "", want: "DFC"},
		{name: "empty", posID: 99, posName: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := posAbbr(tt.posID, tt.posName); got != tt.want {
				t.Fatalf("posAbbr(%d, %q) = %q, want %q", tt.posID, tt.posName, got, tt.want)
			}
		})
	}
}
