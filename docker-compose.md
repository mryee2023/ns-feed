# Docker Compose 使用说明

## 目录结构
```
.
├── docker-compose.yml    # Docker Compose 配置文件
├── .env                 # 环境变量配置（可选）
└── data/
    ├── config.yaml     # 配置文件（可选）
    └── sqlite.db       # SQLite数据库文件
```

## 配置方式

支持两种配置方式：

### 1. 配置文件方式

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

### 2. 环境变量方式

1. 创建必要的目录：
```bash
mkdir -p data
touch data/sqlite.db
```

2. 创建并修改环境变量文件：
```bash
cp .env.example .env
vim .env
```

配置以下环境变量：
- `TG_TOKEN`: Telegram Bot Token
- `ADMIN_ID`: 管理员的 Telegram Chat ID

## 启动和管理

1. 启动服务：
```bash
docker-compose up -d
```

2. 查看日志：
```bash
docker-compose logs -f
```

3. 停止服务：
```bash
docker-compose down
```

## 命令行参数说明

当使用环境变量方式时，支持以下命令行参数：

基本参数：
- `--token`: Telegram Bot Token（必需）
- `--admin`: 管理员的 Telegram Chat ID（必需）
- `--db`: SQLite 数据库文件路径（默认：/db/sqlite.db）
- `--feed`: NodeSeek RSS feed URL（默认：https://rss.nodeseek.com）
- `--interval`: RSS抓取间隔（默认：10s）
- `--port`: HTTP 服务端口（默认：:8080）

## 配置说明

- 容器会自动重启（unless-stopped）
- 时区设置为 Asia/Shanghai
- 日志配置：
  - 最大文件大小：10MB
  - 保留3个历史文件
- 配置文件以只读方式挂载（ro）
- SQLite数据库文件可读写
- 优先使用环境变量配置，如果环境变量未设置则使用配置文件
