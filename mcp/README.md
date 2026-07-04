# game/mcp — AI mcp / tools 服务器

给 AI 提供游戏咨询能力的服务器,一套工具同时以两种方式对外：

1. **MCP over HTTP**（JSON-RPC 2.0 / Streamable HTTP）—— 供 Claude、Cursor 等 MCP 客户端接入。
2. **OpenAI 兼容 tools** —— 供 `command` 里注册的 AI 直接作为 function tools 调用。

工具按模块分文件（`tools_<模块>.go`），覆盖 wiki 式攻略数据：**通用资源**、**角色**、**副本**、**养成**、**世界信息**、**服务器状态**、**玩家查询**。
每个内容模块都提供两类工具：`xxx_info` 等**按 id 精查详情**，以及 `xxx_list` **最小快照**（只含 `id+名称`，用最省 token 的方式枚举全表，可 `limit`/`offset` 分页）。
玩家查询默认只读数据库离线数据；`player_live` 额外提供在线玩家实时快照（经游戏主线程读取,避免并发竞争）。

---

## 1. 配置

`config.json` 中的 `Mcp` 段（缺省已启用）：

```json
{
  "Mcp": {
    "Enable": true,
    "Path": "/mcp",
    "Token": ""
  }
}
```

| 字段 | 说明 |
| --- | --- |
| `Enable` | 是否启用；`false` 时 `mcp.New` 返回 `nil`，不挂载任何路由 |
| `Path` | 挂载路径前缀，缺省 `/mcp` |
| `Token` | 访问令牌；**留空会在启动时自动生成一次性 uuid 并打印到日志**（重启会变，建议在此固定）|

> 服务挂在主 HTTP 服务器上（`config.HttpNet`，缺省 `0.0.0.0:8080`）。
> 因此端点为 `http://<host>:8080<Path>`，例如 `http://127.0.0.1:8080/mcp`。

---

## 2. 鉴权

除非 `Token` 为空且你没改配置，否则每个请求都要带令牌，三种方式任选其一：

```
Authorization: Bearer <token>
X-Api-Key: <token>
?token=<token>          # 作为 query 参数
```

令牌不匹配返回 `401`。（未配置 `Token` 时看服务端启动日志里自动生成的那串。）

---

## 3. 协议与端点

- `POST <Path>`：处理一次 JSON-RPC 请求，返回 `application/json`。
- `GET  <Path>`：返回 `405`（本服务不提供服务端主动推流）。

支持的方法：`initialize`、`notifications/initialized`、`ping`、`tools/list`、`tools/call`。

### curl 示例

初始化：

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}'
```

列出工具：

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H "Authorization: Bearer <token>" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
```

调用工具（查询角色 101001 的中文信息）：

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H "Authorization: Bearer <token>" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call",
       "params":{"name":"character_info","arguments":{"id":101001,"lang":"zh-Hans"}}}'
