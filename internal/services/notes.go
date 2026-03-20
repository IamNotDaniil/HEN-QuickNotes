package services

import (
	"context"
	"strings"

	"github.com/example/hen-quicknotes/internal/models"
	"github.com/example/hen-quicknotes/internal/repository"
)

type NotesService struct {
	repo repository.NoteStore
}

func NewNotesService(repo repository.NoteStore) *NotesService {
	return &NotesService{repo: repo}
}

func (s *NotesService) Create(ctx context.Context, title, content, rawTags string) (models.Note, error) {
	return s.repo.Create(ctx, strings.TrimSpace(title), strings.TrimSpace(content), parseTags(rawTags))
}

func (s *NotesService) Update(ctx context.Context, id int64, title, content, rawTags string) (models.Note, error) {
	return s.repo.Update(ctx, id, strings.TrimSpace(title), strings.TrimSpace(content), parseTags(rawTags))
}

func (s *NotesService) TogglePin(ctx context.Context, id int64) (models.Note, error) {
	return s.repo.TogglePin(ctx, id)
}

func (s *NotesService) Duplicate(ctx context.Context, id int64) (models.Note, error) {
	return s.repo.Duplicate(ctx, id)
}

func (s *NotesService) Delete(ctx context.Context, id int64) error { return s.repo.Delete(ctx, id) }
func (s *NotesService) List(ctx context.Context, filter models.NoteFilter) ([]models.Note, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Tag = strings.TrimSpace(filter.Tag)
	return s.repo.List(ctx, filter)
}
func (s *NotesService) Tags(ctx context.Context) ([]string, error) { return s.repo.AllTags(ctx) }
func (s *NotesService) TagSuggestions(ctx context.Context, q string) ([]string, error) {
	tags, err := s.repo.AllTags(ctx)
	if err != nil {
		return nil, err
	}
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return tags, nil
	}
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), q) {
			result = append(result, tag)
		}
	}
	return result, nil
}
func (s *NotesService) GetByID(ctx context.Context, id int64) (models.Note, error) {
	return s.repo.GetByID(ctx, id)
}

func parseTags(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
