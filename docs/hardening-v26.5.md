# Lolo 本地加固版

本页对应 `lab/v26.5-hardening`，基于上游 1.1.12 协议与资源更新。

## 协议与模型

- 303/303 请求均有路由，`router_coverage_test.go` 防止退化。
- `QuestModel`：主线、章节奖励、下一任务。
- `LifeModel`：生活等级、熟练度、技能、成就和生产凭据。
- `ExploreModel`：探索 token、到期时间、奖励凭据。
- `EquipmentGrowth`：武器、防具、海报、铭刻和拆解。
- `RestaurantModel` / `PetModel`：餐厅与宠物。
- `DungeonRuntime`：副本轮次、遭遇、怪物死亡、宝箱和结算。

## 持久化

- 玩家 Blob 使用 gzip + JSON。
- 奖励、下线和关服使用同步保存；新号初始状态可确定性重建，不在登录主循环同步写入大 Blob。
- 异步保存按玩家保持写入顺序。
- SQLite 使用 WAL、busy timeout 和 FULL synchronous；账号链并发创建在事务外读，并在单语句写前完成 ID 分配。

## 验证

```powershell
C:\OverFieldLab\runtime\go1.27\go\bin\go.exe test ./...
C:\OverFieldLab\runtime\go1.27\go\bin\go.exe vet ./...
```

协议机器人支持 `login`、`steady`、`scene`、`quest`、`life`、`loot`。真实客户端使用本地 UID `1000000` 进入场景 `9999`，移动坐标从 `(2417,32,996)` 变化到 `(3046,-573,2855)`。

## 当前专项

- 采集、怪物掉落、Pickup、宝箱和探索奖励使用服务端权威状态；伪造索引、重复拾取和重复领奖会被拒绝。
- 网关拒绝超过 `512000` 字节的包体、非法压缩和未知压缩标志，避免客户端声明超长包导致内存风险。
- `loot` 协议场景已在隔离服务完成联跑，证据为 `C:\OverFieldLab\evidence\lolo-loot-smoke\result.json`。
- 旧存档迁移覆盖损坏任务清理、空生活技能 Blob 默认值补齐；保存失败回滚覆盖奖励、领取标记和物品事务。
- 大房间启用空间网格兴趣管理、每玩家最新移动记录合并、64 人兴趣上限和 10Hz 移动同步；动作与玩家进出变化也按兴趣范围发送。
- 1000 机器人场景压测独立复跑两次均为 SDK/网关/完整登录 `1000/1000`、零请求超时、零断线，RTT p99 `262.14ms`；两次数据库 `integrity_check=ok`。证据为 `C:\OverFieldLab\evidence\lolo-stress-fix\result.json`。
- dev/pprof 资源基线记录到服务端 `1082` goroutine、结束堆 `605.4MB`、采样堆峰值约 `1114MB`，GC 后回落；证据为 `C:\OverFieldLab\evidence\lolo-perf-baseline\result.json`。
- 303/303 路由随机输入模糊测试实际执行 `118832` 次，无 panic、崩溃或物品增加；零值请求回归仍保留。
- 真实子进程跳过 SQLite 关闭的 WAL 测试验证已提交保留、未提交回滚；100 机器人在线时强制终止服务后 `integrity_check=ok`，重启后既有账号连续登录成功。证据为 `C:\OverFieldLab\evidence\lolo-crash-recovery\result.json`。

## 限制

- 跨服联机未开放。
- 部分活动缺权威奖励配置，当前拒绝发奖。
- 上游 PR #13 仍 Open，维护者未 Review。
- 默认场景聊天仍为房间级广播，已计入压测流量；千人场景服务端堆峰值约 1.1GB，部署时应按容量规划。
