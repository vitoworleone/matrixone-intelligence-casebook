# agent-runtime-v2 Trace Design

## Purpose

`agent-runtime-v2` needs a built-in trace facility for every runtime run. The
trace must be generic runtime infrastructure, not an Explore-specific feature.
Explore, workflow-agent migration work, data agents, and future A2A agents
should all use the same trace model.

The trace facility records what the runtime did, in order, with enough context
to debug a failed or low-quality answer without reconstructing state from
frontend SSE dumps and service logs.

## Non-goals

1. Do not design Explore-only tables, event names, or APIs.
2. Do not encode RAG, NL2SQL, or other domain-specific workflows in the core
   runtime schema.
3. Do not replace A2A streaming. A2A remains the live protocol; trace storage is
   for persistence, reload, and diagnosis.
4. Do not expose raw chain-of-thought as a first-class trace field.
5. Do not use trace persistence to hide runtime failures. Recording trace must
   not turn a failed run into a successful run.

## Current Gap

The current Explore A2A path persists the final A2A event snapshot into chat
message `modified_response` after a run finishes. That is useful for frontend
reload, but it is not a structured diagnostic store:

1. Events are written only at completion.
2. Large event arrays are hard to query by run, turn, LLM call, tool call, or
   artifact.
3. Runtime-level facts such as parent/child event order, TTFT, token usage,
   cancellation, and tool latency are not normalized.
4. Debugging still requires joining frontend JSON, A2A events, and catalog logs
   by hand.

`agent-runtime-v2` should own this as a runtime capability.

## Model

The runtime trace has three layers:

1. **Run**: one agent execution under a trace id. Each runtime task allocates a
   new trace id; a conversation/session never reuses a trace id across tasks.
2. **Event**: ordered runtime facts inside the run.
3. **Blob**: optional large payloads attached to an event.

The core schema is intentionally generic. Tool-specific payloads live in
`payload_json` or blobs. For example, a RAG search is still a generic
`tool.call.completed` event whose `name` is `search_rag_chunks`; RAG route
details are tool payload, not core event types.

## Runtime API

`agent-runtime-v2` exposes a recorder interface:

```go
type TraceRecorder interface {
    StartRun(ctx context.Context, run TraceRun) (context.Context, error)
    RecordEvent(ctx context.Context, event TraceEvent) error
    AttachBlob(ctx context.Context, blob TraceBlob) error
    FinishRun(ctx context.Context, result TraceRunResult) error
}
```

The runtime passes trace scope through `context.Context`. Runtime-owned wrappers
record standard events:

1. A2A task/run lifecycle.
2. Turn lifecycle.
3. LLM call lifecycle and metrics.
4. Tool call lifecycle.
5. Artifact creation.
6. Runtime transitions.
7. Cancellation, timeout, and errors.

Agent implementations should not write trace database tables directly. They may
attach structured tool payloads through the runtime tool result path.

## Standard Event Types

Core event types:

```text
run.started
run.completed
run.failed
run.canceled

turn.started
turn.completed
turn.failed

llm.call.started
llm.output.delta
llm.call.completed
llm.call.failed

tool.call.started
tool.call.completed
tool.call.failed

artifact.created
runtime.transition
runtime.compaction
runtime.error
```

Domain-specific operations must use these standard event types. Examples:

1. `search_rag_chunks` is `tool.call.started/completed/failed`.
2. `run_sql` is `tool.call.started/completed/failed`.
3. `update_plan` is `tool.call.started/completed/failed`; its plan payload is
   tool output.
4. Parser or embedding calls inside a tool may be included in the tool payload,
   or later represented as nested runtime events if the runtime owns those
   clients.

## Run Table

Trace tables are tenant database tables. The recorder must resolve the tenant
connection from `workspace_id` at run start and use that same trace context for
events, blobs, finish updates, and trace queries.

Fresh install schema:

```sql
CREATE TABLE IF NOT EXISTS agent_runtime_trace_run (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    trace_id VARCHAR(64) NOT NULL,
    runtime_version VARCHAR(32) NOT NULL DEFAULT 'v2',
    agent_code VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL DEFAULT '',
    session_id VARCHAR(128) NOT NULL DEFAULT '',
    task_id VARCHAR(128) NOT NULL DEFAULT '',
    context_id VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL,
    started_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    ended_at TIMESTAMP(6) NULL,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    input_summary TEXT,
    output_summary TEXT,
    error_message TEXT,
    metadata_json JSON,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uk_agent_runtime_trace_run_trace_id (trace_id),
    KEY idx_agent_runtime_trace_run_workspace_session (workspace_id, session_id, id),
    KEY idx_agent_runtime_trace_run_workspace_task (workspace_id, task_id),
    KEY idx_agent_runtime_trace_run_agent_time (agent_code, started_at)
);
```

## Event Table

Fresh install schema:

```sql
CREATE TABLE IF NOT EXISTS agent_runtime_trace_event (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    trace_id VARCHAR(64) NOT NULL,
    event_id VARCHAR(128) NOT NULL,
    parent_event_id VARCHAR(128) NOT NULL DEFAULT '',
    seq BIGINT NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT '',
    started_at TIMESTAMP(6) NULL,
    ended_at TIMESTAMP(6) NULL,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    input_summary TEXT,
    output_summary TEXT,
    error_message TEXT,
    payload_json JSON,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uk_agent_runtime_trace_event_event (trace_id, event_id),
    UNIQUE KEY uk_agent_runtime_trace_event_seq (trace_id, seq),
    KEY idx_agent_runtime_trace_event_type (trace_id, event_type),
    KEY idx_agent_runtime_trace_event_created (created_at)
);
```

