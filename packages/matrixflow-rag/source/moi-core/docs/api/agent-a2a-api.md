# Agent A2A API

Catalog exposes one generic A2A entrypoint for all agent implementations. The
caller selects the concrete agent with `agent_code` or `agent_id`; Explore is
only one registered agent, usually selected as `agent_code=explore`.

## Endpoints

```text
GET  /api/v1/agents/card?agent_code=<code>
GET  /api/v1/agents/card?agent_id=<id>
POST /api/v1/agents/a2a
```

Both endpoints require the normal Catalog API key middleware.

## Agent Card

`GET /api/v1/agents/card` returns the selected agent card directly.

Example:

```http
GET /api/v1/agents/card?agent_code=explore
```

Response:

```json
{
  "name": "Matrixflow Explore Agent",
  "version": "v2",
  "protocolVersion": "0.3.0",
  "url": "https://moi.example.test/api/v1/agents/a2a",
  "capabilities": {
    "streaming": true,
    "stateTransitionHistory": true
  },
  "defaultInputModes": ["text/plain"],
  "defaultOutputModes": ["text/plain", "application/json"]
}
```

Generic Card `url` is the Catalog A2A endpoint `/api/v1/agents/a2a`, not
`/api/v1/agents/card/a2a`. Scoped well-known cards still map
`.../.well-known/agent-card.json` to `.../a2a`.

## JSON-RPC Envelope

`POST /api/v1/agents/a2a` uses A2A JSON-RPC without wrapping the payload in an
Explore-specific response shape.

The top-level selector is Matrixflow routing metadata:

```json
{
  "agent_code": "explore",
  "jsonrpc": "2.0",
  "id": "req_001",
  "method": "message/send",
  "params": {
    "message": {
      "parts": [
        {"kind": "text", "text": "对比金盘科技 2024 年和 2025 年的整体经营情况"}
      ]
    },
    "locale": "zh-CN"
  }
}
```

Supported methods:

1. `message/send`
2. `message/stream`
3. `tasks/get`
4. `tasks/cancel`
5. `tasks/resubscribe`

`message/send`, `tasks/get`, and `tasks/cancel` return a JSON-RPC response
whose `result` is an A2A task object.

`params.locale` and the `Accept-Language` header are transport/UI locale
metadata. They may be stored in turn metadata and forwarded to ToolGateway so
tool result display metadata can be localized by the client. They do not decide
the LLM answer language. Agent Runtime v2 instructs the model to write
user-visible assistant text in the same natural language as the user's current
message, unless the user explicitly asks for another language.

## Streaming

`message/stream` and `tasks/resubscribe` return Server-Sent Events.

Each SSE event has:

```text
event: message
id: <sequence>
data: {"jsonrpc":"2.0","id":"req_001","result":{...}}
```

The `id` line is the task event sequence. Clients should keep it so they can
call `tasks/resubscribe` with `afterSeq` after reconnect.

The server may send SSE comment heartbeats such as `: heartbeat` while waiting
for the next A2A event. Clients must ignore comment frames.

The first successful `message/stream` event must contain a task object so the
UI can bind to a stable task id before status and artifact updates arrive.
Subsequent events contain A2A `status-update` or `artifact-update` results.

Example stream request:

```json
{
  "agent_code": "explore",
  "jsonrpc": "2.0",
  "id": "req_stream_001",
  "method": "message/stream",
  "params": {
    "message": {
      "parts": [
        {"kind": "text", "text": "对比金盘科技 2024 年和 2025 年的整体经营情况"}
      ]
    },
    "locale": "zh-CN"
  }
}
```

## Workflow Agent

The workflow DSL agent is registered as `agent_id=workflow`.

It uses `agent-runtime-v2` and exposes a tool-first planner for natural
language to Matrixflow DSL. The LLM does not write directly to Catalog. It must
search/inspect WorkItems, submit a DSL candidate through
`submit_workflow_candidate`, and the backend compiler validates node ids,
versions, graph edges, forms, model bindings, and catalog sink fields before a
candidate artifact is emitted. Once that candidate is accepted, the runtime ends
the turn and the server renders the final summary from the accepted candidate;
it does not ask the model to produce another success message. Acceptance means
the candidate is ready to save, not that it has already been persisted to the
workflow list.
WorkItem `input_ui_schema` may describe compound UI-only values, but backend DSL
normalization only writes fields declared by the WorkItem `input_schema` into
`work_item.input`.

