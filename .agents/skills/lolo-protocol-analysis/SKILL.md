---
name: lolo-protocol-analysis
description: Analyze a completed Iridium session and map observed official-client traffic to Lolo protocol definitions, routes, handlers, and state. Use for evidence extraction and gap analysis; do not control the game or implement speculative behavior.
---

# Lolo Protocol Analysis

Require a completed session directory containing `manifest.json` and `packets.ndjson`. Read NDJSON as independent JSON records, not as one JSON array.

Treat every file under `protocol/` as complete and read-only. A message that cannot be decoded by the current protocol set indicates capture corruption/loss or a client, resource, and protocol version mismatch; it is not evidence that a protocol definition should be added or changed.

## Establish The Sequence

1. Verify manifest state is `stopped`; report `error` and `lastError` without continuing if decoding failed materially.
2. Restrict records to the game-control time window or starting packet ID.
3. Order by `id`, retaining `time`, `direction`, `messageId`, `messageName`, `requestId`, and `sequenceId`.
4. Pair request and response by message family, direction, time, and echoed `requestId`; attach server notices that immediately follow.
5. Compare baseline and variant sessions field by field. Classify each conclusion as observed invariant, observed variable, or unresolved hypothesis.

Do not discard zero values, empty arrays, or absent fields. They can distinguish protocol presence from business defaults. Raw bodies and decoded values are intentionally complete; do not redact or normalize them during local analysis.

## Map To The Repository

Search in this order without editing any result:

1. `protocol/cmd/cmd_id.go` and `protocol/cmdid.csv` for message IDs;
2. `protocol/proto/<Message>.proto` and generated `.pb.go` for field numbers and types;
3. `game/game_router.go` for request routing;
4. the matching `game/player_*.go` handler and `game/model` state;
5. `config.Resources.ResourcePath`, the matching LoloResource file under `Excel/` or `Config/`, and its `gdconf` loader/accessor;
6. `db`, player models, gateway sends, and related notices only when the observed behavior uses them.

Keep protocol field types aligned with generated Go types. If a message is unknown, use `decodeError`, `rawBase64`, dynamic field numbers, and the pcap only to diagnose the bad capture or version mismatch. Never edit a protocol schema or generated protocol artifact.

Do not continue an unknown-message analysis into implementation. Re-capture from before game launch and verify that the official client version, LoloResource version, and repository protocol version match. Dynamic wire output is diagnostic only and never authorizes changes under `protocol/`.

## Evidence Report

Produce a concise implementation handoff with:

- session path and exact record IDs;
- request, response, and notice message names/IDs;
- complete relevant bodies;
- exact LoloResource path, source file, record IDs, and fields that govern the behavior;
- request-to-response and input-to-output field relationships;
- persistence or follow-up behavior observed across a second action when relevant;
- current route/handler locations and the precise mismatch or missing behavior;
- unresolved questions that require another controlled capture.

Repository code is evidence of the current implementation, not evidence of official behavior. Keep those two sources separate.
