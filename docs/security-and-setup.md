# Authentication and mute recovery

Discord OAuth opens in the system browser. The app generates random, one-use state and validates it before processing the code or denial. The local callback is loopback-only and expires after five minutes. The code is exchanged by the existing GuildSync server, which holds the Discord client secret. No client secret, bot token or database password belongs in this repository or executable.

Saved sessions are revalidated on startup and every 30 seconds while the application runs. The backend rechecks permissions for each mute event. Invalid sessions or failed verification stop local activity. Saved session endpoint fields must match the bundled GuildSync endpoint, so a modified file cannot redirect its token to a different server.

Configuration/session files use restrictive file permissions on Unix and live in the current user's configuration directory. Windows protection follows that directory's account ACLs. The bearer token is stored in a local file, as in GuildSync; this version does not use a platform credential vault. Sign out clears it locally even if remote logout cannot be reached. A server-side token may remain valid until its normal expiry if remote logout fails.

The embedded frontend is served with a content security policy that refuses objects, frames and form submissions. Script, style and network origins are deliberately not restricted: the bundle is not served from a plain HTTP origin on every platform, and a restrictive policy that turns out wrong in a webview fails closed and leaves a blank window. The window only ever renders its own bundle, so there is no remote content to allowlist. The update check sends no account data to GitHub and only opens release pages under the project's own repository; any other URL is rejected before the browser opens.

The client sends `guildsync:voice-mute-hotkey` with only state and session ID. The server derives identity and the bot derives the channel. Heartbeats run every two seconds while a mute is active, and the local auto-unmute timer sends the release. Logout, disconnect and shutdown cancel them. Server session expiry after eight seconds covers an undelivered release event. Reconnection never restarts a request without a new shortcut press.

The bot owns durable moderation/temporary-mute state and recovery. The companion neither decides who to unmute nor bypasses moderator mutes. An unavailable backend or bot fails closed.


## Discord-only access
Standalone login uses /api/auth/discord/voice-token; verification and logout use /api/voice/auth/session and /api/voice/auth/logout. The voice token has its own audience and scope and is rejected by GuildSync endpoints. Socket.IO uses source voice-mute and registers only access/hotkey handlers, without GuildSync data rooms. These sessions require no GuildSync approval; the bot still enforces current Discord membership, allowed roles and rank protection. Authorization is Discord-role based and has no per-account allowlist: nobody is admitted by being named individually, only by the Discord roles they hold, and a GuildSync account role does not grant mute access.

The bot's settings are `GUILDSYNC_VOICE_MUTE_ENABLED`, `GUILDSYNC_VOICE_MUTE_ALLOWED_ROLE_IDS` (comma-separated role names or numeric IDs; blank allows nobody) and `GUILDSYNC_VOICE_MUTE_RANK_ROLE_IDS` (rank order, lowest to highest; equal and higher ranks are protected). Their defaults live in `NodeJS/GuildSync-Discord-Bot/.env` in the GuildSync repository, and the same values can be overridden live in GuildSync under **Reports and Admin → Administrator Configuration → Voice channel mute**. A saved server setting wins over the environment default and applies without a restart. The companion stores no allowlist and cannot grant access; it only displays the `allowed` flag the backend returns, so a refusal is fixed on the server, not in the client.

Backend startup creates separate voice identity/session tables, and ordinary GuildSync signup continues to use guildsync_users unchanged. New voice sessions expire after 30 days. Deploy the matching backend first and sign in again to replace an older saved GuildSync session.
