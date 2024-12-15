# Docker Compose 使用说明

## 目录结构
```
.
├── docker-compose.yml
└── data/
    ├── config.yaml    # 配置文件
    └── sqlite.db      # SQLite数据库文件
```

## 使用步骤

1. 创建必要的目录和文件：
```bash
mkdir -p data
touch data/sqlite.db
cp src/etc/config.simple.yaml data/config.yaml
```

2. 修改配置文件：
```bash
vim data/config.yaml
```

3. 启动服务：
```bash
docker-compose up -d
```

4. 查看日志：
```bash
docker-compose logs -f
```

5. 停止服务：
```bash
docker-compose down
```

6. 手动执行同步：
```bash
docker-compose exec ns-feed-bot ./ns-rss -f /etc/config.yaml -db /db/sqlite.db -sync
```

## 配置说明

- 容器会自动重启（unless-stopped）
- 时区设置为 Asia/Shanghai
- 日志配置：
  - 最大文件大小：10MB
  - 保留3个历史文件
- 配置文件以只读方式挂载（ro）
- SQLite数据库文件可读写
