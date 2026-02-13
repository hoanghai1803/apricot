package models

import "time"

// ReadingListItem represents a blog post saved to the user's reading list.
type ReadingListItem struct {
	ID      int64      `json:"id"`
	BlogID  int64      `json:"blog_id"`
	Blog    *Blog      `json:"blog,omitempty"`
	Summary *string    `json:"summary,omitempty"`
	Status  string     `json:"status"`
	Notes   *string    `json:"notes,omitempty"`
	Tags    []string   `json:"tags"`
	AddedAt time.Time  `json:"added_at"`
	ReadAt  *time.Time `json:"read_at,omitempty"`
}
