# Pre-PR Checklist

Run before opening any pull request. Every item MUST be green.
No partial merges.

---

## 1. Code

- [ ] No anonymous closures used for business logic (see `go-code-convention.md`)
- [ ] No commented-out code
- [ ] No unresolved `TODO` or `FIXME` in changed files
- [ ] No debug `fmt.Println` or `log.Printf` in changed files

## 2. Tests

- [ ] Every new exported function has at least one test
- [ ] Tests cover the happy path
- [ ] Tests cover at least one error or edge case per function

## 3. Commands (all must exit 0)

```sh
go build ./...
go vet ./...
go test ./...
go test -race ./...
staticcheck ./...
```

Install staticcheck if missing:

```sh
go install honnef.co/go/tools/cmd/staticcheck@latest
```

## 4. Provider interfaces

- [ ] No concrete provider type imported inside `internal/loop`
- [ ] Provider injected via interface in every constructor

## 5. Commit hygiene

- [ ] All commits follow `docs/git-convention.md`
- [ ] No commit message mentions Claude, AI, or assistant tooling
- [ ] Each commit is one logical change

## 6. PR body

- [ ] Title is `type(scope): description` format
- [ ] Summary section has 2–4 bullets
- [ ] Test plan section has checkboxes
- [ ] Body contains no mention of Claude, AI, or assistant tooling
