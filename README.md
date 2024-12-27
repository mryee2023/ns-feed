##NodeSeek Feed 关键字通知机器人

### 1. 机器人简介
NodeSeek Feed 关键字通知机器人是一个Telegram机器人，用于监控NodeSeek RSS指定关键字，并将文章标题、链接、发布时间等信息推送到指定群组、频道或个人。


### 2. 使用方法

#### 2.1 添加机器人

在Telegram中搜索 `@nsfeed_noc_bot` 并添加机器人。

#### 2.2 设置关键字
- 发送 `/start` 启动机器人
- 发送 `/help` 查看帮助
- 发送 `/add` 添加关键字
- 发送 `/list` 查看关键字列表
- 发送 `/delete` 删除关键字
- 发送 `/on` 启用关键字通知
- 发送 `/off` 关闭关键字通知
- 发送 `/quit` 退出机器人通知


### 3. 其他

当前已实现在频道中推送关键字通知，后续会增加个人中推送关键字通知的功能。


### 4. 安装说明

Docker安装
```shell
docker run -d --name ns-feed-bot -v /path/to/config.yaml:/etc/config.yaml -v /path/to/sqlite.db:/db/sqlite.db  i0x3eb/ns-feed-bot:latest
```

Docker Compose安装

[docker-compose.yml](docker-compose.yml)




### 5. 配置文件说明

```yaml
port: :8080   # 机器人监听端口，默认8080，便于uptime检测
tgToken: your_telegram_bot_token # 机器人Token
nsFeed: https://rss.nodeseek.com
adminId: 0 # 管理员ID,系统启动/退出时会发送通知，执行/status命令时可发送汇总数据
fetchTimeInterval: 10s   # RSS抓取时间间隔,最小10s

```