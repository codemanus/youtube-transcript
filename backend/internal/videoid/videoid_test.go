package videoid

import "testing"

func TestFromInput(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"dQw4w9WgXcQ", "dQw4w9WgXcQ", true},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ", true},
		{"https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ", true},
		{"https://www.youtube.com/embed/dQw4w9WgXcQ", "dQw4w9WgXcQ", true},
		{"https://www.youtube.com/shorts/dQw4w9WgXcQ", "dQw4w9WgXcQ", true},
		{"not-a-url", "", false},
		{"https://example.com/watch?v=dQw4w9WgXcQ", "", false},
	}
	for _, tt := range tests {
		got, err := FromInput(tt.in)
		if tt.ok {
			if err != nil || got != tt.want {
				t.Errorf("FromInput(%q) = %q, %v; want %q, nil", tt.in, got, err, tt.want)
			}
		} else {
			if err == nil {
				t.Errorf("FromInput(%q) = %q, nil; want error", tt.in, got)
			}
		}
	}
}
