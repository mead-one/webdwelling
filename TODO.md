# Webdwelling Review To-do List

Findings from a code review on 2026-10-21, ordered by priority. File references point at the code as of commit `c2e55d3`.

## 1. Fix First

- [x] **Replace the hardcoded JWT secret** (`internal/auth/middleware.go:15`). The repo is public, so anyone can sign a token for any `user_id`. Load the secret from an environment variable or a root-only file (systemd `EnvironmentFile`), refuse to start if it's missing or shorter than 32 bytes, and rotate it. Rotating logs everyone out.
- [x] **Add CSRF protection.** Set `SameSite=Strict` (or `Lax`) on the `jwt` cookie and add Echo's `middleware.CSRF()`. Send the token from `bookmarks.ts` and include it in the login form.
- [x] **Make logout a POST** (`routes.go:400`). As a GET, an `<img>` tag on any site can log you out.
- [x] **Set `Secure` on the cookie** whenever the app is served over HTTPS. Use one helper for setting and clearing the cookie, since the same code appears three times in `middleware.go`.
- [x] **Check folder ownership in `AddBookmark` and `EditBookmark`** (`sqlite.go:283`, `:347`). Right now any `folder_id` is accepted, including other users' folders and folders that don't exist. That makes `GetBookmarksByUserID` crash on a nil map entry (`sqlite.go:274`), and the user's pages return 500 until someone edits the database. Also make `GetBookmarksByUserID` skip orphaned entries rather than panic.
- [ ] **Fix the `folder-tree` template name clash.** `bookmarks.html` and `home.html` both define `folder-tree`, and `home.html` is parsed last, so its read-only version is used on `/bookmarks` too. Server-rendered folders lose their action buttons and `data-id`. Rename them, e.g. `folder-tree-edit` and `folder-tree-view`.
- [ ] **Fix the edit dialog loading the wrong bookmark** (`bookmarks.ts:430-431`). It looks up `a.bookmark-title` and `span.bookmark-tags` on `parentFolder` instead of `bookmarkLi`, so it loads the folder's first bookmark, and saving overwrites the clicked one.

## 2. Security

- [ ] **Stop revealing which usernames exist through login** (`sqlite.go:175-195`, `middleware.go:29-35`). An unknown username returns 500 and a wrong password returns 401. Treat `sql.ErrNoRows` as invalid credentials, run bcrypt against a dummy hash when the user doesn't exist so the timing matches, and return the same 401 in both cases.
- [ ] **Rate-limit `/login`.** `middleware.RateLimiter` is enough to start with.
- [ ] **Remove the `innerHTML` interpolation** (`bookmarks.ts:778`, `:1132`). Folder names are inserted unescaped. Build these elements with `createElement` and `textContent`, as the rest of the file does. This becomes stored XSS once public pages show folders to other people.
- [ ] **Validate bookmark URLs on the server.** Accept only `http` and `https`. Without this, a `javascript:` URL set via `href` in JS runs before the page is reloaded.
- [ ] **Stop returning `err.Error()` to clients** (`routes.go:61`, `:124`, `:242`, `:310`). Log the detail and send a generic message.
- [ ] **Remove the debug prints** that log user data (`routes.go:235-236`, `sqlite.go:311`, `sqlite.go:488`).
- [ ] **Add hardening middleware:**
  - `middleware.BodyLimit`
  - maximum lengths for title, URL, tags and folder name
  - `middleware.Secure()` and a CSP header
- [ ] **Pin the JWT signing method** with `jwt.WithValidMethods([]string{"HS256"})`. Consider shorter token lifetimes or a way to revoke tokens, e.g. a per-user `token_version` claim.
- [x] **Make usernames unique on their own.** `UNIQUE(username, email, public_slug)` allows duplicate usernames whenever `email` is NULL. Add `UNIQUE(username)`, plus `UNIQUE(email)` and `UNIQUE(public_slug)` if those should be unique too.
- [ ] **Reconsider the `prevent_username_public_slug_clash` trigger** (`sqlite.go:100-112`). The public page column exists so that loading a public page doesn't reveal whether a username exists. The trigger undoes that: when someone picks a public page name, `'Public slug taken by existing username'` tells them that name is a username. If public pages are only ever addressed by this column and usernames never appear in URLs, the two namespaces never meet, so the trigger can go. The column then just needs its own `UNIQUE`. If a clash rule really is needed, return a generic "name unavailable" error. The trigger also only runs on `INSERT`, so it doesn't apply when the value is changed later.
- [x] **Delete `web/static/bookmarks.js.bak`.** It's committed and served publicly by `e.Static`.

## 3. Correctness

- [x] **Use `cfg.Port`.** `main.go:38` always listens on `:1323`.
- [ ] **Fix the shared `cases.Caser` data race** (`routes.go:41`). It isn't safe to use from several goroutines. Compute page titles once at startup.
- [ ] **URL-encode request bodies in `bookmarks.ts`.** Use `new URLSearchParams({...})`. Currently URLs containing `&` or `#` get cut off and `+` turns into a space.
- [ ] **Stop the edit dialog rewriting relative URLs** (`bookmarks.ts:466`). Read `getAttribute("href")` instead of `.href`, which the browser has already resolved, so `example.com` doesn't come back as `https://host/example.com`.
- [ ] **Fix the schema** (`sqlite.go:57-97`):
  - The foreign keys reference a table `folders` that doesn't exist; it should be `bookmark_folders`.
  - Enable `PRAGMA foreign_keys = ON`, e.g. with `?_foreign_keys=on` in the DSN.
  - `UNIQUE(user_id, name)` on folders blocks same-named folders under different parents. Change it to `UNIQUE(user_id, parent_folder_id, name)` and remember that NULL parents are distinct from each other.
  - Add `created_at` to `bookmarks`.
  - Select and honour folder `public` in `GetBookmarksByUserID`. It's never selected now, so it's always `false`.
