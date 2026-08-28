---
name: lolo-iridium-capture
description: Start, control, and inspect the repository's Iridium capture service before launching an authorized local Lolo client. Use for full-flow live capture, device discovery, offline pcap decoding, packet polling, and capture artifacts; do not use for game UI actions or server implementation.
---

# Lolo Iridium Capture

Run Iridium from the repository root with normal shell tools. Do not automate a terminal through Windows Computer Use.

## Preflight

1. Confirm `tools/Iridium/config.json` exists and review ports `11000-11100` unless current server configuration proves a different range.
2. Confirm the official game process is not running. Do not start or accept a canonical capture after the game has launched.
3. Check whether Npcap or a compatible pcap driver is already available with:

   `go run ./tools/Iridium -l -format json`

4. Select a device by exact `deviceName` or by a local IP returned for that device. Do not guess a device when several are plausible.
5. Require `packetFilter` to be an empty array for official-behavior evidence. Filtering is allowed only for later exploratory viewing, never for the canonical capture.
6. Use an already installed driver. Installing software or changing capture/security settings requires the applicable user confirmation.

## Start Iridium

Launch a persistent shell process:

`go run ./tools/Iridium -config ./tools/Iridium/config.json`

Use `-ip <local-ip>` to override the configured device, `-listen 127.0.0.1:<port>` when port 1984 is occupied, and `-output-dir <path>` for a temporary evaluation directory. Keep the process session alive until capture is stopped and status is read.

Verify `GET /api/health` returns `ok: true`. Use `GET /api/devices` when API-based device discovery is more convenient.

## Capture The Full Client Flow

Start with `POST /api/capture/start` and a JSON body such as:

```json
{
  "label": "feature-baseline",
  "ip": "192.168.1.20",
  "dumpJson": true,
  "includeRaw": true
}
```

The response must be HTTP 202. HTTP 409 means another session is active; inspect `GET /api/status` instead of starting another capture. Wait until status is `capturing` before launching the game.

Do not launch the game until status is `capturing`. The canonical session starts with no running game process and includes the complete stream from game launch through dispatch/login, server entry, the target action, its response, and follow-up notices. If the game was already running, stop it and discard the partial session; begin a new capture before relaunching.

Use `GET /api/packets?afterId=<id>` only for incremental monitoring. Optional `name` and `direction` query filters affect the API view, not the saved stream. Do not add packet names to `config.packetFilter` during canonical capture.

Stop with `POST /api/capture/stop` only after the scenario settles or the client exits. Poll `GET /api/status` until `stopped` or `error`; do not analyze a still-open dump. Preserve the returned absolute `sessionDir`.

Each session contains:

- `manifest.json`: state, device, ports, time range, paths, packet count, and error;
- `packets.ndjson`: one complete decoded record per line, including `requestId`, `sequenceId`, object fields, and raw body base64;
- `capture.pcapng`: original live traffic when `autoSavePcapFiles` is enabled.

These files intentionally retain original values and are gitignored. Keep them local and do not transmit them without an explicit destination from the user.

For an existing pcap, upload it to `POST /api/upload` as form field `file`; Iridium creates the same manifest and NDJSON outputs in `offline` mode.

## Failure Rules

- Game launched before capture: discard the session and repeat from a stopped game process.
- Zero packets: check device/IP and port range, then start a new named session before launching the game.
- Filtered canonical session: discard it and repeat with an empty `packetFilter`.
- `error` state: report `lastError` exactly and keep the manifest.
- Truncated or decode-error packets: preserve raw base64 and pcap, then diagnose capture loss or client/protocol version mismatch. The repository protocol files are complete; do not modify them to fit a bad or mismatched capture.
- Slow or closed browser: continue through `/api/packets`; SSE subscribers never define capture success.
