# MOI Platform Prototype

This is a static, interactive prototype for an enterprise data and AI platform. It covers data connection and processing, knowledge and search, Agent applications, governance, administration, and the public-facing product site.

## Open locally

Serve this directory with any static HTTP server, then open `index.html`. The prototype uses static mock data only and does not require a backend, credentials, or environment configuration.

## Structure

- `index.html` — prototype entry point
- `data-connection/`, `data-processing/`, `resource-center/` — data platform flows
- `app-dev/` — Agent and application flows
- `account/`, `admin/`, `monitor/`, `user-perm/` — platform management surfaces
- `website/` — public product-site screens
- `images/`, `styles/`, `scripts/` — static resources used by the prototype

The original offline research and document-editing scripts are intentionally not included: they are not needed to run this prototype.

## Upstream synchronization

This directory is maintained as a Git subtree of `moi-prototype/html`. See the
[casebook synchronization guide](../moi-platform-prototype-sync.md) before
updating it. Keep public-demo sanitization and casebook-only changes in this
repository rather than copying files manually.