Example stream request:

```json
{
  "agent_id": "workflow",
  "jsonrpc": "2.0",
  "id": "workflow_req_001",
  "method": "message/stream",
  "params": {
    "message": {
      "parts": [
        {"kind": "text", "text": "创建一个读取 PDF、解析、切分并写入知识库的工作流"}
      ],
      "metadata": {
        "current_dsl_yaml": "",
        "submitted_values": {},
        "selected_node_refs": [
          {"node_id": "moi:document.parse", "version": "v1"},
          {"node_id": "moi:catalog.sink.write", "version": "v1"}
        ]
      }
    },
    "locale": "zh-CN",
    "model": "qwen-plus"
  }
}
```

Follow-up edits should keep the previous task context by sending the previous
task id as `contextId` and passing the current canvas DSL in
`message.metadata.current_dsl_yaml`. The agent treats that DSL as editable
context and returns a new candidate.

`message.metadata.selected_node_refs` is optional. When present, it must be a
non-empty array of exact WorkItem refs selected by the user in the workflow
editor. Each ref must include both `node_id` and `version`. The agent treats
these refs as strong user preferences and preloads backend-verified compact
contracts for those WorkItems, including input/output facts and nearby
upstream/next/sink candidates. This lets the planner build the DSL without
repeating discovery calls when the selected contracts are enough. If a selected
WorkItem cannot be composed into a valid workflow, the agent must ask the user
through the `request_user_input` tool before omitting it; it must not silently
drop the selected node or force an invalid DSL.

### Workflow Artifacts

The workflow frontend should consume artifacts by
`metadata.matrixflow_type` or part `metadata.matrixflow_type`:

| Type | Part kind | Meaning |
|------|-----------|---------|
| `workflow.assistant_delta` | `text` | User-facing assistant text. Runtime text deltas are streamed as append chunks; for an accepted workflow candidate, the server also emits a rendered candidate summary. |
| `workflow.candidate` | `data` | Final compiled DSL candidate. The payload is `SubmitCandidateResult` and includes `dsl_yaml`, `input_form`, diagnostics, and change notes. |
| `matrixflow.workflow.plan` | `data` | Planner steps from the runtime update-plan tool. |
| `workflow.tool_result` | `data` | Structured result for a completed workflow tool call. Artifact metadata includes `tool` and `call_id` so clients can merge it into the matching trace item and render agent-specific tool UI. |
| `matrixflow.workflow.llm_call` | `data` | LLM call telemetry for debugging. |
| `matrixflow.workflow.run_finished` | `data` | Runtime finish reason, turn count, and usage. |

Status update message parts also include `metadata.display` with a stable key,
for example `workflow.status.started`, `workflow.status.tool_started`,
`workflow.status.user_input_required`, and `workflow.status.completed`.
Clients may show `display.default_text` or `display.fallback_text` when they do
not have a localized string for the key.

Generic runtime tool events use the same display metadata contract. Tool call
and tool result DataParts include `metadata.display.key` values such as
`agent.a2a.tool_started` and `agent.a2a.tool_finished`, plus stable params like
`tool` and `call_id`. Clients should render those UI labels with the current UI
locale; they must not infer the assistant answer language from those display
keys or from `params.locale`.

The A2A runtime projection is shared by Explore and Workflow agents. Common
task lifecycle, status updates, assistant deltas, LLM call traces, tool call
traces, plan updates, cancellation, and user-input request artifacts are emitted
by `agent-runtime-v2/a2a`. Agent implementations should differ only by injected
tools and business artifacts such as `explore.answer`, `explore.session_title`,
and `workflow.candidate`.

Workflow-specific tool renderers should handle at least these tool names:

- `browse_capability_groups`: grouped visible WorkItems for capability discovery.
- `inspect_workitem`: one WorkItem contract with schema, UI schema, runtime config contract, explicit binding facts, and complete source code when the WorkItem is a code-based custom operator.
- `search_workitems`: WorkItem matches and `knowledge` hits for DSL/topology constructs such as `chain`, `parallel`, `xor`, `or`, `loop`, `subnet`, `jq`, and reusable workflow patterns.
- `inspect_workflow_dag`: user-drawn canvas DAG facts and WorkItem contracts. When `message.metadata.workflow_dag` is present, the agent must inspect it before submission and treat it as user-provided structure evidence. If the final DSL intentionally differs from that DAG, `submit_workflow_candidate` requires an explicit `workflow_dag_deviation_reason`; otherwise the backend rejects the candidate to prevent silent structure changes.
- `submit_workflow_candidate`: validation status, candidate DSL, input form, and diagnostics.

