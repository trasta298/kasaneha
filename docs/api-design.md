# API設計

## API概要

KasanehaのバックエンドAPIは、RESTfulな設計原則に従って構築されます。

### 基本情報
- **Base URL**: `http://localhost:8080/api/v1`
- **認証方式**: JWT Token (Authorization: Bearer {token})
- **データ形式**: JSON
- **文字エンコーディング**: UTF-8

## APIエンドポイント

### 1. 認証関連

#### POST /auth/login
ユーザーログイン

```typescript
// Request
interface LoginRequest {
  username: string;
  password: string;
}

// Response
interface LoginResponse {
  token: string;
  user: {
    id: string;
    username: string;
    created_at: string;
  };
}
```

#### POST /auth/register
ユーザー登録

```typescript
// Request
interface RegisterRequest {
  username: string;
  password: string;
  email?: string;
}

// Response: Same as LoginResponse
```

### 2. チャットセッション関連

#### GET /sessions/today
今日のチャットセッション取得

```typescript
// Response
interface TodaySessionResponse {
  session: {
    id: string;
    date: string; // YYYY-MM-DD
    status: 'active' | 'completed';
    created_at: string;
    updated_at: string;
  } | null;
}
```

#### POST /sessions
新しいチャットセッション作成

```typescript
// Request
interface CreateSessionRequest {
  date: string; // YYYY-MM-DD
}

// Response
interface CreateSessionResponse {
  session: {
    id: string;
    date: string;
    status: 'active';
    created_at: string;
    updated_at: string;
  };
  initial_message: {
    id: string;
    content: string;
    sender: 'ai';
    timestamp: string;
  };
}
```

#### GET /sessions/:sessionId/messages
セッションのメッセージ一覧取得

```typescript
// Response
interface MessagesResponse {
  messages: Array<{
    id: string;
    content: string;
    sender: 'user' | 'ai';
    timestamp: string;
    metadata?: Record<string, any>;
  }>;
  session: {
    id: string;
    date: string;
    status: 'active' | 'completed';
  };
}
```

#### POST /sessions/:sessionId/messages
メッセージ送信

```typescript
// Request
interface SendMessageRequest {
  content: string;
}

// Response
interface SendMessageResponse {
  user_message: {
    id: string;
    content: string;
    sender: 'user';
    timestamp: string;
  };
  ai_response: {
    id: string;
    content: string;
    sender: 'ai';
    timestamp: string;
  };
}
```

#### PUT /sessions/:sessionId/complete
セッション完了

```typescript
// Response
interface CompleteSessionResponse {
  session: {
    id: string;
    status: 'completed';
    updated_at: string;
  };
  analysis_job_id: string; // 非同期分析ジョブID
}
```

### 3. 履歴・分析関連

#### GET /sessions
セッション履歴一覧取得

```typescript
// Query Parameters
interface SessionsQuery {
  year?: number;
  month?: number;
  limit?: number;
  offset?: number;
}

// Response
interface SessionsResponse {
  sessions: Array<{
    id: string;
    date: string;
    status: 'active' | 'completed';
    message_count: number;
    has_analysis: boolean;
    created_at: string;
    updated_at: string;
  }>;
  pagination: {
    total: number;
    limit: number;
    offset: number;
  };
}
```

#### GET /sessions/:sessionId/analysis
セッション分析データ取得

```typescript
// Response
interface AnalysisResponse {
  analysis: {
    id: string;
    session_id: string;
    summary: string;
    emotional_state: {
      primary_emotion: string;
      emotions: Record<string, number>; // emotion: score (0-1)
      confidence: number;
    };
    behavioral_insights: string[];
    tension_score: number; // 0-100
    relative_score: number; // -50 to +50 (compared to user average)
    keywords: string[];
    created_at: string;
  } | null;
}
```

#### GET /analysis/scores
テンションスコア履歴取得

```typescript
// Query Parameters
interface ScoresQuery {
  start_date?: string; // YYYY-MM-DD
  end_date?: string;   // YYYY-MM-DD
  granularity?: 'daily' | 'weekly' | 'monthly';
}

// Response
interface ScoresResponse {
  scores: Array<{
    date: string;
    tension_score: number;
    relative_score: number;
    session_id: string;
  }>;
  statistics: {
    average: number;
    min: number;
    max: number;
    trend: 'up' | 'down' | 'stable';
  };
}
```

### 4. カレンダー関連

#### GET /calendar/:year/:month
月間カレンダーデータ取得

```typescript
// Response
interface CalendarResponse {
  month_data: {
    year: number;
    month: number;
    days: Array<{
      date: string; // YYYY-MM-DD
      has_session: boolean;
      tension_score?: number;
      status?: 'active' | 'completed';
    }>;
  };
}
```

## エラーハンドリング

### エラーレスポンス形式

```typescript
interface ErrorResponse {
  error: {
    code: string;
    message: string;
    details?: Record<string, any>;
  };
}
```

### 主要エラーコード

| HTTPステータス | エラーコード | 説明 |
|---------------|-------------|------|
| 400 | `INVALID_REQUEST` | リクエストパラメータが不正 |
| 401 | `UNAUTHORIZED` | 認証が必要 |
| 403 | `FORBIDDEN` | アクセス権限なし |
| 404 | `NOT_FOUND` | リソースが見つからない |
| 409 | `SESSION_EXISTS` | 今日のセッションが既に存在 |
| 429 | `RATE_LIMIT_EXCEEDED` | レート制限に達した |
| 500 | `INTERNAL_ERROR` | サーバー内部エラー |
| 503 | `AI_SERVICE_UNAVAILABLE` | Gemini APIが利用不可 |

## レート制限

### 制限ルール
- **メッセージ送信**: 1分間に20回まで
- **セッション作成**: 1日1回まで
- **分析取得**: 1分間に60回まで

### レスポンスヘッダー
```
X-RateLimit-Limit: 20
X-RateLimit-Remaining: 19
X-RateLimit-Reset: 1640995200
```

## WebSocket API (将来拡張)

リアルタイム機能のために、将来的にWebSocket APIを追加予定。

### エンドポイント
- `ws://localhost:8080/ws/sessions/:sessionId`

### メッセージ形式
```typescript
interface WebSocketMessage {
  type: 'message' | 'typing' | 'analysis_complete';
  data: any;
  timestamp: string;
}
```

## API仕様管理

- **OpenAPI 3.0**: 仕様書の自動生成
- **Swagger UI**: 開発時のAPI探索用
- **Postman Collection**: APIテスト用 