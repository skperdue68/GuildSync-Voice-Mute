# Authentication and mute recovery

Discord OAuth opens in the system browser. The app generates random, one-use state and validates it before processing the code or denial. The local callback is loopback-only and expires after five minutes. The code is exchanged by the existing GuildSync server, which holds the Discord client secret. No client secret, bot token or database password belongs in this repository or executable.

Saved sessions are revalidated on startup and every 30 seconds while the application runs. The backend rechecks permissions for each mute event. Invalid sessions or failed verification stop local activity. Saved session endpoint fields must match the bundled GuildSync endpoint, so a modified file cannot redirect its token to a different server.

Configuration/session files use restrictive file permissions on Unix and live in the current user's configuration directory. Windows protection follows that directory's account ACLs. The bearer token is stored in a local file, as in GuildSync; this version does not use a platform credential vault. Sign out clears it locally even if remote logout cannot be reached. A server-side token may remain valid until its normal expiry if remote logout fails.

The embedded frontend is served with a content security policy: scripts and styles must come from the bundle (inline allowed because Wails injects its runtime that way), network access is limited to the bundle origin and `guildsync.perdues.me`, and objects, frames and form submissions are refused. The update check sends no account data to GitHub and only opens release pages under the project's own repository; any other URL is rejected before the browser opens.

The client sends `guildsync:voice-mute-hotkey` with only state and session ID. The server derives identity and the bot derives the channel. Heartbeats run every two seconds while a mute is active, and the local auto-unmute timer sends the release. Logout, disconnect and shutdown cancel them. Server session expiry after eight seconds covers an undelivered release event. Reconnection never restarts a request without a new shortcut press.

The bot owns durable moderation/temporary-mute state and recovery. The companion neither decides who to unmute nor bypasses moderator mutes. An unavailable backend or bot fails closed.


## Discord-only access
Standalone login uses /api/auth/discord/voice-token; verification and logout use /api/voice/auth/session and /api/voice/auth/logout. The voice token has its own audience and scope and is rejected by GuildSync endpoints. Socket.IO uses source voice-mute and registers only access/hotkey handlers, without GuildSync data rooms. These sessions require no GuildSync approval; the bot still enforces current Discord membership, allowed roles and rank protection. Backend startup creates separate voice identity/session tables, and ordinary GuildSync signup continues to use guildsync_users unchanged. New voice sessions expire after 30 days. Deploy the matching backend first and sign in again to replace an older saved GuildSync session.
