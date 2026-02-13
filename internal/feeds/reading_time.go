package feeds

import (
	"math"
	"strings"
	"unicode"
)

// wpmTechnical is the average words-per-minute reading speed for technical
// content, based on research suggesting ~238 WPM for technical material.
const wpmTechnical = 238

// CalculateReadingTime estimates reading time in minutes for the given text.
// Uses 238 WPM for technical content. Returns a minimum of 1 minute.
// Returns 0 for empty text.
func CalculateReadingTime(text string) int {
	words := countWords(text)
	if words == 0 {
		return 0
	}

	minutes := math.Ceil(float64(words) / wpmTechnical)
	if minutes < 1 {
		minutes = 1
	}
	return int(minutes)
}

// countWords counts whitespace-delimited words in the text.
func countWords(text string) int {
	count := 0
	inWord := false
	for _, r := range text {
		if unicode.IsSpace(r) || strings.ContainsRune(".,;:!?\"'()[]{}—–-", r) {
			if inWord {
				count++
				inWord = false
			}
		} else {
			inWord = true
		}
	}
	if inWord {
		count++
	}
	return count
}
