package config

import "testing"

func TestParseByteSize(t *testing.T) {
	for input, want := range map[string]int{
		"4MB":     4 << 20,
		"10mb":    10 << 20,
		"512KB":   512 << 10,
		"1GB":     1 << 30,
		"1048576": 1 << 20,
		" 2 MB ":  2 << 20,
		"":        0,
		"ten":     0,
		"-1MB":    0,
	} {
		if got := parseByteSize(input); got != want {
			t.Errorf("parseByteSize(%q) = %d, want %d", input, got, want)
		}
	}
}
