# LoloProto

开放空间客户端协议的 Go 生成代码，由 `../../tools/generate_protocol.py` 从 `client/proto` 的客户端原始描述集生成，不手改。

| 目录 | 来源 | Go 包 |
| --- | --- | --- |
| proto | client/proto/net（package pb） | proto |
| excel | client/proto/config（package protocol），JSON 标签为资源导出的 C# 属性名 | excel |
| config | client/proto/other/scene_config.proto | config |
| cmdid.csv、cmd/cmd_id.go | net 消息的 `(pb.msg_id)` | cmd |
