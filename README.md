# lolo

[![Discord](https://img.shields.io/badge/Join-Discord-blue?logo=discord&logoSize=auto)](https://discord.gg/GNKBE7X75d)

开放空间/OverField 服务端部分实现

支持的客户端版本:cn-最新(ios/windows/android/是cn客户端都支持)

# 本项目仅供学习与交流,请在下载后的24H内删除,相关版权归 **[四川星期零信息科技有限公司 ©版权所有](https://www.nekotan7.com)**

## 如何获取
1. 自编译
2. 前往[releases](https://github.com/BanterSR/Lolo/releases)或[actions](https://github.com/BanterSR/Lolo/actions)下载最新版
  

## 使用方法
1. 下载运行必要文件 [data](https://github.com/BanterSR/Lolo/tree/main/data)和[LoloResource](https://github.com/BanterSR/LoloResource) 分别保存到`data`文件夹和`Resource`文件夹中
2. 用各种方法代理客户端http:`http://dsp-prod-of.inutan.com:18881/*` `https://sdkapi-sz.inutan.com/*` 和`http://sdkapi-of.inutan.com/*` 到服务器https （默认:`https://127.0.0.1:4430`）
3. 使用`main/main.exe config gen`生成或补齐`config.json`修改`config.json`文件
4. 启动Lolo和客户端

## 更多提示
1. 更简单的的安卓代理方法(在模拟器中可能没有效果):[AnimeGamesProxy](https://github.com/Xuoos/AnimeGamesProxy/releases)
2. 虽然默认支持tls但是依然推荐使用http,并通过反向代理使用https
3. 如果你的pc端无法登录,请关闭config.json中的ChecnToken
4. 如果你想使用其他sdk,这里有一个接入示例:[Lolo-sdk](https://github.com/BanterSR/Lolo-sdk/blob/main/src/handlers/token.rs#L91)

## 已实现功能

- [x] 登录
  - [x] 自动注册
  - [x] 账号密码登录
  - [x] 服务发现
- [x] 账号
  - [x] 修改昵称
- [x] 场景
  - [x] 多人游戏
  - [x] 切换房间
  - [x] 切换场景
  - [x] 时间系统
  - [x] 天气系统
  - [x] 互动
    - [x] 载具
    - [x] 椅子
  - [x] 私人房间
  - [x] 演奏
  - [x] 传送锚点
- [x] 花园
  - [x] 花园设置
  - [x] 花园预设
  - [x] 摆放家具
  - [x] 摆放角色
- [ ] 照片墙
- [x] 队伍
  - [x] 队伍切换
- [x] 背包
- [x] 卡池
  - [x] 新手卡池
  - [x] 普通卡池
  - [x] 限定卡池
  - [x] 服装卡池
- [x] 角色
  - [x] 服装更换
  - [x] 更新装备
  - [x] 角色升级
  - [x] 角色进阶
  - [x] 角色升星
  - [x] 角色技能升级
  - [x] 角色成就
  - [ ] 服装染色支持(无法还原算法 不考虑实现)
- [x] 好友
  - [x] 添加好友
  - [x] 删除好友
  - [x] 拉黑好友
  - [x] 搜索玩家
  - [x] 附近的人
  - [x] 传送到好友身边
  - [x] 修改好友备注
- [x] 聊天
  - [x] 频道
  - [x] 世界
  - [ ] 系统
  - [x] 私聊
- [x] 商店
  - [x] 货币购买
  - [ ] 付费购买
  - [x] 角色纪念
- [ ] 战斗
- [ ] 深渊
- [x] 邮箱(不考虑)
- [x] GM(不考虑)
- [x] 收集
  - [x] 收藏收集
  - [x] 月亮收集
  - [x] 资源点采集
  - [ ] 宝箱收集

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

        var proxyHost = "127.0.0.1:8080";
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

## 数据库安装参考

postgres:
```
docker run -d \
  # 容器名称
  --name postgres \
  # PostgreSQL 环境变量配置
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_USER=user \
  # 端口映射：主机5432 -> 容器5432
  -p 5432:5432 \
  # 数据卷映射：主机目录 -> 容器目录
  -v /data/postgres_data:/var/lib/postgresql \
  # 镜像名称
  docker.1ms.run/postgres:latest \
  # PostgreSQL 运行时参数
  -c shared_buffers=1024MB \
  -c max_connections=200
```