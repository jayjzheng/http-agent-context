package hac

import "testing"

func TestWantsHAC(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   bool
	}{
		{"exact match", "application/vnd.hac+json", true},
		{"with quality", "application/vnd.hac+json;q=1.0", true},
		{"zero quality", "application/vnd.hac+json;q=0", false},
		{"among others", "application/json, application/vnd.hac+json", true},
		{"with fallback", "application/vnd.hac+json, application/json;q=0.9", true},
		{"plain json only", "application/json", false},
		{"empty", "", false},
		{"wildcard", "*/*", false},
		{"application wildcard", "application/*", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wantsHAC(tt.accept); got != tt.want {
				t.Errorf("wantsHAC(%q) = %v, want %v", tt.accept, got, tt.want)
			}
		})
	}
}

func TestHACIsOnlyAcceptable(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   bool
	}{
		{"only hac", "application/vnd.hac+json", true},
		{"hac with json fallback", "application/vnd.hac+json, application/json;q=0.9", false},
		{"hac with zero quality others", "application/vnd.hac+json, application/json;q=0", true},
		{"plain json only", "application/json", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hacIsOnlyAcceptable(tt.accept); got != tt.want {
				t.Errorf("hacIsOnlyAcceptable(%q) = %v, want %v", tt.accept, got, tt.want)
			}
		})
	}
}

func TestParseAccept(t *testing.T) {
	ranges := parseAccept("text/html, application/json;q=0.9, */*;q=0.1")
	if len(ranges) != 3 {
		t.Fatalf("got %d ranges, want 3", len(ranges))
	}
	if ranges[0].quality != 1.0 {
		t.Errorf("first range quality = %v, want 1.0", ranges[0].quality)
	}
	if ranges[1].quality != 0.9 {
		t.Errorf("second range quality = %v, want 0.9", ranges[1].quality)
	}
}

func TestParseMediaRangeInvalid(t *testing.T) {
	mr := parseMediaRange("not-a-media-type")
	if mr.typ != "" {
		t.Errorf("expected empty type for invalid input, got %q", mr.typ)
	}
}
