package repository

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/example/hen-quicknotes/internal/models"
)

type FileNoteRepository struct {
	mu     sync.RWMutex
	path   string
	nextID int64
	notes  map[int64]models.Note
}

type fileSnapshot struct {
	NextID int64         `json:"next_id"`
	Notes  []models.Note `json:"notes"`
}

func NewFileNoteRepository(path string) *FileNoteRepository {
	return &FileNoteRepository{path: path, nextID: 1, notes: make(map[int64]models.Note)}
}

func (r *FileNoteRepository) InitSchema(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return r.saveLocked()
		}
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return r.saveLocked()
	}
	var snapshot fileSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}
	r.nextID = snapshot.NextID
	if r.nextID < 1 {
		r.nextID = 1
	}
	for _, note := range snapshot.Notes {
		r.notes[note.ID] = note
		if note.ID >= r.nextID {
			r.nextID = note.ID + 1
		}
	}
	return nil
}

func (r *FileNoteRepository) Create(_ context.Context, title, content string, tags []string) (models.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	note := models.Note{ID: r.nextID, Title: title, Content: content, Tags: normalizeTags(tags), CreatedAt: now, UpdatedAt: now}
	r.notes[note.ID] = note
	r.nextID++
	return note, r.saveLocked()
}

func (r *FileNoteRepository) Update(_ context.Context, id int64, title, content string, tags []string) (models.Note, error) {
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
	return note, r.saveLocked()
}

func (r *FileNoteRepository) TogglePin(_ context.Context, id int64) (models.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	note, ok := r.notes[id]
	if !ok {
		return models.Note{}, ErrNotFound
	}
	note.Pinned = !note.Pinned
	note.UpdatedAt = time.Now().UTC()
	r.notes[id] = note
	return note, r.saveLocked()
}

func (r *FileNoteRepository) Duplicate(_ context.Context, id int64) (models.Note, error) {
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
	return clone, r.saveLocked()
}

func (r *FileNoteRepository) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.notes[id]; !ok {
		return ErrNotFound
	}
	delete(r.notes, id)
	return r.saveLocked()
}

func (r *FileNoteRepository) GetByID(_ context.Context, id int64) (models.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	note, ok := r.notes[id]
	if !ok {
		return models.Note{}, ErrNotFound
	}
	return note, nil
}

func (r *FileNoteRepository) List(_ context.Context, filter models.NoteFilter) ([]models.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return filterAndSort(r.notes, filter), nil
}

func (r *FileNoteRepository) AllTags(_ context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return collectTags(r.notes), nil
}

func (r *FileNoteRepository) saveLocked() error {
	notes := make([]models.Note, 0, len(r.notes))
	for _, note := range r.notes {
		notes = append(notes, note)
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].ID < notes[j].ID })
	payload, err := json.MarshalIndent(fileSnapshot{NextID: r.nextID, Notes: notes}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, payload, 0o644)
}
