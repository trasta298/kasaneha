# デプロイメント設計

## Docker設定

### 開発環境用 Docker Compose

```yaml
# docker-compose.dev.yml
version: '3.8'

services:
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile.dev
    container_name: kasaneha_frontend_dev
    ports:
      - "4321:4321"
    volumes:
      - ./frontend:/app
      - /app/node_modules
    environment:
      - NODE_ENV=development
      - PUBLIC_API_URL=http://localhost:8080/api/v1
    depends_on:
      - backend

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile.dev
    container_name: kasaneha_backend_dev
    ports:
      - "8080:8080"
      - "6060:6060" # pprof
    volumes:
      - ./backend:/app
    environment:
      - ENV=development
      - DATABASE_URL=postgres://kasaneha:password@postgres:5432/kasaneha_db?sslmode=disable
      - GEMINI_API_KEY=${GEMINI_API_KEY}
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:15-alpine
    container_name: kasaneha_postgres
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_USER=kasaneha
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=kasaneha_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backend/migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U kasaneha"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

### 本番環境用 Docker Compose

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  nginx:
    image: nginx:alpine
    container_name: kasaneha_nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf
      - ./nginx/ssl:/etc/nginx/ssl
      - static_files:/var/www/static
    depends_on:
      - frontend
      - backend
    restart: unless-stopped

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile.prod
    container_name: kasaneha_frontend
    volumes:
      - static_files:/app/dist
    environment:
      - NODE_ENV=production
      - PUBLIC_API_URL=https://api.kasaneha.app/api/v1
    restart: unless-stopped

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile.prod
    container_name: kasaneha_backend
    environment:
      - ENV=production
      - DATABASE_URL=postgres://kasaneha:${POSTGRES_PASSWORD}@postgres:5432/kasaneha_db?sslmode=require
      - GEMINI_API_KEY=${GEMINI_API_KEY}
      - JWT_SECRET=${JWT_SECRET}
      - REDIS_URL=redis://redis:6379
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped

  postgres:
    image: postgres:15-alpine
    container_name: kasaneha_postgres
    environment:
      - POSTGRES_USER=kasaneha
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - POSTGRES_DB=kasaneha_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U kasaneha"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    container_name: kasaneha_redis
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
  static_files:
```

## Dockerfile設定

### フロントエンド

#### 開発用
```dockerfile
# frontend/Dockerfile.dev
FROM node:18-alpine

WORKDIR /app

# パッケージファイルをコピーして依存関係をインストール
COPY package*.json ./
RUN npm ci

# ソースコードをコピー
COPY . .

# 開発サーバー起動
EXPOSE 4321
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
```

#### 本番用
```dockerfile
# frontend/Dockerfile.prod
FROM node:18-alpine AS builder

WORKDIR /app

# 依存関係をインストール
COPY package*.json ./
RUN npm ci

# ソースコードをコピーしてビルド
COPY . .
RUN npm run build

# Nginxで静的ファイルを配信
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx/default.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### バックエンド

#### 開発用
```dockerfile
# backend/Dockerfile.dev
FROM golang:1.21-alpine

# 開発ツールをインストール
RUN go install github.com/cosmtrek/air@latest

WORKDIR /app

# Go modulesを先にコピー
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# Airでホットリロード
EXPOSE 8080 6060
CMD ["air"]
```

#### 本番用
```dockerfile
# backend/Dockerfile.prod
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 依存関係をダウンロード
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# バイナリをビルド
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/api/main.go

# 最小限の実行環境
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# バイナリをコピー
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080
CMD ["./main"]
```

## Nginx設定

```nginx
# nginx/nginx.conf
events {
    worker_connections 1024;
}

