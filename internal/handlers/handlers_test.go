package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/example/hen-quicknotes/internal/handlers"
	"github.com/example/hen-quicknotes/internal/models"
	"github.com/example/hen-quicknotes/internal/repository"
	"github.com/example/hen-quicknotes/internal/services"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	repo := repository.NewMemoryNoteRepository()
	svc := services.NewNotesService(repo)
	h := handlers.New(svc)
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}

func createNote(t *testing.T, h http.Handler, title, content, tags string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(url.Values{"title": {title}, "content": {content}, "tags": {tags}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d", res.Code)
	}
}

func TestHealthz(t *testing.T) {
	h := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	if strings.TrimSpace(res.Body.String()) != "ok" {
		t.Fatalf("body = %q", res.Body.String())
	}
}

func TestNoteDetailPage(t *testing.T) {
	h := newTestServer(t)
	createNote(t, h, "Permalink note", "Opened as a dedicated page", "go, share")

	req := httptest.NewRequest(http.MethodGet, "/notes/1", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "Permalink note") || !strings.Contains(body, "Opened as a dedicated page") {
		t.Fatalf("unexpected note detail page: %s", body)
	}
	if !strings.Contains(body, "Back to notes") {
		t.Fatalf("expected back link on detail page: %s", body)
	}
}

func TestTagSuggestions(t *testing.T) {
	h := newTestServer(t)
	createNote(t, h, "Go note", "Tagged", "go, golang, htmx")
	createNote(t, h, "Second", "Tagged too", "backend")

	req := httptest.NewRequest(http.MethodGet, "/tags/suggest?q=go", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "value=\"go\"") || !strings.Contains(body, "value=\"golang\"") {
		t.Fatalf("expected matching tags in suggestions: %s", body)
	}
	if strings.Contains(body, "backend") {
		t.Fatalf("expected suggestions to be filtered: %s", body)
	}
}

func TestDuplicateCreatesCopy(t *testing.T) {
	h := newTestServer(t)
	createNote(t, h, "Template note", "Reuse me", "go, template")

	req := httptest.NewRequest(http.MethodPost, "/notes/1/duplicate", nil)
	req.Header.Set("HX-Request", "true")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("duplicate status = %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "Template note (copy)") {
		t.Fatalf("expected duplicated title in body: %s", body)
	}
	if strings.Contains(body, "📌 Template note (copy)") {
		t.Fatalf("duplicate should not inherit pinned state: %s", body)
	}
}

func TestNonHTMXDuplicateRedirectsToCopiedNote(t *testing.T) {
	h := newTestServer(t)
	createNote(t, h, "Template note", "Reuse me", "go, template")

	req := httptest.NewRequest(http.MethodPost, "/notes/1/duplicate", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", res.Code)
	}
	if location := res.Header().Get("Location"); location != "/notes/2" {
		t.Fatalf("location = %q", location)
	}
}

func TestDeleteMissingNoteReturnsNotFound(t *testing.T) {
	h := newTestServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/notes/999", nil)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d", res.Code)
	}
}

func TestPinMovesNoteToTop(t *testing.T) {
	h := newTestServer(t)
	createNote(t, h, "Older note", "Regular note", "go")
	createNote(t, h, "Important note", "Should be pinned", "go")

	req := httptest.NewRequest(http.MethodPost, "/notes/2/pin", nil)
	req.Header.Set("HX-Request", "true")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("pin status = %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "📌 Important note") {
		t.Fatalf("expected pinned marker in body: %s", body)
	}
	if strings.Index(body, "📌 Important note") > strings.Index(body, "Older note") {
		t.Fatalf("expected pinned note before regular note: %s", body)
	}
}

