# Iridium-OverField

Iridium 是 Lolo 仓库内的 TCP 游戏协议抓包、解码和可视化工具。它同时服务两种使用方式：

- 人类通过 `http://127.0.0.1:1984/` 查看实时包、筛选内容和上传 pcap；
- 自动化或 AI 通过稳定的 JSON API、`manifest.json` 和 `packets.ndjson` 获取完整证据。

结构化输出保留原始字段和值，并可保留原始包体 base64。所有会话默认只写入本机 `captures/`，该目录已被 Git 忽略。

## 前置条件

1. 安装 Npcap 驱动或 Wireshark 提供的兼容抓包驱动。
2. 仓库需要包含 `protocol/proto` 和 `protocol/cmd` 中的当前协议定义。
3. 在进入游戏场景或触发目标操作前开始抓包。

## 网卡选择

从仓库根目录列出网卡：

```powershell
go run ./tools/Iridium -l
go run ./tools/Iridium -l -format json
```

将返回的设备名写入 `tools/Iridium/config.json` 的 `deviceName`，或者启动时通过本机 IP 自动选择：

```powershell
go run ./tools/Iridium -config ./tools/Iridium/config.json -ip 192.168.1.20
```

不要在多个候选网卡之间猜测。抓不到包时，先检查网卡/IP 和游戏端口范围。

## 启动

```powershell
go run ./tools/Iridium -config ./tools/Iridium/config.json
```

常用覆盖参数：

```text
-listen 127.0.0.1:1985   覆盖 HTTP 监听地址
-output-dir <path>        覆盖抓包会话目录
-auto-start               HTTP 服务启动后立即开始实时抓包
```

默认监听 `127.0.0.1:1984`，避免把包含完整游戏数据的 API 暴露到局域网。

## 人类界面

打开 `http://127.0.0.1:1984/`：

- Start/Stop 控制实时抓包；
- 表格持续显示解析后的请求、响应和通知；
- 上传 pcap 会创建一个离线解析会话；
- 浏览器过慢或关闭不会阻塞底层抓包，完整结果仍会写入会话目录。

旧界面使用的 `GET /api/start` 和 `GET /api/stop` 保持兼容。

## 自动化 API

健康和状态：

```text
GET  /api/health
GET  /api/status
GET  /api/devices
```

开始和停止：

```text
POST /api/capture/start
POST /api/capture/stop
```

开始请求示例：

```json
{
  "label": "battle-baseline",
  "ip": "192.168.1.20",
  "dumpJson": true,
  "includeRaw": true
}
```

`deviceName` 可替代 `ip`。HTTP 202 表示已接受，HTTP 409 表示已有会话处于 `starting`、`capturing` 或 `stopping`。

查询内存中的最近数据：

```text
GET /api/packets?afterId=0&limit=200
GET /api/packets?name=PlayerMainDataRsp&direction=server_to_client
GET /api/stream
```

`direction` 只接受 `client_to_server` 或 `server_to_client`。`/api/stream` 是供网页使用的 SSE；自动化应优先使用 `/api/packets` 和最终 NDJSON。

上传离线 pcap：

```text
POST /api/upload
Content-Type: multipart/form-data
字段名: file
```

## 会话产物

每次实时或离线解析都创建独立目录：

```text
captures/<时间>-<label>/
|-- manifest.json
|-- packets.ndjson
`-- capture.pcapng
```

`manifest.json` 记录状态、模式、设备、端口、时间、包数量、错误和绝对文件路径。停止后状态应为 `stopped`；`error` 时读取 `lastError`。

`packets.ndjson` 每行是一个独立 JSON 对象，主要字段为：

```json
{
  "id": 1,
  "time": 1787900000000,
  "timestamp": "2026-08-28T16:00:00+08:00",
  "direction": "client_to_server",
  "messageId": 1005,
  "messageName": "PlayerMainDataReq",
  "requestId": 12,
  "sequenceId": 34,
  "bodySize": 18,
  "object": {},
  "rawBase64": "...",
  "decodeError": ""
}
```

NDJSON 不是一个 JSON 数组，应逐行解析。`packetBufferSize` 只限制 `/api/packets` 的内存窗口，不限制落盘的 NDJSON。

## 配置

```json
{
  "deviceName": "",
  "packetFilter": [],
  "autoSavePcapFiles": true,
  "dumpJson": true,
  "includeRawInDump": true,
  "maxPort": 11100,
  "minPort": 11000,
  "listenAddr": "127.0.0.1:1984",
  "outputDir": "./captures",
  "packetBufferSize": 2000
}
```

相对 `outputDir` 以配置文件所在目录为基准。`packetFilter` 中的消息名不会出现在网页、API 或 NDJSON 中，但原始 pcap 仍保留对应网络包。