http {
    upstream backend {
        server backend:8080;
    }

    upstream frontend {
        server frontend:80;
    }

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=chat:10m rate=5r/s;

    server {
        listen 80;
        server_name kasaneha.app www.kasaneha.app;

        # HTTPS リダイレクト
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name kasaneha.app www.kasaneha.app;

        # SSL設定
        ssl_certificate /etc/nginx/ssl/fullchain.pem;
        ssl_certificate_key /etc/nginx/ssl/privkey.pem;
        ssl_session_timeout 1d;
        ssl_session_cache shared:SSL:50m;
        ssl_session_tickets off;

        # セキュリティヘッダー
        add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload";
        add_header X-Frame-Options DENY;
        add_header X-Content-Type-Options nosniff;
        add_header X-XSS-Protection "1; mode=block";

        # API リクエスト
        location /api/ {
            limit_req zone=api burst=20 nodelay;
            
            proxy_pass http://backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            
            # タイムアウト設定
            proxy_connect_timeout 5s;
            proxy_send_timeout 60s;
            proxy_read_timeout 60s;
        }

        # チャットAPI（より厳しいレート制限）
        location /api/v1/sessions/ {
            limit_req zone=chat burst=10 nodelay;
            
            proxy_pass http://backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # 静的ファイル
        location / {
            proxy_pass http://frontend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            # キャッシュ設定
            location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
                expires 1y;
                add_header Cache-Control "public, immutable";
            }
        }

        # ヘルスチェック
        location /health {
            access_log off;
            return 200 "healthy\n";
            add_header Content-Type text/plain;
        }
    }
}
```

## 環境変数管理

### .env.example
```bash
# Database
DATABASE_URL=postgres://kasaneha:password@localhost:5432/kasaneha_db
POSTGRES_PASSWORD=your_secure_password_here

# AI Service
GEMINI_API_KEY=your_gemini_api_key_here

# Security
JWT_SECRET=your_jwt_secret_here

# Redis (本番環境)
REDIS_URL=redis://localhost:6379

# Application
ENV=development
PORT=8080
HOST=0.0.0.0

# Frontend
PUBLIC_API_URL=http://localhost:8080/api/v1

# Monitoring (オプション)
SENTRY_DSN=your_sentry_dsn_here
```

### 本番環境での環境変数管理
```bash
# .env.prod (Git管理外)
DATABASE_URL=postgres://kasaneha:$(cat /run/secrets/postgres_password)@postgres:5432/kasaneha_db?sslmode=require
GEMINI_API_KEY=$(cat /run/secrets/gemini_api_key)
JWT_SECRET=$(cat /run/secrets/jwt_secret)
POSTGRES_PASSWORD=$(cat /run/secrets/postgres_password)
```

## デプロイメント戦略

### ローリングデプロイメント

```bash
#!/bin/bash
# deploy.sh

set -e

echo "Starting deployment..."

# 環境変数確認
if [ -z "$GEMINI_API_KEY" ]; then
    echo "Error: GEMINI_API_KEY is not set"
    exit 1
fi

# バックアップ作成
echo "Creating database backup..."
docker exec kasaneha_postgres pg_dump -U kasaneha kasaneha_db > "backup_$(date +%Y%m%d_%H%M%S).sql"

# 新しいイメージをビルド
echo "Building new images..."
docker-compose -f docker-compose.prod.yml build

# データベースマイグレーション
echo "Running database migrations..."
docker-compose -f docker-compose.prod.yml run --rm backend ./main migrate up

# サービスを順次更新
echo "Updating backend..."
docker-compose -f docker-compose.prod.yml up -d --no-deps backend

echo "Waiting for backend to be healthy..."
sleep 10

echo "Updating frontend..."
docker-compose -f docker-compose.prod.yml up -d --no-deps frontend

echo "Updating nginx..."
docker-compose -f docker-compose.prod.yml up -d --no-deps nginx

echo "Deployment completed successfully!"
```

### ブルーグリーンデプロイメント

```yaml
# docker-compose.blue-green.yml
version: '3.8'

