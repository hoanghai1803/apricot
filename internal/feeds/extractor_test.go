package feeds

import "testing"

func TestTruncateWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWords int
		want     string
	}{
		{
			name:     "under limit returns original",
			input:    "hello world",
			maxWords: 5,
			want:     "hello world",
		},
		{
			name:     "exactly at limit returns original",
			input:    "one two three",
			maxWords: 3,
			want:     "one two three",
		},
		{
			name:     "over limit is truncated",
			input:    "one two three four five six",
			maxWords: 3,
			want:     "one two three",
		},
		{
			name:     "empty string returns empty",
			input:    "",
			maxWords: 5,
			want:     "",
		},
		{
			name:     "single word under limit",
			input:    "hello",
			maxWords: 5,
			want:     "hello",
		},
		{
			name:     "single word at limit",
			input:    "hello",
			maxWords: 1,
			want:     "hello",
		},
		{
			name:     "multiple spaces between words",
			input:    "one   two   three   four",
			maxWords: 2,
			want:     "one two",
		},
		{
			name:     "leading and trailing whitespace",
			input:    "  one two three  ",
			maxWords: 2,
			want:     "one two",
		},
		{
			name:     "whitespace only string",
			input:    "   ",
			maxWords: 5,
			want:     "   ",
		},
		{
			name:     "tabs and newlines",
			input:    "one\ttwo\nthree\rfour",
			maxWords: 2,
			want:     "one two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateWords(tt.input, tt.maxWords)
			if got != tt.want {
				t.Errorf("truncateWords(%q, %d) = %q, want %q", tt.input, tt.maxWords, got, tt.want)
			}
		})
	}
}
