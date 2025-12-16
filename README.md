# lolo

[![Discord](https://img.shields.io/badge/Join-Discord-blue?logo=discord&logoSize=auto)](https://discord.gg/GNKBE7X75d)

开放空间/OverField 服务端部分实现

支持的客户端版本:cn-最新(ios/windows/android/是cn客户端都支持)

## 如何获取
1. 自编译
2. 前往[releases](https://github.com/BanterSR/Lolo/releases)或[actions](https://github.com/BanterSR/Lolo/actions)下载最新版
  

## 使用方法
1. 下载运行必要文件 [data](https://github.com/BanterSR/Lolo/tree/main/data)和[LoloResource](https://github.com/BanterSR/LoloResource) 分别保存到`data`文件夹和`Resource`文件夹中
2. 用各种方法代理客户端http:`http://dsp-prod-of.inutan.com:18881/dispatch/region_info` 到服务器http （默认:`http://127.0.0.1:18881`）
3. 使用`main/main.exe -g`生成`config.json`修改`config.json`文件
4. 启动Lolo和客户端

## 已实现功能

- [x] 登录
  - [ ] sdk登录
  - [x] 服务发现
- [x] 账号
  - [x] 修改昵称
- [x] 角色
  - [ ] 服装更换
  - [x] 更新装备
  - [x] 角色升级
  - [x] 角色进阶
- [x] 场景
  - [x] 多人游戏
  - [x] 切换房间
  - [x] 切换场景
  - [x] 时间系统
  - [x] 天气系统
  - [ ] 互动
- [x] 背包
- [x] 好友
  - [x] 添加好友
  - [x] 删除好友
  - [x] 拉黑好友
- [x] 聊天
  - [x] 频道
  - [x] 世界
  - [ ] 系统
  - [x] 私聊
- [x] 卡池
  - [x] 新手卡池
  - [x] 普通卡池
  - [x] 限定卡池
  - [ ] 服装卡池

## Fiddler脚本

```javascript
import System;
import System.Windows.Forms;
import Fiddler;
import System.Text.RegularExpressions;

class Handlers {
    static function OnBeforeRequest(oS: Session) {
        var targetDomains = [
            "http://dsp-prod-of.inutan.com:18881/dispatch/region_info"
            ];
        var host = oS.host.ToLower();
        var path = oS.PathAndQuery.ToLower();
        var url = oS.fullUrl.ToLower();

        var proxyHost = "127.0.0.1:18881";
        var isTls = false;
        
        var domainMatch = false;
        for (var i = 0; i < targetDomains.length; i++) {
            if (
                host.EndsWith("." + targetDomains[i]) || 
                host == targetDomains[i] || 
                url == targetDomains[i]
            ) {
                domainMatch = true;
                break;
            }
        }
        
        if (domainMatch) {
            oS.host = proxyHost;
            if (isTls) {
                oS.oRequest.headers.UriScheme = "https";
                FiddlerObject.log("Redirecting: " + oS.fullUrl + " → https://" + proxyHost);
            }else{
                oS.oRequest.headers.UriScheme = "http";
                FiddlerObject.log("Redirecting: " + oS.fullUrl + " → http://" + proxyHost);
            }
        }
    }
};
```