## Blob Table

Fresh install schema:

```sql
CREATE TABLE IF NOT EXISTS agent_runtime_trace_blob (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    trace_id VARCHAR(64) NOT NULL,
    event_id VARCHAR(128) NOT NULL,
    blob_kind VARCHAR(64) NOT NULL,
    content_type VARCHAR(128) NOT NULL DEFAULT 'application/json',
    content_json JSON,
    content_text MEDIUMTEXT,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_agent_runtime_trace_blob_event (trace_id, event_id, blob_kind)
);
```

`content_text` uses `MEDIUMTEXT` because raw LLM requests, tool results, and
large artifact bodies can exceed the roughly 64KB capacity of `TEXT`. Summary
fields remain `TEXT`.

## LLM Event Payload

`llm.call.completed` payload should include:

```json
{
  "model": "qwen3-max",
  "backend_id": 123,
  "endpoint_id": 456,
  "input_tokens": 6050,
  "output_tokens": 98,
  "reasoning_tokens": 0,
  "cached_tokens": 5830,
  "cache_hit_ratio": 0.96,
  "ttft_ms": 571,
  "duration_ms": 2400,
  "finish_reason": "tool_calls"
}
```

Large request and response bodies should be stored as blobs:

1. `blob_kind = llm.request`
2. `blob_kind = llm.response`

Raw body access should be an explicit diagnostic API path, not part of the
default trace list response.

## Tool Event Payload

`tool.call.completed` payload should include:

```json
{
  "tool_name": "search_rag_chunks",
  "call_id": "call_abc",
  "argument_summary": {
    "query": "backend engineer candidate",
    "top_k": 8
  },
  "result_summary": {
    "row_count": 8,
    "artifact_ids": ["artifact_1"]
  },
  "domain_payload": {
    "routes": ["fulltext", "vector"],
    "candidate_count": 8,
    "chunk_count": 8
  }
}
```

The runtime does not interpret `domain_payload`. It stores it and exposes it to
diagnostic tooling.

## Query API

Catalog should expose generic runtime trace APIs:

```text
GET /api/v1/workspaces/:id/agent-runtime/traces
GET /api/v1/workspaces/:id/agent-runtime/traces/:trace_id
GET /api/v1/workspaces/:id/agent-runtime/traces/:trace_id/events
GET /api/v1/workspaces/:id/agent-runtime/traces/:trace_id/blobs/:event_id
```

List filters:

```text
agent_code
session_id
task_id
status
event_type
from
to
limit
cursor
```

Default APIs return summaries and structured payloads. Blob bodies are fetched
through the blob endpoint.

## Storage and Retention

Trace rows are diagnostic data and can grow quickly. The first implementation
should support:

1. Configurable enable/disable flag.
2. Configurable retention duration.
3. Configurable maximum blob size per event.
4. Explicit error if a blob exceeds the configured limit.

Do not silently truncate payloads unless the trace event records that truncation
as structured metadata.

## Integration Points

Initial implementation status:

1. Fresh tenant schema defines `agent_runtime_trace_run`,
   `agent_runtime_trace_event`, and `agent_runtime_trace_blob`.
2. Auto-upgrade offset `1.0.0/41` creates the same three trace tables in every
   tenant database for existing deployments.
3. `agent-runtime-v2` owns the generic `TraceRecorder` interface and records
   run, turn, LLM, tool, compaction, and error events.
4. Every LLM call stores the raw provider request and raw provider response as
   `llm.request` and `llm.response` blobs. A blob size limit is enforced as an
   explicit runtime error; payloads are not silently truncated.
5. Catalog wires a DB-backed recorder into the Explore runtime-v2 agent.
6. Explore A2A task metadata carries the generated `trace_id`, so the frontend
   can display it and operators can query the persisted trace.

Further implementation should be phased:

1. Add schema definitions for fresh install.
2. Add one new auto-upgrade offset that creates the three trace tables.
3. Add `TraceRecorder` interfaces and no-op recorder in `agent-runtime-v2`.
4. Add catalog DB-backed recorder.
5. Wire recorder into generic A2A agent construction.
6. Wrap runtime LLM calls.
7. Wrap runtime tool calls.
8. Wrap artifact emitter.
9. Add generic trace query APIs.
10. Update frontends and diagnostic scripts to consume trace APIs.

All schema work must follow the catalog auto-upgrade rules: fresh install schema
and upgrade handler must be updated together, and the branch must introduce only
one new offset.

## Relationship To A2A

A2A remains the live wire protocol. Trace persistence records the runtime facts
that produced A2A task, status, message, and artifact updates.

The trace model must not invent a parallel protocol. When possible, trace events
should reference A2A ids:

1. `task_id`
2. `context_id`
3. `message_id`
4. `artifact_id`
5. tool call id

This keeps frontend live state and backend diagnostics joinable without forcing
the frontend to send a large conversation payload back to the server.

## Security

Trace storage may contain prompts, tool arguments, SQL text, retrieved snippets,
and provider responses. API access must be scoped by workspace and user
permissions. Raw blobs should be treated as diagnostic data and should not be
returned by default list APIs.

Sensitive data redaction is a separate explicit feature. The first
implementation must not claim data is redacted unless the redaction path is
implemented and tested.
