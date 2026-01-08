# Changelog

## 0.1.1

- Added `-c/--copy` flag to copy the current code to the clipboard for:
  - `totp get`
  - `totp temp`
  - `totp add` (copies the "Current code" output)
- When copying succeeds, prints a masked confirmation like `12**** (copied)`.
- When copying fails, prints the full code with a note like `123456 (copy failed)`.

## 0.1.0

- Forked from a macOS-only TOTP CLI (Keychain-based) and made it cross-platform.
- Migrated keyring storage to `github.com/zalando/go-keyring` (macOS Keychain, Linux Secret Service/DBus, Windows Credential Manager).
- Changed keyring service name to `totp` (no legacy read/migration support; users must re-add secrets after upgrading).
- Implemented cross-platform `totp list` using an index file at `~/.totp.json` (names only).
- Added auto-healing for the index file on `totp list` by removing entries missing from the keyring.
- Added duplicate-name handling for `totp add` and `totp scan`: prompts repeatedly until an unused name is provided.
- Added secret validation:
  - secrets must be valid Base32
  - spaces are ignored and input is normalized to uppercase
- Added UX sanity-check on `totp add`: prints `Current code: <code>` before storing.
- Added `totp completion` command to generate shell completions for:
  - bash
  - zsh
  - fish
  - PowerShell
- Overhauled README with detailed usage examples, platform notes, security considerations, troubleshooting, and completion setup.
- Renamed/positioned project as `totp-cli` and updated repository references.
- Updated Go module path to `github.com/munim/totp-cli`.
- Reworked GitHub Actions:
  - removed standalone CI workflow
  - added tag-driven release workflow for tags matching `x.x.x`
  - builds and uploads release artifacts for Linux/Windows and a macOS universal binary (arm64+amd64 via `lipo`)
  - runs gofmt check, `go test`, and `go vet` as part of the release pipeline.
