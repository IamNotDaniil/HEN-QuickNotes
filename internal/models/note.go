package models

import "time"

type Note struct {
	ID        int64
	Title     string
	Content   string
	Tags      []string
	Pinned    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type NoteFilter struct {
	Query string
	Tag   string
}
