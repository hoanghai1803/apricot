package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/hoanghai1803/apricot/internal/models"
)

// AddTagToItem adds a tag to a reading list item. The tag is created if it
// doesn't exist yet. Returns an error if the reading list item doesn't exist
// or the tag is already attached.
func (s *Store) AddTagToItem(ctx context.Context, readingListID int64, tagName string) error {
	tagName = strings.TrimSpace(strings.ToLower(tagName))
	if tagName == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	// Verify the reading list item exists.
	var exists bool
	if err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM reading_list WHERE id = ?)`, readingListID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("checking reading list item: %w", err)
	}
	if !exists {
		return ErrNotFound
	}

	// Create tag if it doesn't exist.
	if _, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO tags (name) VALUES (?)`, tagName,
	); err != nil {
		return fmt.Errorf("creating tag: %w", err)
	}

	// Get tag ID.
	var tagID int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT id FROM tags WHERE name = ?`, tagName,
	).Scan(&tagID); err != nil {
		return fmt.Errorf("getting tag id: %w", err)
	}

	// Link tag to reading list item.
	if _, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO reading_list_tags (reading_list_id, tag_id) VALUES (?, ?)`,
		readingListID, tagID,
	); err != nil {
		return fmt.Errorf("linking tag to item: %w", err)
	}

	return nil
}

// RemoveTagFromItem removes a tag from a reading list item. If the tag is no
// longer used by any item, it is deleted from the tags table.
func (s *Store) RemoveTagFromItem(ctx context.Context, readingListID int64, tagName string) error {
	tagName = strings.TrimSpace(strings.ToLower(tagName))

	// Get tag ID.
	var tagID int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT id FROM tags WHERE name = ?`, tagName,
	).Scan(&tagID); err != nil {
		return ErrNotFound
	}

	// Remove the link.
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM reading_list_tags WHERE reading_list_id = ? AND tag_id = ?`,
		readingListID, tagID,
	)
	if err != nil {
		return fmt.Errorf("removing tag from item: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}

	// Clean up unused tags.
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM tags WHERE id = ? AND NOT EXISTS (
			SELECT 1 FROM reading_list_tags WHERE tag_id = ?
		)`, tagID, tagID,
	); err != nil {
		return fmt.Errorf("cleaning up unused tag: %w", err)
	}

	return nil
}

// GetAllTags returns all distinct tag names, ordered alphabetically.
func (s *Store) GetAllTags(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name FROM tags ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("querying tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning tag: %w", err)
		}
		tags = append(tags, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tags: %w", err)
	}

	if tags == nil {
		tags = []string{}
	}
	return tags, nil
}

// GetReadingListByTag returns reading list items that have the given tag.
func (s *Store) GetReadingListByTag(ctx context.Context, tag string) ([]models.ReadingListItem, error) {
	tag = strings.TrimSpace(strings.ToLower(tag))

	items, err := s.GetReadingList(ctx, "")
	if err != nil {
		return nil, err
	}

	var filtered []models.ReadingListItem
	for _, item := range items {
		for _, t := range item.Tags {
			if t == tag {
				filtered = append(filtered, item)
				break
			}
		}
	}

	if filtered == nil {
		filtered = []models.ReadingListItem{}
	}
	return filtered, nil
}

// loadTagsForItems loads tags for the given reading list items and attaches
// them to each item's Tags field. This is called by GetReadingList after the
// main query to avoid complicating the already-complex JOIN.
func (s *Store) loadTagsForItems(ctx context.Context, items []models.ReadingListItem) error {
	if len(items) == 0 {
		return nil
	}

	// Build IN clause with item IDs.
	ids := make([]string, len(items))
	args := make([]any, len(items))
	for i, item := range items {
		ids[i] = "?"
		args[i] = item.ID
	}

	query := fmt.Sprintf(
		`SELECT rlt.reading_list_id, t.name
		 FROM reading_list_tags rlt
		 JOIN tags t ON t.id = rlt.tag_id
		 WHERE rlt.reading_list_id IN (%s)
		 ORDER BY t.name`,
		strings.Join(ids, ","),
	)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("loading tags for items: %w", err)
	}
	defer rows.Close()

	// Build a map of reading_list_id -> []string.
	tagMap := make(map[int64][]string)
	for rows.Next() {
		var rlID int64
		var tagName string
		if err := rows.Scan(&rlID, &tagName); err != nil {
			return fmt.Errorf("scanning tag row: %w", err)
		}
		tagMap[rlID] = append(tagMap[rlID], tagName)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating tag rows: %w", err)
	}

	// Attach tags to items.
	for i := range items {
		if tags, ok := tagMap[items[i].ID]; ok {
			items[i].Tags = tags
		} else {
			items[i].Tags = []string{}
		}
	}

	return nil
}
