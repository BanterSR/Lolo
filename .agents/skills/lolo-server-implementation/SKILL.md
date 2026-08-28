---
name: lolo-server-implementation
description: Implement and verify Lolo Go server behavior from a completed protocol evidence report and Iridium records. Use when official request, response, notice, and state rules are established; return to protocol analysis when evidence is ambiguous.
---

# Lolo Server Implementation

Start from the evidence report and open the cited NDJSON records yourself. Also open the exact LoloResource files named by the report through the active `config.Resources.ResourcePath` (default `./Resource`). Do not derive unobserved constants, defaults, rewards, persistence rules, or field relationships from message names or existing placeholder code.

Every file under `protocol/` is complete and read-only. Never edit `.proto`, generated `.pb.go`, `protocol/cmdid.csv`, command maps, or any other protocol artifact. A decode mismatch must return to capture/version diagnosis.

## Trace Before Editing

Confirm all applicable layers:

- numeric ID and message registration in `protocol/cmd`;
- protobuf request/response/notice field types;
- route entry in `game/game_router.go`;
- neighboring handler conventions in the relevant `game/player_*.go` file;
- player/model state ownership and persistence path;
- the exact resource records under `Resource/Excel` or `Resource/Config` and the existing `gdconf` loader/accessor for them;
- outbound notices or related client refresh requests.

Preserve unrelated dirty worktree changes. Prefer existing repository helpers and models over a new abstraction.

## Implement The Behavior

Follow the local handler shape: decode the typed request from `msg.Body`, initialize an explicit response status, send with the incoming `msg.PacketId`, validate state before mutation, and emit only notices proven by capture. Keep API/protocol IDs and generated Go field types aligned. Reuse the existing `gdconf`, model, database, `alg`, and send helpers used by neighboring handlers instead of introducing a parallel architecture.

Persist independent observed values independently. A value shown in the official response is not automatically a stored field; establish persistence with a second read, reconnect, or existing authoritative state path.

Do not implement an empty or placeholder handler merely to keep the client connected. Prohibited substitutes include ignoring request fields, returning unconditional success, returning nil/empty collections without capture and resource support, hardcoding guessed IDs/rewards/state, adding TODO-only branches, or emitting fake notices. If the full capture and LoloResource do not establish the real rule, stop and request the missing evidence.

Match the current project conventions before general preferences: read adjacent handlers and models, keep changes in their existing ownership layer, follow local naming and response construction, use `gofmt`, and avoid unrelated refactors.

## Verification

Add focused tests for deterministic business rules and state mutation when practical. At minimum:

1. run `gofmt` on changed Go files;
2. run focused tests for changed packages;
3. run `go test ./...` when repository baseline and runtime dependencies permit;
4. run `go vet` or an existing lint command for affected packages;
5. check `git diff --check` and verify `git diff -- protocol` is empty;
6. replay the same client scenario against the local server when configuration permits and compare response, notices, and visible state with the official capture.

Report passing checks separately from pre-existing or environment-specific blockers. Completion requires observable equivalence for the requested scenario, not only compilation.
