# Iridium-OverField

Iridium is Lolo's TCP game-protocol sniffer, decoder, and visualizer. Humans can inspect live packets at `http://127.0.0.1:1984/`; automation can consume the JSON API and per-session NDJSON artifacts.

Captured values are preserved as observed. Session output stays under the local, gitignored `captures/` directory unless `outputDir` is overridden.

## Requirements

- Npcap or a compatible capture driver supplied by Wireshark.
- Current protocol definitions under `protocol/proto` and `protocol/cmd`.
- Start capture before entering the scene or triggering the target action.

## Run

From the repository root:

```powershell
go run ./tools/Iridium -l -format json
go run ./tools/Iridium -config ./tools/Iridium/config.json -ip 192.168.1.20
```

The `-listen`, `-output-dir`, and `-auto-start` flags override the corresponding runtime behavior. The default listener is loopback-only at `127.0.0.1:1984`.

## API

```text
GET  /api/health
GET  /api/status
GET  /api/devices
GET  /api/packets?afterId=0&limit=200
GET  /api/packets?name=PlayerMainDataRsp&direction=server_to_client
GET  /api/stream
POST /api/capture/start
POST /api/capture/stop
POST /api/upload
```

Start request:

```json
{
  "label": "feature-baseline",
  "ip": "192.168.1.20",
  "dumpJson": true,
  "includeRaw": true
}
```

Wait for `GET /api/status` to report `capturing` before the client action. After stopping, wait for `stopped` and use the absolute `sessionDir` from the response. HTTP 409 means another session is active.

Each session contains `manifest.json`, `packets.ndjson`, and, for live capture when enabled, `capture.pcapng`. NDJSON contains one complete packet record per line with direction, message and request IDs, sequence ID, decoded object, decode error, and optional raw body base64. Parse it line by line, not as a JSON array.

The bundled frontend continues to use the compatible legacy `GET /api/start` and `GET /api/stop` routes. A slow or closed SSE client does not block capture; the NDJSON output remains authoritative.

See [README-zh.md](README-zh.md) for the full configuration and workflow reference.
