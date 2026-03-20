package repository

import (
	"context"

	"github.com/example/hen-quicknotes/internal/models"
)

type NoteStore interface {
	InitSchema(ctx context.Context) error
	Create(ctx context.Context, title, content string, tags []string) (models.Note, error)
	Update(ctx context.Context, id int64, title, content string, tags []string) (models.Note, error)
	TogglePin(ctx context.Context, id int64) (models.Note, error)
	Duplicate(ctx context.Context, id int64) (models.Note, error)
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (models.Note, error)
	List(ctx context.Context, filter models.NoteFilter) ([]models.Note, error)
	AllTags(ctx context.Context) ([]string, error)
}
