# Development guide

[Project overview](../README.md) · [中文](开发指南.md)

## Run locally

Use Node.js 22.9 or later. Run `npm ci` after cloning or when the lockfile changes. Copy [`.env.example`](https://github.com/Bai-009/canvas-first-workflow/blob/824397ed995dc7e1293a383bf3b90d1bc1cdf63e/.env.example) to `.env`, fill in the model endpoint, key, and name, then run `npm run web`.

The server listens on `127.0.0.1:5174` by default. To use another port, run `PORT=5175 npm run web`. The web application connects the execution and revision agents by default. These agents generate workflow configurations using a real model; they do not execute the data-processing tasks described by the nodes.

The model service must support Chat Completions and tool calling. Planning uses streaming responses. Provider compatibility varies; see the [observations](观察.md) for recorded behavior.

## Build and check

| Command | Purpose |
| --- | --- |
| `npm run web` | Build and start the local application |
| `npm run build` | Generate contract types, check environment boundaries, compile, and copy assets |
| `npm run build:watch` | Build once, then watch source files and runtime assets |
| `npm run typecheck` | Build and check compile-only type regression cases |
| `npm test` | Build and run the Node.js test suite |
| `npm run validate:fixture` | Build and validate the example plan |

CI runs type checking, tests, and fixture validation. The regular test suite uses test doubles and does not require a model API key.

Application source code is TypeScript. Backend and application modules use `.mts`, compiled to `.mjs`. `prototype/plan-data.js` is generated demonstration data; business-code examples inside workflow nodes are also data, not application implementation.

`tsconfig.browser.json` and `tsconfig.server.json` check the browser and server separately. Browser code does not receive Node.js globals, and server code does not receive DOM globals. Shared contracts and data-shape rules live in `shared/`.

A full build clears `dist/` and `.build-tools/`, then regenerates the runtime output. Do not edit generated directories. Prompts, node definitions, schemas, and static pages are copied into the build as well.

Watch mode updates source and assets. Restart the server after changing backend code. After deleting source files or changing schemas, run a full build. Direct Node.js commands should use entry points under `dist/`.

## Two ways to explore

**Live application:** run `npm run web` and use the local HTTP address. Plans, construction, and revisions come from model calls made during the session.

**Offline prototype:** after `npm run build`, open `dist/prototype/index.html`. It does not call a model. Its plans come from recorded model responses; its execution canvas is hand-authored. Use it to explore interaction timing, not to assess execution-agent behavior. The whole `dist/prototype/` directory can be used as a standalone demo.

Regenerate the prototype's plan data with:

```bash
node dist/src/prototype/build-demo-data.mjs
```

## Sessions and local data

| Content | Location |
| --- | --- |
| Sent global messages, planning context, plan versions, latest canvas, configuration, and revision records | Local `.sessions/` directory |
| Generation and revision traces | Local `.runs/` directory |
| Unsent global and node drafts | Current browser, scoped by session |
| Appearance preference | Current browser |
| Model endpoint and credentials | Server-side `.env` |

`.sessions/`, `.runs/`, and `.env` are ignored by Git. Set `SESSION_DIR` to choose another session directory. A storage directory is intended for one local server process. Builds do not remove sessions.

Session files are fully written and synced before atomic replacement. Restarting restores committed results; unfinished planning, construction, or revision is marked interrupted and does not automatically call the model again. Corrupt files are retained and reported. Save failures are shown with a retry option. Browser drafts are not synced across devices and are lost if browser data is cleared.

Switching sessions does not stop work in another session. Deleted sessions can be restored from the sidebar's deleted-items view.

## Command-line tools

Generate a plan:

```bash
npm run plan -- "Extract key fields from new contract PDFs every day and write them to a database"
```

The model may submit a plan or ask for missing information. Asking a question without submitting a plan is a valid outcome.

Run a multi-turn planning session and save its output:

```bash
npm run plan:chat -- --save .runs/my-session
```

Inside the CLI, `/start` begins construction and `/canvas` shows the canvas. `/note r1 your note` adds an annotation; use the identifier from the actual plan. Unlike the web application, CLI construction requires an explicit executor selection:

```bash
EXECUTOR_MODULE=src/executor/executor.mjs npm run plan:chat -- --save .runs/my-session
```

To inspect state-machine behavior with a fixed-response executor:

```bash
EXECUTOR_MODULE=fixtures/doubles/fixed-executor.mjs npm run plan:chat
```

This replaces only the executor. Planning still uses the configured model. Fixed responses test control flow and do not demonstrate real construction capability.

Run the plan validator directly after building:

```bash
node dist/src/plan/validate-plan-proposal.mjs fixtures/contract-processing.plan.json
```

The planning prompt defaults to Chinese. Set `PLAN_PROMPT_LANG=en` to select the English version. Support for `MODEL_REASONING`, including accepted values, depends on the model provider.

## Code map

| Directory | Responsibility |
| --- | --- |
| `web/` | Canvas, node cards, inputs, plan card, sessions, and themes |
| `src/web/` | HTTP endpoints, event streams, and browser-session integration |
| `src/plan/` | Planning agent, multi-turn sessions, plan submission, and validation |
| `src/executor/` | Step construction, whole-workflow revision, and submission tools |
| `src/state-machine/` | Dependency scheduling, context assembly, execution ownership, and commits |
| `src/model/` | Model adapters, streaming assembly, and response validation |
| `src/storage/` | Local session storage |
| `shared/` | Cross-environment types, contracts, and data-shape rules |
| `contracts/`, `nodes/`, `prompts/` | JSON schemas, node catalog, and system prompts |
| `test/`, `fixtures/` | Regression tests, examples, doubles, and observed runs |
| `prototype/` | Standalone offline interaction demo |

The model adapter normalizes some compatible formats, such as omitted function-call types and object-valued arguments. Invalid JSON argument strings are left for the agent's correction loop. Missing required tool identifiers and other interface errors still fail; compatibility does not mean every provider has been tested.

See the [TypeScript migration record](TypeScript迁移.md) for boundary checks and migration details. Earlier design agreements and implementation notes are preserved in the [2026-09-07 README archive](archive/readme-2026-09-07.md). Those historical documents are in Chinese; check old progress claims against the current code.
