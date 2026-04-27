# PR Workflow

Mandatory process for every change in this repository.
All rules are authoritative.

---

## 1. Branch naming

See `docs/git-convention.md` section 2 for the full branch naming rules.

---

## 2. Step-by-step process (MANDATORY)

### Step 1 — Branch from main

```sh
git checkout main && git pull
git checkout -b feat/<name>
```

### Step 2 — Implement

Follow `docs/go-code-convention.md` and `docs/go-logging-convention.md`.

### Step 3 — Write tests, then run the checklist

Every item in `docs/pre-pr-checklist.md` MUST be green before Step 4.

### Step 4 — Commit

Follow `docs/git-convention.md`. One logical change per commit.

### Step 5 — Open PR

```sh
gh pr create --title "type(scope): description" --body "$(cat <<'EOF'
## Summary
- <what changed and why>

## Test plan
- [ ] <what was tested>
EOF
)"
```

### Step 6 — Run PR review skill

Invoke the `pr-review` skill. Address every finding. Do not merge with open findings.

### Step 7 — Merge (squash)

```sh
gh pr merge --squash
```

The squash commit message MUST follow `docs/git-convention.md`.

---

## 3. PR body rules

- Summary: 2–4 bullets on what changed and why it changed.
- Test plan: checkbox list of exactly what was exercised.
- MUST NOT mention any AI tool or assistant by name.
- MUST NOT reference internal conversation context or session details.

---

## 4. Forbidden in any PR

- Commented-out code left in place.
- Unresolved `TODO` or `FIXME` in changed files.
- Debug `fmt.Println` added during development.
- Any mention of AI tools or assistants by name.
