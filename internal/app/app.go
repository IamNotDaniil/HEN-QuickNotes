package app

import (
	"context"
	"net/http"
	"os"
	"path/filepath"

	"github.com/example/hen-quicknotes/internal/handlers"
	"github.com/example/hen-quicknotes/internal/repository"
	"github.com/example/hen-quicknotes/internal/services"
)

func NewServer(ctx context.Context, dataFile string) (*http.Server, error) {
	repo := repository.NewFileNoteRepository(resolveDataFile(dataFile))
	if err := repo.InitSchema(ctx); err != nil {
		return nil, err
	}
	svc := services.NewNotesService(repo)
	h := handlers.New(svc)
	mux := http.NewServeMux()
	h.Register(mux)
	return &http.Server{Addr: ":8080", Handler: mux}, nil
}

func resolveDataFile(path string) string {
	if path != "" {
		return path
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "data/notes.json"
	}
	return filepath.Join(cwd, "data", "notes.json")
}
