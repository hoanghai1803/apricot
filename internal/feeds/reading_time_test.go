package feeds

import (
	"strings"
	"testing"
)

func TestCalculateReadingTime(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{
			name: "empty text",
			text: "",
			want: 0,
		},
		{
			name: "whitespace only",
			text: "   \n\t  ",
			want: 0,
		},
		{
			name: "single word",
			text: "hello",
			want: 1,
		},
		{
			name: "short paragraph",
			text: "This is a short paragraph with just a few words in it.",
			want: 1,
		},
		{
			name: "238 words equals 1 minute",
			text: strings.Repeat("word ", 238),
			want: 1,
		},
		{
			name: "239 words equals 2 minutes",
			text: strings.Repeat("word ", 239),
			want: 2,
		},
		{
			name: "1000 words is about 5 minutes",
			text: strings.Repeat("word ", 1000),
			want: 5,
		},
		{
			name: "2000 words is about 9 minutes",
			text: strings.Repeat("word ", 2000),
			want: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateReadingTime(tt.text)
			if got != tt.want {
				t.Errorf("CalculateReadingTime() = %d, want %d", got, tt.want)
			}
		})
	}
}