```

`tools/call` 的返回形如：

```json
{"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"{\"id\":101001,\"name\":\"...\"}"}]}}
```

工具结果是 JSON 文本，放在 `content[0].text` 里。

---

## 4. 在 MCP 客户端里接入

**原生 Streamable HTTP**（支持 URL + header 的客户端，如 Cursor / 部分版本的 Claude）：

```json
{
  "mcpServers": {
    "lolo": {
      "type": "streamable-http",
      "url": "http://127.0.0.1:8080/mcp",
      "headers": { "Authorization": "Bearer <token>" }
    }
  }
}
```

**只支持 stdio 的客户端**（如桌面版 Claude），用 `mcp-remote` 桥接：

```json
{
  "mcpServers": {
    "lolo": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "http://127.0.0.1:8080/mcp",
               "--header", "Authorization: Bearer <token>"]
    }
  }
}
```

---

## 5. 工具清单

`lang` 全部可选 `zh-Hans`(简体) / `zh-Hant`(繁体) / `en` / `ja` / `ko`，缺省 `zh-Hans`。
所有 `*_list` 入参统一为 `lang?` / `limit?` / `offset?`，返回 `{total,count,list:[{id,name}]}`（不传 `limit` 返回全部）。

### 通用资源（`tools_game.go`）

| 工具 | 入参 | 说明 |
| --- | --- | --- |
| `game_item` | `id`, `lang?` | 物品名称/描述/品质/背包分类 |
| `item_list` | list | 枚举全部物品/货币（很多，建议分页）|
| `game_gacha_pools` | `lang?` | 当前开放卡池（含 up 角色与时间）|
| `game_search` | `keyword`, `type?`, `lang?`, `limit?` | 按名称搜角色/物品（要枚举全部改用 `*_list`）|

### 角色（`tools_character.go`）

| 工具 | 入参 | 说明 |
| --- | --- | --- |
| `character_info` | `id`, `lang?` | 角色名/元素/阵营/性别/武器与护甲类型/技能id/卡池时间/海报 |
| `character_skills` | `id`, `level?`, `lang?` | 各技能名称/描述/类型/冷却/耗蓝（按等级）|
| `character_growth` | `id`, `lang?` | 各阶属性上限与升阶/突破材料、官方推荐练度（武器/目标等级/练度副本）|
| `character_voice` | `id`, `lang?` | 语音台词条目（分类/标签/解锁等级/音频路径）|
| `character_list` | list | 枚举全部角色 |

### 副本（`tools_dungeon.go`）

| 工具 | 入参 | 说明 |
| --- | --- | --- |
| `dungeon_info` | `id`, `lang?` | 推荐等级/体力/限时/三星时间/怪物列表/掉落 |
| `monster_info` | `id`, `level?`, `lang?` | 阵营/元素/技能id/指定等级属性/掉落 |
| `dungeon_list` / `monster_list` | list | 枚举全部副本 / 怪物 |

### 养成（`tools_cultivate.go`）

| 工具 | 入参 | 说明 |
| --- | --- | --- |
| `make_recipe` | `id`, `lang?` | 制作配方：产物/材料/等级/耗时/成功率 |
| `weapon_info` | `id`, `lang?` | 武器名称/类型/评分/耐久/被动/各级攻击·暴击成长 |
| `armor_info` | `id`, `lang?` | 护甲名称/部位/评分/套装/被动 |
| `inscription_info` | `id`, `lang?` | 铭文各级被动效果/金币/升级材料 |
| `make_list` / `weapon_list` / `armor_list` / `inscription_list` | list | 枚举各表 |

### 世界信息（`tools_world.go`）

| 工具 | 入参 | 说明 |
| --- | --- | --- |
| `scene_info` | `id`, `lang?` | 场景名/区域/出生点·采集·宝箱·副本·怪物数量 |
| `quest_info` | `id`, `lang?` | 任务名/任务组/类型/奖励id/条件组 |
| `story_info` | `id`, `lang?` | 剧情章节名/前置/解锁等级/剧情id |
| `achievement_info` | `id`, `lang?` | 成就名/描述/条件类型/计数 |
| `shop_info` | `id`, `lang?` | 商店名/类型/开放时间/在售物品 |
| `scene_list` / `quest_list` / `story_list` / `achievement_list` / `shop_list` | list | 枚举各表 |

### 服务器咨询（`tools_server.go`）

| 工具 | 入参 | 说明 |
| --- | --- | --- |
| `server_status` | 无 | 版本 / 模式 / 在线人数 / 注册人数 |
| `server_content` | 无 | 角色 / 物品 / 武器 / 开放卡池数量 |

### 玩家查询（`tools_player.go`）

| 工具 | 入参 | 说明 |
| --- | --- | --- |
| `player_basic` | `userId` | 昵称/等级/经验/头像/签名/在线状态（离线库）|
| `player_live` | `userId` | **在线玩家实时快照**：当前场景id/坐标/房间号/世界等级/当前队伍/角色与物品数量/最后活跃 |
| `player_account` | `userId` | 渠道/设备/封禁状态 |
| `player_characters` | `userId`, `lang?` | 已拥有角色（id/名称/等级/星级）|
| `player_items` | `userId`, `bagTag?`, `limit?`, `lang?` | 背包基础物品，可按分类过滤 |
| `player_team` | `userId`, `lang?` | 当前队伍三名角色 |
| `player_search` | `keyword`, `limit?` | 按昵称模糊搜索玩家 |
| `player_gacha_records` | `userId`, `gachaId`, `page?`, `lang?` | 某卡池抽卡记录（每页 5 条）|
| `player_friends` | `userId`, `type?` | `friend`/`apply_received`/`apply_sent`/`black` |

> 除 `player_live` 外的玩家查询均读数据库离线快照（可能落后约 5 分钟）；返回里的 `online` 表示该玩家当前是否在内存中。
> `player_live` 仅在玩家在线时可用，经游戏主线程读取实时字段，避免并发竞争。

---

## 6. 在 command 的 AI 中使用（OpenAI 兼容，无需改 command 包）

同一套工具通过 `mcp.Default()` 拿到，`OpenAITools()` 给出工具定义，`CallTool()` 执行：

```go
// 挂上工具
if m := mcp.Default(); m != nil {
    params.Tools = m.OpenAITools()
}

