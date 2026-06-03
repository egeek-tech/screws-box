## Summary

<!-- What does this change do, and why? -->

Closes #

## Type of change

<!-- Pick the type that matches what the change actually IS — never inflate or
     downgrade it to manipulate release behavior. The type prefix on your commits
     drives release-please. -->

- [ ] `feat` — new capability *(cuts a minor release)*
- [ ] `fix` — corrects a defect *(cuts a patch release)*
- [ ] `perf` — performance improvement
- [ ] `deps` — dependency change
- [ ] `docs` — documentation
- [ ] `ci` / `chore` / `refactor` / `test` / `build` / `style` — no release on its own

## Breaking changes

<!-- If this breaks existing behavior, config, or data, describe it here and add a
     `BREAKING CHANGE:` footer to the commit body. Otherwise: "None". -->

None

## Checklist

- [ ] Commit messages are valid Conventional Commits, and the type honestly reflects the change
- [ ] `go test ./... -race` passes
- [ ] `pre-commit run --all-files` is clean (gofmt, golangci-lint, hadolint, actionlint)
- [ ] `govulncheck ./...` is clean
- [ ] Tests use testify (`require`/`assert`) with a `t.Run` per case
- [ ] New / changed exported symbols have doc comments
- [ ] README / `.env.example` updated if configuration or behavior changed
