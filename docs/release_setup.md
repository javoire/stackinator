# Release Process

This project uses [semantic-release](https://semantic-release.gitbook.io/) for automated versioning and [GoReleaser](https://goreleaser.com/) for building and distributing releases.

## Automated Releases (Recommended)

Releases are **automatically created** when you merge PRs to `main` using [conventional commits](https://www.conventionalcommits.org/):

### Commit Message Format

- `feat: add new feature` → Minor version bump (0.1.0 → 0.2.0)
- `fix: resolve bug` → Patch version bump (0.1.0 → 0.1.1)
- `perf: improve performance` → Patch version bump
- `docs: update documentation` → No release
- `chore: update dependencies` → No release
- `ci: update workflows` → No release

### Breaking Changes

Add `BREAKING CHANGE:` in the commit body for major version bumps (0.1.0 → 1.0.0):

```
feat: redesign CLI interface

BREAKING CHANGE: command structure has changed
```

### The Workflow

1. **Merge PR to main** with semantic commit messages
2. **Semantic-release automatically**:
   - Analyzes commits since last release
   - Determines version bump
   - Creates and pushes a git tag (e.g., `v0.2.0`)
   - Updates `CHANGELOG.md`
3. **GoReleaser automatically** (triggered by tag):
   - Builds binaries for macOS, Linux, and Windows
   - Creates a GitHub release with artifacts
   - Updates the Homebrew tap at `javoire/homebrew-tap`

### Verify the Release

- Check the **Actions** tab for workflow status
- Verify the **Releases** page has the new version
- Check `javoire/homebrew-tap` for the updated formula

## Manual Release (Alternative)

If you need to create a release manually:

```bash
git checkout main
git pull origin main
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

This triggers the GoReleaser workflow directly.

## Testing Locally

Test the release process without publishing:

```bash
brew install goreleaser
goreleaser check
goreleaser release --snapshot --clean
```

Artifacts will be in `./dist/`.

## Troubleshooting

**Homebrew tap not updating**: Ensure the GitHub token has write access to this repository.

**Build fails**: Check GoReleaser logs in GitHub Actions. Adjust build configuration in `.goreleaser.yml` if needed.
