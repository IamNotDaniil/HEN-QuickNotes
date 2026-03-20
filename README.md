# HEN-QuickNotes

Minimalist note-taking service built with Go, server-side rendering, HTMX interactions, and local JSON persistence for a zero-dependency developer experience.

## Features

- Create, edit, and delete notes directly from the main screen.
- Instant search across titles, note content, and tags.
- Tag filtering with shareable URLs via `hx-push-url`.
- Lightweight SSR UX: the server returns either the full page or an HTML fragment for the notes list.
- Progressive enhancement: create/update/delete flows now degrade gracefully for plain browser requests too.
- Responsive dark UI focused on speed and low JS usage.
- Export the current note list (including active search/tag filters) as Markdown.
- Open a dedicated permalink page for each note via `GET /notes/{id}`.
- Pin important notes so they stay at the top of the list.
- Get live tag autocomplete suggestions while typing in the note form.
- Duplicate an existing note to reuse it as a template or draft.
- Local file persistence via `data/notes.json`, so notes survive restarts without an external database.

## Stack

- Go `net/http`
- HTMX from CDN
- Plain CSS for a fast, dependency-light setup
- File-backed repository with a clean repository interface

## Project structure

- `cmd/server` — application entrypoint.
- `internal/app` — server wiring and dependency initialization.
- `internal/handlers` — HTTP handlers for full pages and HTML fragments.
- `internal/models` — note and filter models.
- `internal/repository` — repository abstractions plus memory/file implementations.
- `internal/services` — business logic and tag parsing.
- `internal/views` — HTML templates for pages and note fragments.
- `static` — CSS assets.
- `migrations` — SQL schema bootstrap for a future SQLite-backed repository.

## Local run

```bash
make run
```

Or without `make`:

```bash
QUICKNOTES_DATA_FILE=./data/notes.json go run ./cmd/server
```

Then open `http://localhost:8080`.

## Developer commands

```bash
make fmt
make test
make build
make clean
```

## Health check

```bash
curl http://localhost:8080/healthz
```

The endpoint returns `200 OK` with body `ok`, which is also used by Docker health checks and deployment probes.

## Docker

```bash
docker build -t quicknotes .
docker run --rm -p 8080:8080 -v $(pwd)/data:/app/data quicknotes
```

Or via Compose:

```bash
docker compose up --build
```

## CI

GitHub Actions runs formatting checks, `go test ./...`, and a build of `./cmd/server` on every push and pull request.

## Persistence model

The running app uses `internal/repository/file.go` to persist notes into a JSON file. Tests still use the in-memory repository for fast isolated execution. A SQL migration is kept in `migrations/001_init.sql` so the app can later move to SQLite without changing the rest of the architecture.

## Export

Use `/export.md` to download notes as Markdown. Active `q` and `tag` filters can be passed through the query string, for example `/export.md?tag=go` or `/export.md?q=htmx`.

## Permalinks

Each note now has a dedicated page at `/notes/{id}`. This gives the project a shareable/read-only note view and lays groundwork for future public-note functionality.

## Pinning

Use the `Pin` / `Unpin` action on a card to keep important notes at the top of the list. Pinned notes are sorted before non-pinned notes and show a `📌` marker in both the list and permalink view.

## Tag autocomplete

The note form now asks `/tags/suggest?q=...` via HTMX while you type into the tags field and updates the datalist with matching existing tags.

## Duplication

Use the `Duplicate` action on a note card or note permalink page to create a quick copy. Duplicates get a ` (copy)` suffix, start as unpinned, and plain browser requests redirect straight to the new note permalink so the copied draft can be reviewed immediately.

## HTMX + browser flow

- `GET /` renders the full page.
- `GET /notes?q=...&tag=...` returns the notes list fragment for HTMX requests and a full page for normal browser requests.
- `POST /notes` validates input, returns inline error banners for HTMX, and uses redirect-after-post for normal browser requests.
- `GET /notes/{id}/edit` re-renders the page with the edit form populated.
- `PUT /notes/{id}` (or `POST` with `_method=PUT`) updates the note.
- `DELETE /notes/{id}` (or `POST` with `_method=DELETE`) removes the note, with `204 No Content` for HTMX and redirect fallback otherwise.