completion, _ := a.openAi.Chat.Completions.New(ctx, params)

// 执行模型请求的工具调用，把结果回填后再请求一次
for _, tc := range completion.Choices[0].Message.ToolCalls {
    out := mcp.Default().CallTool(tc.Function.Name, tc.Function.Arguments)
    messages = append(messages, openai.ToolMessage(out, tc.ID))
}
// 带着追加了 tool 结果的 messages 再调一次 Completions 即可拿到最终回答
```

`mcp.New` 未启用（`Enable=false`）时 `mcp.Default()` 返回 `nil`，记得判空。

---

## 7. 扩展工具

在对应的 `tools_*.go` 里 `registerTool` 一个 `Tool` 即可，两种前端自动生效：

```go
s.registerTool(&Tool{
    Name:        "game_monster",
    Description: "查询怪物配置",
    InputSchema: schema(H{
        "id":   prop("integer", "怪物id"),
        "lang": langProp(),
    }, "id"), // 后面的可变参数是 required 字段
    Handler: func(args Args) (any, error) {
        c := gdconf.GetMonsterCharacterConfigure(args.Uint32("id"))
        if c == nil {
            return nil, fmt.Errorf("怪物不存在")
        }
        return H{"id": c.GetID() /* ... 只挑高价值字段 ... */}, nil
    },
})
```

**约定**：工具要“按需精确命中”——按 id 直查、只挑高价值字段、搜索/列表一律带 `limit`，
不要把整张资源表或整个玩家 JSON 塞进返回值（浪费 AI 上下文和 token）。

---

## 8. 注意事项

- **玩家离线数据只读离线库**：`player_*`（除 `player_live`）经 `db.GetOFGameByUserId` 反序列化成 `model.Player` 快照，绝不直接读在线 `game.userMap`。
- **实时快照 `player_live`**：不直接触碰在线玩家内存，而是给游戏主线程投递 `SnapshotPlayer` 任务，在**唯一写线程**内只读取少量高价值字段拼成紧凑结果——既避免并发竞争，也避免高频访问带来的全量序列化开销。
- **i18n 文本**：名称/描述来自 `String_<Lang>.json`（`gdconf` 启动时按语言加载，见 `gdconf/excel.String.go`）。
  已索引的文本表：物品/角色名/技能/副本/任务/场景/成就/商店/剧情/制作；语音来自 `Voice_<Lang>.json`（`gdconf/excel.Voice.go`）。缺失文件时工具仍可用，只是名称为空。
- **角色语音的边界**：`character_voice` 只有台词**条目**（分类/标签/解锁等级/音频路径）。服务器数据里**没有台词全文（`TextValue` 为空）也没有声优/CV**，角色性格/人物档案等 prose 属客户端资源，服务端不具备，故不提供。
- **最小快照 `*_list`**：解决"关键词不能为空导致无法枚举全表"的问题——每个内容模块都有 `*_list`，只回 `id+名称`，最省 token；超大表（物品/任务/成就/怪物）用 `limit`/`offset` 分页。
- **邮件**：玩家个人邮箱当前未持久化（`GetMails` 返回空、仅有静态系统邮件），故未提供邮件查询工具；
  实现邮件入库后再补 `player_mails` 即可。
