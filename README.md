# lolo

[![Discord](https://img.shields.io/badge/Join-Discord-blue?logo=discord&logoSize=auto)](https://discord.gg/GNKBE7X75d)

开放空间/OverField 服务端部分实现

目标客户端:1.1.12(PC 程序版本 `2026-10-21-14-05-50`,资源版本 `2026-11-07-22-39-35`)

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
5. 当前sdk仅支持cn-安卓/ios 客户端,其他区服/平台客户端需要登录请参考注意事项3关闭ChecnToken

## 实现状态

全部模块按目标客户端重新实现中,现有业务代码基于旧版本,不视为已实现。待实现清单见 [docs/requirements.md](docs/requirements.md)。GM 不在服务端提供,由 Momoko 管理平台负责。

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

## 数据库配置（读写分离 / 异步落库）

`config.json` 的 `DB` 段，字段说明：

```json
"DB": {
  "dbType": "sqlite",          // 类型：sqlite / mysql / postgres
  "dsn": "./db/lolo.db",       // 主库（写）连接串
  "readDsn": [],               // 从库（读）连接串列表，空=读写都走主库
  "maxIdleConns": 20,          // 最大空闲连接数
  "maxOpenConns": 40,          // 最大连接数
  "connMaxLifetime": "1h0m0s", // 连接最大复用时间
  "persistShardNum": 0,        // 异步落库分片数，0=CPU核数（上限 maxOpenConns）
  "persistBufSize": 1024       // 每分片队列缓冲
}
```

### 读写分离（readDsn）

`readDsn` 填**数据库主从复制中“从库”的连接串**：读自动走从库、写自动走主库，无需改代码。从库必须是**真正在复制主库**的实例（在数据库侧配好主从复制，或使用云厂商只读副本），程序只负责填连接串。

**SQLite** —— 进程内单文件，不支持读写分离，`readDsn` 留空：

```json
"DB": { "dbType": "sqlite", "dsn": "./db/lolo.db", "readDsn": [] }
```

**MySQL** —— DSN 格式 `用户:密码@tcp(主机:端口)/库名?参数`（`parseTime=True` 必填）：

```json
"DB": {
  "dbType": "mysql",
  "dsn": "lolo:pass@tcp(10.0.0.1:3306)/lolo?charset=utf8mb4&parseTime=True&loc=Local",
  "readDsn": [
    "lolo:pass@tcp(10.0.0.2:3306)/lolo?charset=utf8mb4&parseTime=True&loc=Local",
    "lolo:pass@tcp(10.0.0.3:3306)/lolo?charset=utf8mb4&parseTime=True&loc=Local"
  ]
}
```

**PostgreSQL** —— DSN 用 kv 形式（或 `postgres://用户:密码@主机:端口/库名?sslmode=disable`）：

```json
"DB": {
  "dbType": "postgres",
  "dsn": "host=10.0.0.1 user=lolo password=pass dbname=lolo port=5432 sslmode=disable TimeZone=Asia/Shanghai",
  "readDsn": [
    "host=10.0.0.2 user=lolo password=pass dbname=lolo port=5432 sslmode=disable TimeZone=Asia/Shanghai"
  ]
}
```

> - 多个 `readDsn` 会随机负载均衡；暂无真从库、只想验证配置时，可把 `readDsn` 临时填成主库地址。
> - 主从复制有延迟（从库读可能滞后主库几十毫秒~数秒）
