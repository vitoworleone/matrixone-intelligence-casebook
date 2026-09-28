# Product

This folder holds the public product artifacts that accompany the casebook.

## Included

- [MOI Platform Prototype](moi-platform-prototype/) — a static, interactive prototype covering the data platform, workflow, knowledge, Agent, governance, and application surfaces.
- [CanvasFlow workflow case study](canvasflow-workflow/) — interaction model, implementation boundaries, and acceptance questions, with attribution to Bai-009.
- [Workflow interaction demo](moi-platform-prototype/videos/workflow-interaction-demo.mp4) — a video walkthrough of CanvasFlow by Bai-009.
- [Prototype synchronization guide](moi-platform-prototype-sync.md) — how to check for and merge updates from `moi-prototype/html`.

## Conventions

- Keep each product artifact in its own named directory.
- Place interactive prototypes or design exports directly in that directory, with an `index.html` entry point when applicable.
- Add `app/` or `docs/` inside a product directory only when that artifact needs runnable code or implementation documentation.
- Do not add a repository-wide shared package, test, evaluation, or data directory unless an implementation genuinely requires it.
