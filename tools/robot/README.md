# robot — Lolo 协议压测机器人

纯协议实现的机器人客户端，走完整 SDK HTTP 登录 + 网关 TCP 握手登入当前服务器，
在多种「测试模式 × 场景」组合下对服务端施压，实时输出性能表现，并可选提供 web 仪表盘。

不改动服务端任何代码，帧格式镜像 `pkg/ofnet/net-tcp.go`（2 字节头长 + PacketHead + body，
snappy 压缩、无加密），协议消息复用 `protocol/proto` 与 `protocol/cmd`。

## 两个正交维度

- **模式 `-mode`**：控制并发数（CCU）随时间的形状
  - `routine`  常规：`ramp` 内爬坡到 `ccu`，保持 `duration`，结束（一次性快照）
  - `endurance` 持久：长时间保持 `ccu`，掉线自动重连，输出延迟漂移（首窗 vs 末窗 p95）
  - `profile`  规则：按 `config` 阶段列表，或内置曲线 `-pattern step|wave|spike` 跑，`-loop` 可循环
- **场景 `-scenario`**：控制每个 bot 的行为
  - `login`   登录风暴：完成登录即断开重连，压登录/登出链路
  - `steady`  定常运转：周期 ping 保活 + 间歇拉取主数据
  - `scene`   场景同步：加入房间后循环上报角色移动 + 偶发动作（战斗为纯客户端，故压场景行为）

任意组合，如 `scene + endurance`、`steady + profile`。

## 构建

```
go build -o robot.exe ./tools/robot/
```

## 用法示例

```bash
# 常规一次性快照：1000 并发，30s 爬坡，保持 5 分钟
robot -mode routine  -scenario scene  -ccu 1000 -ramp 30s -duration 5m

# 持久 soak：500 并发保持 6 小时，掉线自动重连
robot -mode endurance -scenario steady -ccu 500 -duration 6h

# 规则性阶梯（config 阶段驱动）+ web 仪表盘
robot -mode profile  -scenario scene  -config tools/robot/config.example.json

# 规则性内置曲线：波形 / 阶梯 / 尖峰
robot -mode profile -pattern wave  -base 100 -peak 1000 -period 2m -duration 10m
robot -mode profile -pattern step  -base 0   -peak 2000 -steps 5 -step-hold 60s
robot -mode profile -pattern spike -base 100 -peak 3000 -spike 15s -step-hold 30s
```

## 常用参数

| flag | 说明 | 默认 |
|------|------|------|
| `-gate` | 网关 TCP 地址 | `127.0.0.1:11000` |
| `-sdk` | SDK HTTP 基地址 | `http://127.0.0.1:8080` |
| `-mode` | `routine\|endurance\|profile` | `routine` |
| `-scenario` | `login\|steady\|scene` | `steady` |
| `-ccu` | 目标并发 bot 数 | `100` |
| `-ramp` / `-duration` | 爬坡 / 保持时长 | `10s` / `60s` |
| `-pattern` | profile 内置曲线 `step\|wave\|spike` | 空 |
| `-peak` / `-base` | 曲线峰值 / 基线并发 | `0` / `0` |
| `-period` / `-steps` / `-step-hold` / `-spike` | 曲线参数 | — |
| `-loop` | profile 循环 | `false` |
| `-ping` / `-action` | ping / 场景动作间隔 | `15s` / `2s` |
| `-report` | 上报间隔 | `1s` |
| `-web` | web 仪表盘地址，如 `:8090`，空则关闭 | 空 |
| `-pprof` | 服务端 pprof 基地址，空则沿用 `-sdk` | 空 |
| `-prefix` / `-password` | 账号前缀 / 密码 | `robot_` / `robot123456` |
| `-config` | JSON 配置文件（见 `config.example.json`） | — |

`-config` 提供基线，命令行 flag 覆盖显式设置的项。

## 配置文件（逐项注释）

即用型副本：`tools/robot/config.example.json` 与 `.build/robot/config.json`。

> 注意：JSON 本身不支持注释，下面的 `//` 仅用于讲解，真实文件里请勿保留。

