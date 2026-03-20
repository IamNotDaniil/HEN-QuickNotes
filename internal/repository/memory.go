package repository

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/example/hen-quicknotes/internal/models"
)

var ErrNotFound = errors.New("note not found")

type MemoryNoteRepository struct {
	mu     sync.RWMutex
	nextID int64
	notes  map[int64]models.Note
}

func NewMemoryNoteRepository() *MemoryNoteRepository {
	return &MemoryNoteRepository{nextID: 1, notes: make(map[int64]models.Note)}
}

func (r *MemoryNoteRepository) InitSchema(context.Context) error { return nil }

func (r *MemoryNoteRepository) Create(_ context.Context, title, content string, tags []string) (models.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	note := models.Note{ID: r.nextID, Title: title, Content: content, Tags: normalizeTags(tags), CreatedAt: now, UpdatedAt: now}
	r.notes[note.ID] = note
	r.nextID++
	return note, nil
}

func (r *MemoryNoteRepository) Update(_ context.Context, id int64, title, content string, tags []string) (models.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	note, ok := r.notes[id]
	if !ok {
		return models.Note{}, ErrNotFound
	}
	note.Title = title
	note.Content = content
	note.Tags = normalizeTags(tags)
	note.UpdatedAt = time.Now().UTC()
	r.notes[id] = note
	return note, nil
}

func (r *MemoryNoteRepository) TogglePin(_ context.Context, id int64) (models.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	note, ok := r.notes[id]
	if !ok {
		return models.Note{}, ErrNotFound
	}
	note.Pinned = !note.Pinned
	note.UpdatedAt = time.Now().UTC()
	r.notes[id] = note
	return note, nil
}

func (r *MemoryNoteRepository) Duplicate(_ context.Context, id int64) (models.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	note, ok := r.notes[id]
	if !ok {
		return models.Note{}, ErrNotFound
	}
	now := time.Now().UTC()
	clone := note
	clone.ID = r.nextID
	clone.Title = note.Title + " (copy)"
	clone.Pinned = false
	clone.CreatedAt = now
	clone.UpdatedAt = now
	r.notes[clone.ID] = clone
	r.nextID++
	return clone, nil
}

func (r *MemoryNoteRepository) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.notes[id]; !ok {
		return ErrNotFound
	}
	delete(r.notes, id)
	return nil
}

func (r *MemoryNoteRepository) GetByID(_ context.Context, id int64) (models.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	note, ok := r.notes[id]
	if !ok {
		return models.Note{}, ErrNotFound
	}
	return note, nil
}

func (r *MemoryNoteRepository) List(_ context.Context, filter models.NoteFilter) ([]models.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return filterAndSort(r.notes, filter), nil
}

func (r *MemoryNoteRepository) AllTags(_ context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return collectTags(r.notes), nil
}

func filterAndSort(notesMap map[int64]models.Note, filter models.NoteFilter) []models.Note {
	q := strings.ToLower(strings.TrimSpace(filter.Query))
	tag := strings.ToLower(strings.TrimSpace(filter.Tag))
	var notes []models.Note
	for _, note := range notesMap {
		if q != "" && !matchesQuery(note, q) {
			continue
		}
		if tag != "" && !hasTag(note.Tags, tag) {
			continue
		}
		notes = append(notes, note)
	}
	sort.Slice(notes, func(i, j int) bool {
		if notes[i].Pinned != notes[j].Pinned {
			return notes[i].Pinned
		}
		if notes[i].UpdatedAt.Equal(notes[j].UpdatedAt) {
			return notes[i].ID > notes[j].ID
		}
		return notes[i].UpdatedAt.After(notes[j].UpdatedAt)
	})
	return notes
}

func collectTags(notesMap map[int64]models.Note) []string {
	set := map[string]struct{}{}
	for _, note := range notesMap {
		for _, tag := range note.Tags {
			set[tag] = struct{}{}
		}
	}
	tags := make([]string, 0, len(set))
	for tag := range set {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

func normalizeTags(tags []string) []string {
	set := map[string]struct{}{}
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			continue
		}
		if _, ok := set[tag]; ok {
			continue
		}
		set[tag] = struct{}{}
		result = append(result, tag)
	}
	sort.Strings(result)
	return result
}

func matchesQuery(note models.Note, q string) bool {
	if strings.Contains(strings.ToLower(note.Title), q) || strings.Contains(strings.ToLower(note.Content), q) {
		return true
	}
	return hasTag(note.Tags, q) || strings.Contains(strings.Join(note.Tags, " "), q)
}

func hasTag(tags []string, target string) bool {
	for _, tag := range tags {
		if strings.EqualFold(tag, target) {
			return true
		}
	}
	return false
}