services:
  # Blue環境
  backend-blue:
    build:
      context: ./backend
      dockerfile: Dockerfile.prod
    container_name: kasaneha_backend_blue
    environment:
      - ENV=production
      - DATABASE_URL=${DATABASE_URL}
      - GEMINI_API_KEY=${GEMINI_API_KEY}
    restart: unless-stopped

  frontend-blue:
    build:
      context: ./frontend
      dockerfile: Dockerfile.prod
    container_name: kasaneha_frontend_blue
    restart: unless-stopped

  # Green環境
  backend-green:
    build:
      context: ./backend
      dockerfile: Dockerfile.prod
    container_name: kasaneha_backend_green
    environment:
      - ENV=production
      - DATABASE_URL=${DATABASE_URL}
      - GEMINI_API_KEY=${GEMINI_API_KEY}
    restart: unless-stopped

  frontend-green:
    build:
      context: ./frontend
      dockerfile: Dockerfile.prod
    container_name: kasaneha_frontend_green
    restart: unless-stopped
```

## モニタリング設定

### Prometheus設定

```yaml
# monitoring/prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'kasaneha-backend'
    static_configs:
      - targets: ['backend:8080']
    metrics_path: /metrics

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres:5432']

  - job_name: 'nginx'
    static_configs:
      - targets: ['nginx:80']
```

### Grafana ダッシュボード

```json
{
  "dashboard": {
    "title": "Kasaneha Metrics",
    "panels": [
      {
        "title": "API Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])"
          }
        ]
      },
      {
        "title": "Database Connections",
        "type": "graph",
        "targets": [
          {
            "expr": "pg_stat_database_numbackends"
          }
        ]
      },
      {
        "title": "Gemini API Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(gemini_api_requests_total[5m])"
          }
        ]
      }
    ]
  }
}
```

## セキュリティ設定

### ファイアウォール設定

```bash
# UFW設定例
sudo ufw default deny incoming
sudo ufw default allow outgoing

# SSH
sudo ufw allow 22/tcp

# HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# PostgreSQL (必要に応じて)
sudo ufw allow from 10.0.0.0/8 to any port 5432

sudo ufw enable
```

### SSL証明書取得

```bash
# Let's Encrypt使用
sudo apt-get update
sudo apt-get install certbot

# 証明書取得
sudo certbot certonly --standalone -d kasaneha.app -d www.kasaneha.app

# 自動更新設定
echo "0 12 * * * /usr/bin/certbot renew --quiet" | sudo crontab -
```

## ログ管理

### ログ収集設定

```yaml
# docker-compose.logging.yml
version: '3.8'

services:
  fluentd:
    image: fluentd:latest
    container_name: kasaneha_fluentd
    volumes:
      - ./logging/fluent.conf:/fluentd/etc/fluent.conf
      - fluentd_logs:/var/log/fluentd
    ports:
      - "24224:24224"

  elasticsearch:
    image: elasticsearch:7.14.0
    container_name: kasaneha_elasticsearch
    environment:
      - discovery.type=single-node
    volumes:
      - elasticsearch_data:/usr/share/elasticsearch/data

  kibana:
    image: kibana:7.14.0
    container_name: kasaneha_kibana
    ports:
      - "5601:5601"
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200

volumes:
  fluentd_logs:
  elasticsearch_data:
```

## バックアップ戦略

### 自動バックアップスクリプト

```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

# データベースバックアップ
docker exec kasaneha_postgres pg_dump -U kasaneha kasaneha_db | gzip > "$BACKUP_DIR/db_backup_$TIMESTAMP.sql.gz"

# ファイルバックアップ
tar -czf "$BACKUP_DIR/files_backup_$TIMESTAMP.tar.gz" ./uploads

# 古いバックアップを削除
find $BACKUP_DIR -name "*.gz" -mtime +$RETENTION_DAYS -delete

# S3にアップロード（オプション）
# aws s3 cp "$BACKUP_DIR/db_backup_$TIMESTAMP.sql.gz" s3://kasaneha-backups/

echo "Backup completed: $TIMESTAMP"
```

### リストア手順

```bash
#!/bin/bash
# restore.sh

BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

echo "Restoring from $BACKUP_FILE..."

# データベース復元
gunzip -c "$BACKUP_FILE" | docker exec -i kasaneha_postgres psql -U kasaneha -d kasaneha_db

echo "Restore completed"
``` 