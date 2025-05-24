# 開発ガイド

## 環境構築

### 前提条件
- Docker & Docker Compose
- Node.js 18+
- Go 1.21+
- Git

### 1. リポジトリクローン
```bash
git clone <repository-url>
cd kasaneha
```

### 2. 環境変数設定
```bash
# .env.local を作成
cp .env.example .env.local

# 必要な環境変数を設定
GEMINI_API_KEY=your_gemini_api_key_here
DATABASE_URL=postgres://kasaneha:password@localhost:5432/kasaneha_db
JWT_SECRET=your_jwt_secret_here
```

### 3. Docker Compose起動
```bash
# 全サービス起動
docker-compose up -d

# または開発用
docker-compose -f docker-compose.dev.yml up -d
```

### 4. データベース初期化
```bash
# マイグレーション実行
cd backend
go run cmd/migrate/main.go up

# テストデータ投入（オプション）
go run cmd/seed/main.go
```

## 開発ワークフロー

### フロントエンド開発

#### セットアップ
```bash
cd frontend
npm install

# 開発サーバー起動
npm run dev
# http://localhost:4321 でアクセス
```

#### 主要コマンド
```bash
# 型チェック
npm run type-check

# ビルド
npm run build

# プレビュー
npm run preview

# コード整形
npm run format

# Linting
npm run lint
```

#### コンポーネント開発
新しいコンポーネントを作成する際のテンプレート：

```typescript
// src/components/example/ExampleComponent.astro
---
export interface Props {
  title: string;
  subtitle?: string;
  variant?: 'primary' | 'secondary';
}

const { title, subtitle, variant = 'primary' } = Astro.props;
---

<div class={`component-container ${variant}`}>
  <h2>{title}</h2>
  {subtitle && <p>{subtitle}</p>}
</div>

<style>
  .component-container {
    @apply p-4 rounded-lg;
  }
  
  .primary {
    @apply bg-blue-100 text-blue-800;
  }
  
  .secondary {
    @apply bg-gray-100 text-gray-800;
  }
</style>
```

### バックエンド開発

#### セットアップ
```bash
cd backend
go mod tidy

# 開発サーバー起動（Hot Reload）
air
# または
go run cmd/api/main.go
```

#### 主要コマンド
```bash
# テスト実行
go test ./...

# カバレッジ確認
go test -cover ./...

# ベンチマーク
go test -bench=. ./...

# モック生成
go generate ./...

# APIドキュメント生成
swag init -g cmd/api/main.go
```

#### 新しいエンドポイント追加
1. `internal/handler/` にハンドラーを作成
2. `internal/service/` にビジネスロジックを実装
3. `internal/repository/` にデータアクセス層を実装
4. `cmd/api/routes.go` にルートを追加

```go
// internal/handler/example_handler.go
package handler

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/render"
)

type ExampleHandler struct {
    service ExampleService
}

func NewExampleHandler(service ExampleService) *ExampleHandler {
    return &ExampleHandler{service: service}
}

func (h *ExampleHandler) GetExample(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    
    result, err := h.service.GetExample(r.Context(), id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    render.JSON(w, r, result)
}
```

## テスト戦略

### フロントエンドテスト

#### ユニットテスト（Vitest）
```typescript
// src/utils/__tests__/chatUtils.test.ts
import { describe, it, expect } from 'vitest';
import { formatMessage } from '../chatUtils';

describe('chatUtils', () => {
  it('should format message correctly', () => {
    const result = formatMessage('Hello, world!');
    expect(result).toBe('Hello, world!');
  });
});
```

#### E2Eテスト（Playwright）
```typescript
// tests/chat.spec.ts
import { test, expect } from '@playwright/test';

test('should send and receive messages', async ({ page }) => {
  await page.goto('/');
  
  // メッセージ入力
  await page.fill('#message-input', 'こんにちは');
  await page.click('#send-button');
  
  // AI応答を待機
  await expect(page.locator('.ai-message')).toBeVisible();
});
```

### バックエンドテスト

#### ユニットテスト
```go
// internal/service/chat_service_test.go
package service

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestChatService_SendMessage(t *testing.T) {
    mockRepo := new(MockChatRepository)
    mockAI := new(MockAIClient)
    
    service := NewChatService(mockRepo, mockAI)
    
    mockRepo.On("SaveMessage", mock.Anything, mock.Anything).Return(nil)
    mockAI.On("GenerateResponse", mock.Anything, mock.Anything).Return("Hello!", nil)
    
    result, err := service.SendMessage(context.Background(), "sessionID", "Hello")
    
    assert.NoError(t, err)
    assert.Equal(t, "Hello!", result.Content)
    mockRepo.AssertExpectations(t)
    mockAI.AssertExpectations(t)
}
```

