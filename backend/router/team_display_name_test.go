package router

import "testing"

func TestNormalizeTeamDisplayName(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		invalid     bool
	}{
		{" Man Utd ", "Man Utd", false},
		{"", "", false},
		{"  ", "", false},
		{"123456789012", "123456789012", false},
		{"1234567890123", "", true},
		{"éééééééééééé", "éééééééééééé", false},
	} {
		got, err := normalizeTeamDisplayName(&tc.input)
		if (err != nil) != tc.invalid {
			t.Fatalf("input %q: error %v", tc.input, err)
		}
		if tc.invalid {
			continue
		}
		if tc.want == "" {
			if got != nil {
				t.Fatal("expected null")
			}
			continue
		}
		if got == nil || *got != tc.want {
			t.Fatalf("input %q: unexpected value", tc.input)
		}
	}
	if got, err := normalizeTeamDisplayName(nil); got != nil || err != nil {
		t.Fatal("nil should stay nil")
	}
}