- [ ] **Add a schema migration mechanism**, e.g. `PRAGMA user_version` plus ordered migration steps. `CREATE TABLE IF NOT EXISTS` never changes an existing database, so none of the schema fixes, and none of the renames below, will reach a deployed DB without it.
- [x] **Return folders in a stable order** (`sqlite.go:242`). The tree is built by ranging over a map. Add `ORDER BY name` and build the tree in query order.
- [x] **Handle rows properly:** `defer rows.Close()` and check `rows.Err()` everywhere.
- [ ] **Make `DeleteBookmarkFolder` one transaction** (`sqlite.go:526-582`), ideally a single recursive CTE. It currently deletes child bookmarks twice and can leave a half-deleted tree.
- [ ] **Make ownership checks atomic.** Replace each SELECT-then-UPDATE with `UPDATE … WHERE id = ? AND user_id = ?` and check `RowsAffected`. In `MoveBookmarkFolder`, check ownership before `FolderIsDescendant` runs.
- [ ] **Return correct status codes:** 400 for bad input, 403/404 for other users' records. For JSON endpoints, return 401 JSON rather than redirecting to the login page.
- [ ] **Fix `EditBookmark` returning `CreatedAt` as the current time.**
- [ ] **Fix the folder-select tree for new or moved folders** (`bookmarks.ts:1128-1144`). The `<ul>` and `<li>` are nested in the reverse of the template's structure, and `moveBookmarkFolderToFolder` leaves the child `<ul>` behind.
- [ ] **Smaller frontend fixes:**
  - Don't turn on drag and drop on the read-only home page.
  - Check `data.success` in `submitDeleteFolder`.
  - Add a `.catch` to the rename request.
  - Remove `onsubmit` from the `<button>`.
  - Remove the `=== null` checks on `input.value`, which can never be true.
  - Include `basePath` in the `<form action>` attributes.
  - Pass `basePath` when rendering `error.html`.

## 4. Duplication and Simplification

- [ ] Remove the duplicate `/bookmarks/move-folder` route (`routes.go:327`).
- [ ] Pull the `"null"`/`""` → `*int` form parsing into a `parseOptionalInt(c, key)` helper. It's currently repeated five times.
- [ ] Put `json` tags on `database.Bookmark` and `database.BookmarkFolder`, then delete `BookmarkResponse` (declared twice) and `BookmarkFolderResponse`.
- [ ] Merge the three template-route closures in `routes.go:436-464` into one handler with different wrappers.
- [ ] Build both nav lists once at startup instead of re-reading every template on every request (`templates.go:41`). This also removes `defer f.Close()` from inside loops (`templates.go:65`, `routes.go:425`).
- [ ] Combine the repeated open/close modal code in `bookmarks.ts`, and run its null checks before the values are used.

## 5. Naming, Standards and Tooling

- [x] **Rename `users.subdirectory`.** The name doesn't describe what the column is for, and it suggests a filesystem path. Suggested: `public_slug`, used as `/{public_slug}`. Also rename:
  - the Go field `Subdirectory` → `PublicSlug`
  - the `CreateUser` parameter
  - the trigger, if it's kept
  - the error strings

  Do this through the migration mechanism above (`ALTER TABLE users RENAME COLUMN subdirectory TO public_slug`). SQLite 3.25+ updates the trigger body automatically.
- [x] **Replace `SELECT *` in `GetUserByUsername`** with an explicit column list, so that adding or renaming columns doesn't silently break the `Scan`.
- [x] **Run `gofmt` and `go vet`, and add `staticcheck`.** Specifics:
  - ~~merge the two `import` blocks in `routes.go`~~
  - ~~start error strings in lowercase~~
  - ~~wrap errors with `%w`~~
  - write `x := …` instead of `var x T = …`
  - ~~use the standard library `slices` instead of `golang.org/x/exp/slices`~~
- [ ] **Fix the Makefile:** `go vet ./...`, `go test ./...`, and `go build -o $(BINARY_PATH) ./cmd/server`. Add tests, starting with the database layer against an in-memory SQLite.
- [ ] **Commit `package.json` and its lockfile** so the TypeScript version is pinned. Change the tsconfig `module` to `none` or `es2020` for a browser script.
- [ ] **Make the paths configurable:** data directory, templates, static files and the config file location.
- [ ] **Harden `system/webdwelling.service`:**
  - `NoNewPrivileges=yes`
  - `ProtectSystem=strict`
  - `ProtectHome=yes`
  - `PrivateTmp=yes`
  - `ReadWritePaths=/opt/webdwelling/data`
  - ~~`EnvironmentFile=` for the JWT secret~~ (done)
