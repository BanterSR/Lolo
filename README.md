# lolo
箱庭游戏服务端部分实现

## 使用方法
1. 编译Lolo `go build main.go` 得到main或main.exe
2. 下载运行必要文件 [data](https://github.com/BanterSR/Lolo/tree/main/data)和[LoloResource](https://github.com/BanterSR/LoloResource) 分别保存到`data`文件夹和`Resource`文件夹中
3. 用各种方法代理客户端http:`http://dsp-prod-of.inutan.com:18881/dispatch/region_info` 到服务器http （默认:`http://127.0.0.1:18881`）
4. 使用`main/main.exe -g`生成`config.json`修改`config.json`文件
5. 启动Lolo和客户端

## 已实现功能

- [x] 登录
  - [ ] sdk登录
- [x] 角色
  - [x] 换装
- [x] 场景
  - [x] 多人游戏
  - [x] 切换房间
  - [x] 切换场景
- [x] 背包
- [x] 养成