# Security policy

## Supported versions

Security fixes are provided for the latest published release. Pre-release builds from the `main` branch are supported on a best-effort basis.

## Reporting a vulnerability

Please use the repository host's private security-advisory feature. Do not open a public issue for an unpatched vulnerability, and never attach a private SQLite database, signing certificate, token, or credential file.

Include the affected version, operating system, reproduction steps, impact, and any suggested mitigation. Use a minimal synthetic database when a reproducer needs data.

The project does not require secrets at runtime. Release-only signing credentials belong in the repository host's encrypted Actions secrets and must never be committed.