func TestMarkdownExportHonorsFilters(t *testing.T) {
	h := newTestServer(t)
	createNote(t, h, "Go note", "Export this one", "go, backend")
	createNote(t, h, "HTMX note", "Should not match", "htmx")

	req := httptest.NewRequest(http.MethodGet, "/export.md?tag=go", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	if got := res.Header().Get("Content-Type"); !strings.Contains(got, "text/markdown") {
		t.Fatalf("content-type = %q", got)
	}
	body := res.Body.String()
	if !strings.Contains(body, "# HEN-QuickNotes export") || !strings.Contains(body, "Go note") {
		t.Fatalf("unexpected export body: %s", body)
	}
	if strings.Contains(body, "HTMX note") {
		t.Fatalf("export should respect tag filter: %s", body)
	}
	if !strings.Contains(body, "#go") {
		t.Fatalf("expected markdown tag rendering: %s", body)
	}
}

func TestCreateSearchEditDeleteFlow(t *testing.T) {
	h := newTestServer(t)

	form := url.Values{"title": {"HTMX ideas"}, "content": {"Ship hypermedia UI fast"}, "tags": {"go, htmx"}}
	req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("create status = %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "HTMX ideas") {
		t.Fatalf("create response missing note: %s", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "Note created.") {
		t.Fatalf("create response missing success banner: %s", res.Body.String())
	}

	searchReq := httptest.NewRequest(http.MethodGet, "/notes?q=hypermedia", nil)
	searchReq.Header.Set("HX-Request", "true")
	searchRes := httptest.NewRecorder()
	h.ServeHTTP(searchRes, searchReq)
	if searchRes.Code != http.StatusOK {
		t.Fatalf("search status = %d", searchRes.Code)
	}
	if !strings.Contains(searchRes.Body.String(), "HTMX ideas") {
		t.Fatalf("search missing note: %s", searchRes.Body.String())
	}

	editReq := httptest.NewRequest(http.MethodGet, "/notes/1/edit", nil)
	editReq.Header.Set("HX-Request", "true")
	editRes := httptest.NewRecorder()
	h.ServeHTTP(editRes, editReq)
	if editRes.Code != http.StatusOK {
		t.Fatalf("edit page status = %d", editRes.Code)
	}
	if !strings.Contains(editRes.Body.String(), "Save changes") {
		t.Fatalf("edit page missing form: %s", editRes.Body.String())
	}

	updateForm := url.Values{"title": {"HTMX ideas updated"}, "content": {"Ship server-rendered UI fast"}, "tags": {"go, sqlite"}}
	updateReq := httptest.NewRequest(http.MethodPut, "/notes/1", strings.NewReader(updateForm.Encode()))
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateReq.Header.Set("HX-Request", "true")
	updateRes := httptest.NewRecorder()
	h.ServeHTTP(updateRes, updateReq)
	if updateRes.Code != http.StatusOK {
		t.Fatalf("update status = %d", updateRes.Code)
	}
	if !strings.Contains(updateRes.Body.String(), "HTMX ideas updated") {
		t.Fatalf("update missing new title: %s", updateRes.Body.String())
	}
	if !strings.Contains(updateRes.Body.String(), "Note updated.") {
		t.Fatalf("update missing success banner: %s", updateRes.Body.String())
	}

	tagReq := httptest.NewRequest(http.MethodGet, "/notes?tag=sqlite", nil)
	tagReq.Header.Set("HX-Request", "true")
	tagRes := httptest.NewRecorder()
	h.ServeHTTP(tagRes, tagReq)
	if !strings.Contains(tagRes.Body.String(), "HTMX ideas updated") {
		t.Fatalf("tag filter missing note: %s", tagRes.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/notes/1", nil)
	deleteReq.Header.Set("HX-Request", "true")
	deleteRes := httptest.NewRecorder()
	h.ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", deleteRes.Code)
	}
}

func TestCreateValidationErrorKeepsFormState(t *testing.T) {
	h := newTestServer(t)
	form := url.Values{"title": {""}, "content": {"Has content"}, "tags": {"go, htmx"}}
	req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "title is required") {
		t.Fatalf("expected validation error, got: %s", body)
	}
	if !strings.Contains(body, "Has content") {
		t.Fatalf("expected form content to be preserved, got: %s", body)
	}
}

func TestNonHTMXCreateRedirects(t *testing.T) {
	h := newTestServer(t)
	form := url.Values{"title": {"Plain browser"}, "content": {"Works without HTMX"}, "tags": {"fallback"}}
	req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", res.Code)
	}
	if location := res.Header().Get("Location"); location != "/" {
		t.Fatalf("location = %q", location)
	}
}

func TestFileRepositoryPersistsNotes(t *testing.T) {
	t.Parallel()
	dataPath := filepath.Join(t.TempDir(), "notes.json")
	repo := repository.NewFileNoteRepository(dataPath)
	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	created, err := repo.Create(context.Background(), "Persist me", "Stored on disk", []string{"go", "files"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dataPath); err != nil {
		t.Fatalf("expected data file to exist: %v", err)
	}

	reloaded := repository.NewFileNoteRepository(dataPath)
	if err := reloaded.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	notes, err := reloaded.List(context.Background(), models.NoteFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || notes[0].ID != created.ID || notes[0].Title != "Persist me" {
		t.Fatalf("unexpected reloaded notes: %+v", notes)
	}
}
