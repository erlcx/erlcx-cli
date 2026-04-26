# Contributing

Thanks for your interest in contributing to ERLCX CLI.

## Getting Started

1. Fork the repository.
2. Create a branch for your change.
3. Make your changes.
4. Run the relevant tests.
5. Open a pull request.

Keep pull requests focused. A small change that is easy to review is better than a large change that mixes unrelated work.

## Code Quality

- Keep behavior predictable and easy to understand.
- Prefer clear errors over silent failures.
- Add tests for new behavior and bug fixes.
- Avoid unrelated formatting or refactors in feature pull requests.
- Do not commit generated binaries, local config, logs, tokens, or secrets.

## Roblox Account Safety

ERLCX CLI must never ask users for sensitive Roblox credentials.

Do not add code that:

- Requests a Roblox password.
- Requests or stores `.ROBLOSECURITY`.
- Scrapes authenticated Roblox pages.
- Stores OAuth tokens in project files.
- Deletes Roblox assets without an explicit user command and clear confirmation.

Authentication should use official Roblox-supported flows.

## Documentation

Update documentation when you change commands, configuration, generated files, or user-visible behavior.

Keep examples practical and copy-pasteable where possible.

## Pull Request Checklist

Before opening a pull request, check that:

- The change has a clear purpose.
- Tests pass.
- New behavior is tested.
- Documentation is updated when needed.
- No secrets or local-only files are committed.
