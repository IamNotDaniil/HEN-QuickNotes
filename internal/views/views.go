package views

import (
	"html/template"
	"io"
	"net/url"
	"strings"

	"github.com/example/hen-quicknotes/internal/models"
)

type PageData struct {
	Notes          []models.Note
	Tags           []string
	Filter         models.NoteFilter
	EditingNote    *models.Note
	CurrentNote    *models.Note
	ErrorMessage   string
	SuccessMessage string
	FormTitle      string
	FormContent    string
	FormTags       string
}

var templates = template.Must(template.New("base").Funcs(template.FuncMap{
	"join":    strings.Join,
	"qescape": url.QueryEscape,
}).Parse(`{{define "page"}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>HEN-QuickNotes</title>
  <script src="https://unpkg.com/htmx.org@1.9.12"></script>
  <link rel="stylesheet" href="/static/styles.css">
</head>
<body>
  <main class="shell">
    <section class="hero">
      <div>
        <p class="eyebrow">Go + HTMX + SSR</p>
        <h1>Quick notes without a heavy frontend.</h1>
        <p class="lead">Create, search, filter, edit, delete, and export notes from one screen. The server returns HTML fragments, and HTMX swaps them in instantly.</p>
      </div>
      <div class="stats">
        <div><strong>{{len .Notes}}</strong><span>notes</span></div>
        <div><strong>{{len .Tags}}</strong><span>tags</span></div>
      </div>
    </section>

    <section class="grid">
      <aside class="panel">{{template "form" .}}</aside>
      <section class="panel stack">
        {{template "toolbar" .}}
        <div id="notes-list">{{template "list" .}}</div>
      </section>
    </section>
  </main>
</body>
</html>
{{end}}

{{define "note-page"}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.CurrentNote.Title}} — HEN-QuickNotes</title>
  <link rel="stylesheet" href="/static/styles.css">
</head>
<body>
  <main class="shell detail-shell">
    <a class="secondary-btn" href="/">← Back to notes</a>
    <article class="panel detail-card">
      <div class="note-top">
        <div>
          <p class="eyebrow">Note permalink</p>
          <h1 class="detail-title">{{if .CurrentNote.Pinned}}📌 {{end}}{{.CurrentNote.Title}}</h1>
          <p class="meta">Created {{.CurrentNote.CreatedAt.Format "02 Jan 2006 15:04"}} · Updated {{.CurrentNote.UpdatedAt.Format "02 Jan 2006 15:04"}}</p>
        </div>
        <div class="inline-actions">
          <form class="inline-form" action="/notes/{{.CurrentNote.ID}}/duplicate" method="post">
            <button class="secondary-btn" type="submit">Duplicate</button>
          </form>
          <a class="secondary-btn" href="/export.md?q={{qescape .CurrentNote.Title}}">Export related notes</a>
        </div>
      </div>
      <p class="content detail-content">{{.CurrentNote.Content}}</p>
      {{if .CurrentNote.Tags}}
      <div class="tag-row">
        {{range .CurrentNote.Tags}}
          <a class="tag" href="/?tag={{.}}">#{{.}}</a>
        {{end}}
      </div>
      {{end}}
    </article>
  </main>
</body>
</html>
{{end}}

{{define "toolbar"}}
<div class="toolbar toolbar-between">
  <form action="/" method="get" hx-get="/notes" hx-target="#notes-list" hx-push-url="true" class="search-row">
    <input type="search" name="q" value="{{.Filter.Query}}" placeholder="Search title, content, or tags"
      hx-get="/notes" hx-target="#notes-list" hx-trigger="keyup changed delay:300ms" hx-push-url="true">
    {{if .Filter.Tag}}<input type="hidden" name="tag" value="{{.Filter.Tag}}">{{end}}
  </form>
  <a class="secondary-btn" href="/export.md{{if or .Filter.Query .Filter.Tag}}?{{if .Filter.Query}}q={{qescape .Filter.Query}}{{end}}{{if and .Filter.Query .Filter.Tag}}&{{end}}{{if .Filter.Tag}}tag={{qescape .Filter.Tag}}{{end}}{{end}}">Export Markdown</a>
</div>
<div class="tag-row">
  <a class="tag {{if eq .Filter.Tag ""}}active{{end}}" href="/">All</a>
  {{range .Tags}}
    <a class="tag {{if eq $.Filter.Tag .}}active{{end}}" href="/?tag={{.}}" hx-get="/notes?tag={{.}}" hx-target="#notes-list" hx-push-url="true">#{{.}}</a>
  {{end}}
</div>
{{end}}

{{define "form"}}
{{$editing := .EditingNote}}
<h2>{{if $editing}}Edit note{{else}}New note{{end}}</h2>
<p class="muted">Comma-separated tags. Works with HTMX, but also falls back to standard browser form posts.</p>
{{if .ErrorMessage}}<div class="banner error">{{.ErrorMessage}}</div>{{end}}
{{if .SuccessMessage}}<div class="banner success">{{.SuccessMessage}}</div>{{end}}
<form class="note-form" action="{{if $editing}}/notes/{{$editing.ID}}{{else}}/notes{{end}}" method="post" {{if $editing}}hx-put="/notes/{{$editing.ID}}"{{else}}hx-post="/notes"{{end}} hx-target="body" hx-swap="outerHTML show:top">
  {{if $editing}}<input type="hidden" name="_method" value="PUT">{{end}}
  <label>Title<input type="text" name="title" placeholder="Sprint retro ideas" value="{{.FormTitle}}" required></label>
  <label>Content<textarea name="content" rows="8" placeholder="Capture the key idea, decision, or todo." required>{{.FormContent}}</textarea></label>
  <label>Tags<input type="text" name="tags" list="known-tags" placeholder="go, htmx, sqlite" value="{{.FormTags}}" hx-get="/tags/suggest" hx-target="#known-tags" hx-trigger="input changed delay:150ms" hx-include="this"></label>
  <datalist id="known-tags">{{template "tag-options" .Tags}}</datalist>
  <div class="actions">
    <button type="submit">{{if $editing}}Save changes{{else}}Add note{{end}}</button>
    {{if $editing}}<a class="secondary-btn" href="/">Cancel</a>{{end}}
  </div>
</form>
{{end}}

{{define "list"}}
{{if .Notes}}
<div class="notes">
{{range .Notes}}
  {{template "card" .}}
{{end}}
</div>
{{else}}
<div class="empty">
  <h3>No notes yet</h3>
  <p>Try a different search, remove the active tag filter, or create your first note.</p>
</div>
{{end}}
{{end}}

{{define "tag-options"}}{{range .}}<option value="{{.}}"></option>{{end}}{{end}}

{{define "card"}}
<article class="note-card" id="note-{{.ID}}">
  <div class="note-top">
    <div>
      <h3><a class="note-link" href="/notes/{{.ID}}">{{if .Pinned}}📌 {{end}}{{.Title}}</a></h3>
      <p class="meta">Updated {{.UpdatedAt.Format "02 Jan 2006 15:04"}}</p>
    </div>
    <div class="inline-actions">
      <form class="inline-form" action="/notes/{{.ID}}/pin" method="post" hx-post="/notes/{{.ID}}/pin" hx-target="body" hx-swap="outerHTML show:top">
        <button class="secondary-btn" type="submit">{{if .Pinned}}Unpin{{else}}Pin{{end}}</button>
      </form>
      <form class="inline-form" action="/notes/{{.ID}}/duplicate" method="post" hx-post="/notes/{{.ID}}/duplicate" hx-target="body" hx-swap="outerHTML show:top">
        <button class="secondary-btn" type="submit">Duplicate</button>
      </form>
      <a class="secondary-btn" href="/notes/{{.ID}}/edit" hx-get="/notes/{{.ID}}/edit" hx-target="body" hx-swap="outerHTML show:top">Edit</a>
      <form class="inline-form" action="/notes/{{.ID}}" method="post" hx-delete="/notes/{{.ID}}" hx-target="#note-{{.ID}}" hx-swap="delete">
        <input type="hidden" name="_method" value="DELETE">
        <button class="danger-btn" type="submit">Delete</button>
      </form>
    </div>
  </div>
  <p class="content">{{.Content}}</p>
  {{if .Tags}}
  <div class="tag-row">
    {{range .Tags}}
      <a class="tag" href="/?tag={{.}}" hx-get="/notes?tag={{.}}" hx-target="#notes-list" hx-push-url="true">#{{.}}</a>
    {{end}}
  </div>
  {{end}}
</article>
{{end}}`))

func RenderPage(w io.Writer, data PageData) error { return templates.ExecuteTemplate(w, "page", data) }
func RenderList(w io.Writer, data PageData) error { return templates.ExecuteTemplate(w, "list", data) }
func RenderNotePage(w io.Writer, data PageData) error {
	return templates.ExecuteTemplate(w, "note-page", data)
}
func RenderTagOptions(w io.Writer, tags []string) error {
	return templates.ExecuteTemplate(w, "tag-options", tags)
}
