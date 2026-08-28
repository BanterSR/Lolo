---
name: lolo-requirement
description: Orchestrate an evidence-backed Lolo server feature from a natural-language requirement through a full pre-launch official-client capture, protocol analysis, implementation, and verification. Use for end-to-end feature requests; use the specialized Lolo skills directly for only one stage.
---

# Lolo Requirement

Turn the request into observable client actions and a server acceptance check, then run only the stages needed to complete it.

## Normalize The Requirement

Extract these facts from ordinary user language; do not require a template:

- target feature and current failure or missing behavior;
- exact official-client action that triggers it;
- relevant initial state and expected state change;
- visible response, notice, inventory, scene, or persistence result;
- variants needed to distinguish constants from input-dependent values.

Inspect the current repository before asking for details that code or existing protocol names can answer. Ask only when a missing user choice would materially change the captured action or implementation.

## Route The Work

1. Record `git status` and preserve unrelated changes.
2. Verify the official game process is not running. A session attached to an already running game is incomplete and cannot establish official behavior.
3. Use `$lolo-iridium-capture` to start a new named local capture session with an empty `packetFilter`, then wait for status `capturing`.
4. Only after capture is active, use `$lolo-game-control` to launch the official client from a stopped state. Authentication is handed to the user.
5. Keep capture running through client startup, dispatch/login, server entry, the target action, its response, and all follow-up notices.
6. Stop capture only after the scenario has settled or the client has exited cleanly.
7. Use `$lolo-protocol-analysis` to establish the startup sequence, request, response, notices, field relationships, and current server gap.
8. Resolve the active `config.Resources.ResourcePath` and verify the relevant LoloResource files under its `Excel/` or `Config/` tree. Use `$lolo-server-implementation` only after both capture and resource evidence identify a concrete change.
9. Re-run the same full-launch client scenario against the local server when configuration permits, and compare observable behavior with the official capture.

Skip capture only when the user supplied a current Iridium session containing the required baseline and variant evidence. Never label guessed or repository-derived behavior as official behavior.

## Evidence Gate

Before implementation, require:

- a session `manifest.json` whose state is `stopped`;
- evidence that capture reached `capturing` before the game process was launched;
- the complete unfiltered stream from client launch through the target scenario;
- the relevant full records from `packets.ndjson`;
- at least one request and its response or an explicit finding that the behavior is notice-only;
- a baseline and one meaningful variant when a field rule cannot be inferred from a single observation;
- a mapping from message ID/name to `protocol/cmd`, `protocol/proto`, `game/game_router.go`, and the current handler or missing route.
- the exact LoloResource file, record IDs, and fields that govern the requested logic, loaded from the active `Resources.ResourcePath` (default `./Resource`);
- confirmation that no file under `protocol/` needs modification. The repository protocol set is complete and read-only.

If the game was already running when capture began, capture is empty, a packet filter excluded messages, the wrong device or port range is selected, login blocks progress, packet loss makes the sequence ambiguous, decoding disagrees with the protocol set, or the relevant LoloResource files are unavailable, stop implementation. Repeat capture or align the client/resource/protocol versions instead of changing protocol files or inventing behavior.

## Boundaries

Capture only traffic from the user's local official Lolo client within the configured game port range. Do not disable anti-cheat, bypass certificate validation, alter Windows security/privacy settings, or capture unrelated applications. Captures retain complete decoded fields and raw bodies locally; do not upload, message, or otherwise transmit them unless the user explicitly requests a named destination.

Never modify files under `protocol/`. Never accept an empty handler, fixed-success response, placeholder constant, ignored request, or fake implementation as completion. A deliberately empty value is valid only when the exact capture state and LoloResource record both establish it.

Finish with the session path, decisive packet IDs, files changed, tests run, and any remaining behavior not established by evidence.
