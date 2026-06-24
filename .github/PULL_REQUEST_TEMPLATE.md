<!-- Use a Conventional Commit title, e.g. "feat(providers): add X" -->

## What & why

<!-- What does this change and why? Link any issue. -->

## Checklist

- [ ] `go build ./... && go vet ./... && go test ./...` pass
- [ ] Conventional Commit title; commits signed off (`git commit -s`)
- [ ] If the command tree / flags / exit codes / providers changed: schema golden
      regenerated (`RIVR_UPDATE_GOLDEN=1 go test ./internal/cli/...`) and the embedded
      `SKILL.md` + docs + landing copy updated in this PR
- [ ] Output schema changes are append-only (no field renames/removals)
- [ ] New/changed provider behavior covered by an `httptest` fixture
- [ ] CHANGELOG `[Unreleased]` updated
