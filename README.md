# totp-cli

A small, cross-platform TOTP CLI that stores shared secrets in your OS keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service).

- Binary name: `totp`
- Repository: `github.com/munim/totp-cli`

## Features

- Store TOTP secrets securely in the **system keyring** (via `github.com/zalando/go-keyring`).
- Manage entries by name:
  - `totp add <name>`: add a Base32 secret (spaces allowed)
  - `totp scan <name> <image>`: import from an `otpauth://totp/...` QR code
  - `totp get <name>`: print the current 6-digit code
  - `totp delete <name>`: remove an entry
  - `totp list`: list registered entry names
  - `totp temp`: generate a code without storing anything
- Shell completion generation: bash, zsh, fish, PowerShell.

## How it works

### Storage

- **Secrets** are stored in the OS keyring under:
  - service: `totp`
  - user: `<name>`
- `totp list` is backed by a local index file:
  - path: `~/.totp.json`
  - contents: **names only** (no secrets)

On `totp list`, the index is **auto-healed** by removing entries that no longer exist in the keyring.

### Secret validation

When you type/paste a secret:

- spaces are ignored (useful for copying from apps that display grouped Base32)
- input is normalized to uppercase
- it must decode as **Base32** (RFC 4648 alphabet)

For `totp add`, `totp` also prints a `Current code: ...` line before storing so you can quickly sanity-check.

## Installation

### Prebuilt binary (macOS universal)

A single static universal binary is published for both Apple Silicon and Intel Macs.

```bash
curl -LO "https://github.com/munim/totp-cli/releases/download/v1.1.3/totp"
chmod +x totp
```

## Quick start

```console
$ totp scan google ./image.jpg
Given QR code successfully registered as "google".

$ totp list
google

$ totp get google
123456
```

## Commands (examples)

### `totp add <name>`

Adds a new entry to the system keyring and records its name in `~/.totp.json`.

```console
$ totp add github
Type secret: JBSW Y3DP EHPK 3PXP
Current code: 123456
Given secret successfully registered as "github".
```

If the name already exists, `totp` will keep prompting until you provide a new, unused name.

### `totp get <name>`

```console
$ totp get github
123456
```

### `totp list`

```console
$ totp list
github
google
```

### `totp delete <name>`

```console
$ totp delete github
Successfully deleted "github".
```

### `totp scan <name> <image>`

Scans an image file containing an `otpauth://totp/...` QR code.

```console
$ totp scan google ./image.jpg
Given QR code successfully registered as "google".
```

If decoding fails with certain QR images, try enabling the PURE_BARCODE hint:

```console
$ totp scan --barcode google ./image.jpg
Given QR code successfully registered as "google".
```

### `totp temp`

Generate a code from a secret without storing anything.

```console
$ totp temp
Type secret: JBSWY3DPEHPK3PXP
123456
```

## Shell completion

`totp` can generate completion scripts for common shells:

```bash
totp completion [bash|zsh|fish|powershell]
```

### Bash

If you have `bash-completion` installed, one common location is:

```bash
totp completion bash > /usr/local/share/bash-completion/completions/totp
```

You may need to restart your shell.

### Zsh

The simplest setup is to source it in your `~/.zshrc`:

```zsh
source <(totp completion zsh)
```

### Fish

Fish looks for completions in `~/.config/fish/completions/`:

```fish
mkdir -p ~/.config/fish/completions

totp completion fish > ~/.config/fish/completions/totp.fish
```

### PowerShell

Add this to your `$PROFILE`:

```powershell
Invoke-Expression ((totp completion powershell) -join [Environment]::NewLine)
```

## Platform requirements / notes

### macOS

Uses Keychain. No extra dependencies.

### Linux

Uses the **Secret Service** API over **DBus** (e.g. GNOME Keyring, KWallet via Secret Service).

If you do not have a keyring daemon/session, `totp` may fail to store/retrieve secrets.

### Windows

Uses Windows Credential Manager.

## Security considerations

- Secrets are stored in the system keyring and not in plaintext files.
- `~/.totp.json` contains **names only**, but it can still reveal which services you use.
- `totp add` prints the derived “current code” to stdout. Avoid running it where your terminal output is logged/recorded.

## Troubleshooting

- **"Invalid secret (expected Base32)"**: make sure you pasted the Base32 secret (not a QR URL) and that it only contains A–Z and 2–7. Spaces are OK.
- **"Given name is not found"** (`totp get <name>`): the entry does not exist in the keyring. Use `totp list` to see indexed names.
- **Linux keyring errors**: ensure you have a Secret Service compatible keyring and a working DBus session.

## Development

```bash
# format
go fmt ./...

# tests
go test ./...

# static analysis
go vet ./...

# build
./build
```

CI runs builds/tests on macOS, Linux, and Windows.

## Notes

- Upgrade note: older versions stored secrets in a macOS-only way; after upgrading, you may need to re-add your secrets.

&nbsp;

--------
*totp-cli* is primarily distributed under the terms of both the [Apache
License (Version 2.0)] and the [MIT license]. See [COPYRIGHT] for details.

[MIT license]: LICENSE-MIT
[Apache License (Version 2.0)]: LICENSE-APACHE
[COPYRIGHT]: COPYRIGHT
