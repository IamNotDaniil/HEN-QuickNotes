package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/hen-quicknotes/internal/models"
	"github.com/example/hen-quicknotes/internal/repository"
	"github.com/example/hen-quicknotes/internal/services"
	"github.com/example/hen-quicknotes/internal/views"
)

type Handler struct {
	svc *services.NotesService
}

func New(svc *services.NotesService) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/healthz", h.healthz)
	mux.HandleFunc("/export.md", h.exportMarkdown)
	mux.HandleFunc("/tags/suggest", h.tagSuggestions)
	mux.HandleFunc("/", h.home)
	mux.HandleFunc("/notes", h.notes)
	mux.HandleFunc("/notes/", h.noteByID)
}

func (h *Handler) tagSuggestions(w http.ResponseWriter, r *http.Request) {
	tags, err := h.svc.TagSuggestions(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.RenderTagOptions(w, tags); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) exportMarkdown(w http.ResponseWriter, r *http.Request) {
	filter := models.NoteFilter{Query: r.URL.Query().Get("q"), Tag: r.URL.Query().Get("tag")}
	notes, err := h.svc.List(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="quicknotes-export.md"`)
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "# HEN-QuickNotes export\n\n")
	if filter.Query != "" {
		_, _ = fmt.Fprintf(w, "- Query filter: `%s`\n", filter.Query)
	}
	if filter.Tag != "" {
		_, _ = fmt.Fprintf(w, "- Tag filter: `%s`\n", filter.Tag)
	}
	_, _ = fmt.Fprintf(w, "- Exported at: %s\n\n", time.Now().UTC().Format(time.RFC3339))
	if len(notes) == 0 {
		_, _ = fmt.Fprintln(w, "_No notes matched the current filters._")
		return
	}
	for _, note := range notes {
		_, _ = fmt.Fprintf(w, "## %s\n\n", note.Title)
		_, _ = fmt.Fprintf(w, "%s\n\n", note.Content)
		if len(note.Tags) > 0 {
			_, _ = fmt.Fprintf(w, "Tags: %s\n\n", strings.Join(prefixTags(note.Tags), ", "))
		}
		_, _ = fmt.Fprintf(w, "Created: %s  \nUpdated: %s\n\n---\n\n", note.CreatedAt.UTC().Format(time.RFC3339), note.UpdatedAt.UTC().Format(time.RFC3339))
	}
}

func prefixTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		out = append(out, "#"+strings.TrimSpace(tag))
	}
	return out
}

func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	h.renderPage(w, r, pageOptions{})
}

func (h *Handler) notes(w http.ResponseWriter, r *http.Request) {
	method := effectiveMethod(r)
	switch method {
	case http.MethodGet:
		if isHTMX(r) {
			h.renderList(w, r)
			return
		}
		h.renderPage(w, r, pageOptions{})
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		title := r.FormValue("title")
		content := r.FormValue("content")
		tags := r.FormValue("tags")
		if err := validateNoteInput(title, content); err != nil {
			h.renderPageWithStatus(w, r, pageOptions{ErrorMessage: err.Error(), FormTitle: title, FormContent: content, FormTags: tags}, http.StatusBadRequest)
			return
		}
		if _, err := h.svc.Create(r.Context(), title, content, tags); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if isHTMX(r) {
			h.renderPage(w, r, pageOptions{SuccessMessage: "Note created."})
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) noteByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/notes/")
	if strings.HasSuffix(path, "/duplicate") {
		idStr := strings.TrimSuffix(strings.TrimSuffix(path, "/duplicate"), "/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		method := effectiveMethod(r)
		if method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		duplicated, err := h.svc.Duplicate(r.Context(), id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if isHTMX(r) {
			h.renderPage(w, r, pageOptions{SuccessMessage: "Note duplicated."})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/notes/%d", duplicated.ID), http.StatusSeeOther)
		return
	}
	if strings.HasSuffix(path, "/pin") {
		idStr := strings.TrimSuffix(strings.TrimSuffix(path, "/pin"), "/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		method := effectiveMethod(r)
		if method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if _, err := h.svc.TogglePin(r.Context(), id); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if isHTMX(r) {
			h.renderPage(w, r, pageOptions{SuccessMessage: "Pin updated."})
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if strings.HasSuffix(path, "/edit") && r.Method == http.MethodGet {
		idStr := strings.TrimSuffix(strings.TrimSuffix(path, "/edit"), "/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		note, err := h.svc.GetByID(r.Context(), id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		h.renderPage(w, r, pageOptions{EditingNote: &note})
		return
	}
	idStr := strings.Split(strings.Trim(path, "/"), "/")[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet {
		note, err := h.svc.GetByID(r.Context(), id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		h.renderNotePage(w, views.PageData{CurrentNote: &note})
		return
	}

	method := effectiveMethod(r)
	switch method {
	case http.MethodDelete:
		if err := h.svc.Delete(r.Context(), id); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if isHTMX(r) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	case http.MethodPut:
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		title := r.FormValue("title")
		content := r.FormValue("content")
		tags := r.FormValue("tags")
		if err := validateNoteInput(title, content); err != nil {
			existing, getErr := h.svc.GetByID(r.Context(), id)
			if getErr != nil {
				http.NotFound(w, r)
				return
			}
			existing.Title = title
			existing.Content = content
			existing.Tags = strings.Split(tags, ",")
			h.renderPageWithStatus(w, r, pageOptions{EditingNote: &existing, ErrorMessage: err.Error(), FormTitle: title, FormContent: content, FormTags: tags}, http.StatusBadRequest)
			return
		}
		if _, err := h.svc.Update(r.Context(), id, title, content, tags); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if isHTMX(r) {
			h.renderPage(w, r, pageOptions{SuccessMessage: "Note updated."})
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type pageOptions struct {
	EditingNote    *models.Note
	ErrorMessage   string
	SuccessMessage string
	FormTitle      string
	FormContent    string
	FormTags       string
}

func (h *Handler) pageData(r *http.Request, opts pageOptions) (views.PageData, error) {
	filter := models.NoteFilter{Query: r.URL.Query().Get("q"), Tag: r.URL.Query().Get("tag")}
	notes, err := h.svc.List(r.Context(), filter)
	if err != nil {
		return views.PageData{}, err
	}
	tags, err := h.svc.Tags(r.Context())
	if err != nil {
		return views.PageData{}, err
	}
	data := views.PageData{Notes: notes, Tags: tags, Filter: filter, EditingNote: opts.EditingNote, ErrorMessage: opts.ErrorMessage, SuccessMessage: opts.SuccessMessage, FormTitle: opts.FormTitle, FormContent: opts.FormContent, FormTags: opts.FormTags}
	if data.EditingNote != nil {
		if data.FormTitle == "" {
			data.FormTitle = data.EditingNote.Title
		}
		if data.FormContent == "" {
			data.FormContent = data.EditingNote.Content
		}
		if data.FormTags == "" {
			data.FormTags = strings.Join(data.EditingNote.Tags, ", ")
		}
	}
	return data, nil
}

func (h *Handler) renderPage(w http.ResponseWriter, r *http.Request, opts pageOptions) {
	h.renderPageWithStatus(w, r, opts, http.StatusOK)
}

func (h *Handler) renderPageWithStatus(w http.ResponseWriter, r *http.Request, opts pageOptions, status int) {
	data, err := h.pageData(r, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := views.RenderPage(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) renderNotePage(w http.ResponseWriter, data views.PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.RenderNotePage(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) renderList(w http.ResponseWriter, r *http.Request) {
	data, err := h.pageData(r, pageOptions{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.RenderList(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func isHTMX(r *http.Request) bool { return strings.EqualFold(r.Header.Get("HX-Request"), "true") }

func effectiveMethod(r *http.Request) string {
	if r.Method != http.MethodPost {
		return r.Method
	}
	if err := r.ParseForm(); err == nil {
		if override := strings.ToUpper(strings.TrimSpace(r.FormValue("_method"))); override != "" {
			return override
		}
	}
	return r.Method
}

func validateNoteInput(title, content string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(content) == "" {
		return errors.New("content is required")
	}
	return nil
}