Unknown tools should keep using the shared generic trace renderer.

Workflow-agent tools return facts and validation feedback. `selected_node_refs`
are user-provided exact anchors; WorkItem contracts, UI/runtime config facts, recall evidence,
and knowledge hits are factual evidence for the model, not hidden backend
obligations. The only acceptance gate for a final DSL candidate is
`submit_workflow_candidate` calling the backend compiler and receiving `ok=true`;
compiler diagnostics are the authoritative repair signal.

`search_workitems` requires the model to provide normalized `keywords` as a
JSON string array, for example `["source","parse","sink"]`. A
comma-separated string such as `"source, parse, sink"` is invalid and should be
returned to the model as a tool argument error instead of being silently
converted.
`phrases` and `user_goal` are optional context: `keywords` drive the database
full-text query, while the longer semantic query used for embedding recall may
include `keywords`, `phrases`, `user_goal`, and `output_target`. The backend
runs full-text and vector recall in parallel and returns an explicit route
error if either route fails or exceeds its route timeout.

`search_workitems` uses the system table `workflow_agent_search_index`. The
index is scoped by `workspace_id`, `user_id`, `locale`, and `agent_code`; it is
rebuilt transactionally from the current visible WorkItem catalog plus static
workflow DSL/topology metadata when the content hash or embedding model changes.
The embedding model is not hardcoded to `BAAI/bge-m3`: the workflow agent
resolves the real configured embedding model id by looking for a model name
containing `bge-m3` in the workspace's effective embedding configuration, which
includes system defaults and workspace embedding backends. If no such model is
available, the tool fails explicitly instead of falling back to keyword-only
search.

### User Input Submit

When the runtime asks the user a question, the model calls the
`request_user_input` tool and the task can pause in `input-required`. The UI
must render that ordinary tool call and submit the answer as an ordinary user
message on the same A2A `contextId`. The answer is plain message text; there is
no separate input-resolution DataPart or resume API.

```json
{
  "agent_id": "workflow",
  "jsonrpc": "2.0",
  "id": "workflow_input_001",
  "method": "message/stream",
  "params": {
    "model": "qwen3.5-plus",
    "message": {
      "role": "user",
      "messageId": "msg_user_002",
      "contextId": "ctx_123",
      "parts": [
        {
          "kind": "text",
          "text": "Where should the result be saved?\n- knowledge_base"
        }
      ]
    }
  }
}
```

Catalog injects workspace and user identity from transport headers before
calling the agent. The next user message must use the same `contextId` as the
pending request so the runtime can continue the same conversation.

The streaming response is the next A2A task on the same context:

```json
{
  "jsonrpc": "2.0",
  "id": "workflow_input_001",
  "result": {
    "kind": "task",
    "id": "task_123",
    "contextId": "ctx_123",
    "status": {
      "state": "working"
    }
  }
}
```

SDK clients submit this message through the normal A2A send or stream method;
UI clients should use `message/stream` so the continuation renders in order.

## Backend Gateway

`moi-backend` exposes the same generic agent gateway under:

```text
GET  /newmoi/agents/card
POST /newmoi/agents/a2a
```

The backend uses `moi-core/go-sdk` `Client.Agents()` and does not translate
Explore events or implement agent tools. It forwards auth-derived context such
as workspace and language through SDK headers.

`GET /newmoi/agents/card` overlays `AgentCard.url` at the public HTTP boundary.
The public discovery URL is `{server.publicOrigin}/newmoi/agents/a2a`. Backend
does not expose Catalog's `/api/v1/agents/a2a` or internal hosts such as
`moi-catalog:8081`, and it does not take the origin from request `Host` or
`X-Forwarded-*` headers. `server.publicOrigin` is required; if it is missing or
the Core Card cannot be parsed as a JSON object, the handler returns `500`
instead of the original Card.
