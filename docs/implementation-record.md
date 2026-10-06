# Implementation and verification record

The design and plan were approved in the conversation on October 5, 2026. Execution used the native approach with independent final reviews.

- GuildSync profile controls: POC PR 86, based on voice-mute PR 85. Role tests passed 3/3; web and desktop frontend builds passed; independent review found no actionable issues.
- Companion authentication/settings: implemented separate application storage, one-use OAuth state, bounded callback listener and HTTP requests, approved-account verification, restrictive session writes and logout cleanup.
- Companion transport: six frontend tests passed, covering heartbeat/release, disconnect without replay, disable/rearm, denied requests and obsolete authentication verification.
- Native listeners: Windows production Wails build passed. Linux and macOS CGO-disabled integration builds passed; these do not verify their native listeners. Native CI builds and real desktop/Discord checks are required before claiming those platforms have been tested.
- Independent companion review found four issues: first hold ignored, stale event after disable, stale verification after logout, and native permission failure without status. All were corrected; regression coverage and the full nine-test Go race suite passed. Go vet passed.
- Frontend dependency lockfile is isolated, with Vite 8.3.2 and socket.io-client 4.8.1. Fresh dependency installation and audit reported zero vulnerabilities.
- Tagged native build/release workflow and user/platform/security documentation are included. No Google Apps Script changes were made.

## Rulings made during execution

- Use native global key-state polling every 25 milliseconds on Windows and macOS rather than installing an event hook/tap. This supports the requested hold behavior without an OS callback that needs lifecycle management or suppresses keys. Cost: an exceptionally short combination lasting less than one polling interval can be missed; this feature is intended for held shortcuts. Wayland uses the portal's activation/deactivation events.
- Use current Vite 8.3.2 in the standalone project rather than GuildSync's older Vite dependency. The first reused lockfile referenced the existing client's dependencies and was discarded; the standalone installation is independent. Cost: source builds require a current supported Node version; Node 24 is documented and used in CI. Existing GuildSync dependencies were preserved.
- Bootstrap the new public repository with a README-only master branch, then submit all application files through a feature branch and PR. Cost: the app cannot be built from the default branch until the user merges that PR.

## Remaining verification limits

Live Discord login, account approval, actual channel mutes, macOS permissions, X11 observation and Wayland portal binding have not been exercised against production accounts/desktops here. The bot's existing permissions, durable state and expiry remain authoritative. Windows executable build success is not an end-to-end Discord test.