```jsonc
{
  // ---- 目标地址 ----
  "gate": "127.0.0.1:11000",       // 网关 TCP 地址（服务端 GateWay.OuterAddr 的可达地址）
  "sdk":  "http://127.0.0.1:8080", // SDK HTTP 基地址（服务端 HttpNet，登录链路走这里）

  // ---- 测试模式 × 场景（两个正交维度）----
  "mode":     "routine",           // 模式: routine 常规 | endurance 持久 | profile 规则
  "scenario": "scene",             // 场景: login 登录风暴 | steady 定常 | scene 场景同步

  // ---- routine / endurance 参数 ----
  "ccu":      200,                 // 目标并发 bot 数（峰值在线机器人数）
  "ramp":     "30s",               // 爬坡时长：在此时间内线性升到 ccu，避免瞬时冲击
  "duration": "2m",                // 保持时长：达到 ccu 后维持多久（endurance 建议数小时）

  // ---- profile 参数（mode=profile 时生效）----
  "pattern":  "",                  // 内置曲线: step 阶梯 | wave 波形 | spike 尖峰；为空则用下方 stages
  "peak":     0,                   // 曲线峰值并发（step/wave/spike 用；为 0 时取 ccu）
  "base":     0,                   // 曲线基线并发（wave/spike 的谷底、step 的起点）
  "period":   "2m",                // wave 一个完整波峰→波谷的周期
  "steps":    5,                   // step 阶梯数（base→peak 分几级递增）
  "stepHold": "60s",               // step 每级 / spike 预热段的持续时长
  "spike":    "10s",               // spike 峰值段的持续时长
  "loop":     false,               // profile 曲线跑完是否从头循环
  "stages": [                      // 显式阶段列表（非空时优先于 pattern）：逐段维持 ccu 达 hold
    {"ccu": 100,  "hold": "60s"},  //   第 1 段：100 并发保持 60 秒
    {"ccu": 500,  "hold": "60s"},  //   第 2 段：500 并发保持 60 秒
    {"ccu": 1000, "hold": "120s"}  //   第 3 段：1000 并发保持 120 秒
  ],

  // ---- 账号 ----
  "prefix":   "robot_",            // 账号用户名前缀，账号 = prefix + 序号；不存在时服务端自动创建
  "password": "robot123456",       // 账号密码（所有机器人共用）

  // ---- 场景节奏 ----
  "ping":   "15s",                 // ping 心跳间隔（保活，避免服务端 60s 读超时踢连接）
  "action": "2s",                  // scene 场景下角色移动上报间隔（每 5 次移动附带一次动作）
  "report": "1s",                  // 控制台实况与仪表盘的采样/刷新间隔

  // ---- 观测 ----
  "web":   ":8090",                // web 仪表盘监听地址，空字符串则关闭
  "pprof": ""                      // 服务端 pprof 基地址，空则沿用 sdk（dev 模式下 /debug/pprof 可用）
}
```

各字段与命令行 flag 一一对应，同名 flag 会覆盖文件中的值。时长字段接受 Go duration 写法（`300ms`/`15s`/`2m`/`6h`）。

## Web 仪表盘

`-web :8090` 后浏览器打开 `http://localhost:8090/`，实时曲线：
并发 CCU（active/target）、吞吐（rps / 登录·秒⁻¹）、RTT 延迟（p50/p95/p99）、服务端 goroutine，
以及登录成功/失败、断线、超时、服务端堆内存/GC 等指标。数据来自 `/api/stats`（JSON）。

## 服务端资源观测（pprof）

lolo 在 **dev 模式**下注册了 `/debug/pprof/*`（与 SDK 同端口 `:8080`）。
本工具每个上报周期抓取服务端 `goroutine` 数与堆内存（`HeapAlloc/Sys/NumGC`），
在实况行、仪表盘与最终汇总中与客户端负载对齐展示。非 dev 模式则显示「pprof 不可达」。

## 结果汇总

进程结束（曲线跑完或 Ctrl+C）后打印：登录成功率、请求/响应、断线、收发流量、
累计 RTT 分位（p50/p95/p99）、延迟漂移，以及**硬件/资源**信息
（客户端 CPU 核数、GOMAXPROCS、OS/Arch、Go 版本、客户端内存，及服务端 pprof 资源快照）。

## 说明

- 走真实 SDK 登录链：`/v1/user/loginByName` → `/v2/users/checkLogin`（AES-ECB 加密 + 签名）
  取得 GateToken，再 TCP `VerifyLoginTokenReq` → `PlayerLoginReq`。账号不存在时服务端自动创建。
- 账号 = `prefix + id`，id 在回收后复用，故账号集合有界（≈ 峰值并发）。
- 工具会在服务端库中创建 `robot_*` 测试账号，压测后可按需清理。
