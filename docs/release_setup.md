# Release Process

This project uses [GoReleaser](https://goreleaser.com/) for automated releases. When you push a tag, GitHub Actions automatically builds binaries and updates the Homebrew tap.

## Prerequisites

Create a public repository on GitHub named `homebrew-tap`:

- Repository: `javoire/homebrew-tap`
- Make it public
- No need to initialize with files

GoReleaser will create the formula automatically on the first release.

## Creating a Release

1. Ensure all changes are committed and pushed to main:

   ```bash
   git checkout main
   git pull origin main
   ```

2. Create and push an annotated tag:

   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

3. GitHub Actions will automatically:

   - Build binaries for macOS, Linux, and Windows
   - Create a GitHub release with all artifacts
   - Update the Homebrew tap at `javoire/homebrew-tap`

4. Verify the release:
   - Check the **Actions** tab to watch the workflow
   - Verify the GitHub **Releases** page has the new release
   - Check `javoire/homebrew-tap` for the updated formula

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
