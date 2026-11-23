# Security policy

## Supported versions

Only the newest published release is supported. Fixes ship in a new release, and users are expected to replace the extracted archive rather than patch it in place. Older archives are not maintained.

## Reporting a vulnerability

Report privately. Do not open a public issue, and do not post details in a discussion or pull request.

Use GitHub's private reporting: open the **Security** tab of this repository and choose **Report a vulnerability**. That starts a private security advisory that only the maintainer can read, and it is the preferred channel because the report, discussion and fix can stay private until a release is out.

Please include:

- the version shown in the window title and below the app heading, plus your operating system and version;
- what an attacker could do, and what they would need in order to do it;
- steps to reproduce, or a proof of concept, if you have one.

Never include your Discord token, the contents of `session.json`, your Discord password, or anything else that would let someone else sign in as you. If you think a credential has already leaked, sign out of the app to revoke the session and change the credential at its source.

## Scope

In scope:

- the desktop companion itself: the Go sources, the window frontend, the shortcut, mute and auto-unmute logic;
- its authentication flow, token storage and the requests it sends to GuildSync;
- the release archives and the workflow that builds them, including anything that would let a third party publish or tamper with a release.

Out of scope:

- the GuildSync bot and backend. Report those to the GuildSync project, not here; this repository does not contain that code and cannot fix it.
- problems that require an attacker to already control the machine, the user's account, or a locally edited `settings.json` or `session.json`.
- vulnerabilities in Discord, macOS, Windows, Linux or WebView2 themselves.
- reports produced only by a scanner with no explanation of the impact.

## What to expect

This is a single-maintainer project, so handling is best effort rather than a contractual response time. Expect an initial reply within about a week. If the report is valid, the fix is prepared privately, released, and then described in the advisory; reporters are credited unless they ask not to be.

## Automated checks in this repository

Dependabot keeps the Go modules, the frontend packages and the workflow actions current, and raises alerts and security-update pull requests for known vulnerable dependencies. CodeQL scans the default branch and pull requests, dependency review rejects pull requests that add a vulnerable dependency, and CI runs `govulncheck` over the Go module and `npm audit` over the frontend on every pull request. Secret scanning with push protection is enabled for the repository.