#### 統合テスト
```go
// internal/handler/integration_test.go
func TestChatHandler_Integration(t *testing.T) {
    // テスト用DBセットアップ
    db := setupTestDB(t)
    defer db.Close()
    
    // サーバー起動
    server := setupTestServer(db)
    defer server.Close()
    
    // テストリクエスト
    resp, err := http.Post(
        server.URL+"/api/v1/sessions/test/messages",
        "application/json",
        strings.NewReader(`{"content":"Hello"}`),
    )
    
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

## データベース操作

### マイグレーション

#### 新しいマイグレーション作成
```bash
# マイグレーションファイル作成
migrate create -ext sql -dir migrations add_new_table
```

#### マイグレーション実行
```bash
# UP
migrate -path migrations -database $DATABASE_URL up

# DOWN
migrate -path migrations -database $DATABASE_URL down 1

# 特定バージョンまで
migrate -path migrations -database $DATABASE_URL goto 2
```

### データシード

```go
// cmd/seed/main.go
package main

import (
    "context"
    "log"
)

func main() {
    db := setupDB()
    defer db.Close()
    
    // テストユーザー作成
    user := &User{
        Username: "testuser",
        Email:    "test@example.com",
        PasswordHash: hashPassword("password"),
    }
    
    if err := db.CreateUser(context.Background(), user); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Seed data created successfully")
}
```

## CI/CD設定

### GitHub Actions

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test-frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - run: cd frontend && npm ci
      - run: cd frontend && npm run type-check
      - run: cd frontend && npm run test
      - run: cd frontend && npm run build

  test-backend:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: password
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: 1.21
      - run: cd backend && go test ./...
```

## デバッグ手順

### フロントエンドデバッグ

#### ブラウザ開発者ツール
```typescript
// デバッグ用のログ関数
export function debugLog(message: string, data?: any) {
  if (import.meta.env.DEV) {
    console.log(`[DEBUG] ${message}`, data);
  }
}

// 状態デバッグ
export function debugStore() {
  if (import.meta.env.DEV) {
    console.log('Current stores:', {
      currentSession: currentSession.get(),
      messages: messages.get(),
      isLoading: isLoading.get(),
    });
  }
}
```

#### ネットワークリクエストデバッグ
```typescript
// APIクライアントでのログ
private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  if (import.meta.env.DEV) {
    console.log(`API Request: ${endpoint}`, options);
  }
  
  const response = await fetch(url, { ...options, headers });
  
  if (import.meta.env.DEV) {
    console.log(`API Response: ${endpoint}`, response.status, await response.clone().json());
  }
  
  return response.json();
}
```

### バックエンドデバッグ

#### ログ設定
```go
// internal/logger/logger.go
package logger

import (
    "os"
    "github.com/sirupsen/logrus"
)

func Setup() *logrus.Logger {
    log := logrus.New()
    
    if os.Getenv("ENV") == "development" {
        log.SetLevel(logrus.DebugLevel)
        log.SetFormatter(&logrus.TextFormatter{
            FullTimestamp: true,
        })
    } else {
        log.SetLevel(logrus.InfoLevel)
        log.SetFormatter(&logrus.JSONFormatter{})
    }
    
    return log
}
```

#### デバッガー使用（delve）
```bash
# delveインストール
go install github.com/go-delve/delve/cmd/dlv@latest

# デバッグモードで起動
dlv debug cmd/api/main.go
```

## パフォーマンス最適化

### フロントエンド最適化

#### バンドルサイズ分析
```bash
# Astroビルド解析
npm run build -- --analyze

# webpack-bundle-analyzer使用
npx webpack-bundle-analyzer dist/stats.json
```

#### 画像最適化
```typescript
// astro.config.mjs
export default defineConfig({
  image: {
    service: squooshImageService(),
  },
  vite: {
    build: {
      rollupOptions: {
        output: {
          manualChunks: {
            vendor: ['chart.js'],
            utils: ['./src/utils/index.ts'],
          },
        },
      },
    },
  },
});
```

### バックエンド最適化

#### プロファイリング
```go
import _ "net/http/pprof"

// main.goで有効化
if os.Getenv("ENV") == "development" {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}
```

#### メモリプロファイル確認
```bash
go tool pprof http://localhost:6060/debug/pprof/heap
go tool pprof http://localhost:6060/debug/pprof/profile
```

## トラブルシューティング

### よくある問題と解決法

#### 1. Docker関連
```bash
# コンテナ再ビルド
docker-compose down
docker-compose build --no-cache
docker-compose up -d

# ボリューム削除
docker volume prune
```

#### 2. データベース接続エラー
```bash
# PostgreSQL接続確認
docker exec -it kasaneha_postgres psql -U kasaneha -d kasaneha_db

# マイグレーション状態確認
migrate -path migrations -database $DATABASE_URL version
```

#### 3. Gemini API関連
```bash
# API キー確認
echo $GEMINI_API_KEY

# レート制限確認
curl -H "Authorization: Bearer $GEMINI_API_KEY" \
     https://generativelanguage.googleapis.com/v1/models
```

## 開発ツール推奨設定

### VSCode設定
```json
{
  "recommendations": [
    "astro-build.astro-vscode",
    "golang.go",
    "bradlc.vscode-tailwindcss",
    "ms-vscode.vscode-typescript-next"
  ],
  "settings": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.fixAll.eslint": true
    }
  }
}
``` 