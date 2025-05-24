# データベース設計

## データベース概要

PostgreSQL 15以上を使用し、日記アプリケーションに必要なデータを効率的に格納・検索できるよう設計されています。

## テーブル設計

### 1. users テーブル
ユーザー情報を管理

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    timezone VARCHAR(50) DEFAULT 'UTC',
    
    -- インデックス
    CONSTRAINT users_username_check CHECK (length(username) >= 3),
    CONSTRAINT users_email_check CHECK (email ~* '^[^@]+@[^@]+\.[^@]+$')
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;
```

### 2. chat_sessions テーブル
日次チャットセッションを管理

```sql
CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_date DATE NOT NULL,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'completed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- 1ユーザー1日1セッションの制約
    UNIQUE(user_id, session_date)
);

CREATE INDEX idx_chat_sessions_user_id ON chat_sessions(user_id);
CREATE INDEX idx_chat_sessions_date ON chat_sessions(session_date);
CREATE INDEX idx_chat_sessions_user_date ON chat_sessions(user_id, session_date);
CREATE INDEX idx_chat_sessions_status ON chat_sessions(status);
```

### 3. messages テーブル
チャットメッセージを格納

```sql
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    sender VARCHAR(10) NOT NULL CHECK (sender IN ('user', 'ai')),
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}'::jsonb,
    
    -- パフォーマンス用の順序保証
    sequence_number INTEGER NOT NULL
);

CREATE INDEX idx_messages_session_id ON messages(session_id);
CREATE INDEX idx_messages_session_sequence ON messages(session_id, sequence_number);
CREATE INDEX idx_messages_created_at ON messages(created_at);
CREATE INDEX idx_messages_sender ON messages(sender);

-- sequenceの自動採番用
CREATE SEQUENCE message_sequence_seq;
```

### 4. analyses テーブル
Geminiによる分析結果を格納

```sql
CREATE TABLE analyses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    summary TEXT NOT NULL,
    emotional_state JSONB NOT NULL, -- 感情状態データ
    behavioral_insights JSONB NOT NULL, -- 行動洞察データ
    tension_score INTEGER NOT NULL CHECK (tension_score >= 0 AND tension_score <= 100),
    relative_score INTEGER CHECK (relative_score >= -50 AND relative_score <= 50),
    keywords JSONB DEFAULT '[]'::jsonb,
    raw_analysis_data JSONB, -- Geminiからの生データ
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 1セッション1分析の制約
    UNIQUE(session_id)
);

CREATE INDEX idx_analyses_session_id ON analyses(session_id);
CREATE INDEX idx_analyses_tension_score ON analyses(tension_score);
CREATE INDEX idx_analyses_created_at ON analyses(created_at);

-- 感情状態検索用のGINインデックス
CREATE INDEX idx_analyses_emotional_state ON analyses USING GIN (emotional_state);
CREATE INDEX idx_analyses_keywords ON analyses USING GIN (keywords);
```

### 5. user_statistics テーブル
ユーザーの統計情報キャッシュ

```sql
CREATE TABLE user_statistics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_sessions INTEGER DEFAULT 0,
    average_tension_score DECIMAL(5,2),
    min_tension_score INTEGER,
    max_tension_score INTEGER,
    most_common_emotions JSONB DEFAULT '[]'::jsonb,
    last_calculated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 1ユーザー1統計レコード
    UNIQUE(user_id)
);

CREATE INDEX idx_user_statistics_user_id ON user_statistics(user_id);
```

## データ型定義

### JSONBフィールドの構造

#### emotional_state (analyses テーブル)
```json
{
  "primary_emotion": "happy",
  "emotions": {
    "happiness": 0.8,
    "sadness": 0.1,
    "anger": 0.05,
    "fear": 0.05,
    "surprise": 0.0,
    "disgust": 0.0
  },
  "confidence": 0.85
}
```

#### behavioral_insights (analyses テーブル)
```json
[
  "積極的に将来の計画について話している",
  "人間関係について前向きな言及が多い",
  "仕事に対するストレスの兆候は見られない"
]
```

#### keywords (analyses テーブル)
```json
["仕事", "友達", "映画", "楽しい", "疲れた"]
```

## インデックス戦略

### 1. 主要検索パターン
- ユーザーの日次セッション検索
- 特定期間の分析データ取得
- テンションスコアの時系列分析

### 2. 複合インデックス
```sql
-- ユーザーの月次データ検索用
CREATE INDEX idx_sessions_user_month ON chat_sessions(user_id, EXTRACT(YEAR FROM session_date), EXTRACT(MONTH FROM session_date));

-- テンションスコアの時系列検索用
CREATE INDEX idx_analysis_score_date ON analyses(
  (SELECT user_id FROM chat_sessions WHERE id = session_id),
  tension_score,
  created_at
);
```

### 3. 部分インデックス
```sql
-- アクティブセッションのみ
CREATE INDEX idx_active_sessions ON chat_sessions(user_id, session_date) WHERE status = 'active';

-- 分析済みセッションのみ
CREATE INDEX idx_analyzed_sessions ON chat_sessions(id) WHERE id IN (SELECT session_id FROM analyses);
```

## パーティショニング戦略

### 日付ベースパーティショニング（将来拡張）

```sql
-- 年単位でのパーティショニング例
CREATE TABLE chat_sessions_2024 PARTITION OF chat_sessions
FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');

CREATE TABLE messages_2024 PARTITION OF messages
FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');
```

## データマイグレーション

### 初期スキーマ作成
```sql
-- migrations/001_initial_schema.sql
-- 上記のテーブル定義をすべて含む

-- migrations/002_add_timezone_support.sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS timezone VARCHAR(50) DEFAULT 'UTC';

-- migrations/003_add_user_statistics.sql
-- user_statisticsテーブルの追加
```

## パフォーマンス最適化

### 1. 接続プール設定
```go
config := pgxpool.Config{
    MaxConns:        30,
    MinConns:        5,
    MaxConnLifetime: time.Hour,
    MaxConnIdleTime: time.Minute * 30,
}
```

### 2. クエリ最適化
```sql
-- 効率的な日次データ取得
EXPLAIN ANALYZE
SELECT s.*, 
       COUNT(m.id) as message_count,
       a.tension_score
FROM chat_sessions s
LEFT JOIN messages m ON s.id = m.session_id
LEFT JOIN analyses a ON s.id = a.session_id
WHERE s.user_id = $1 
  AND s.session_date >= $2 
  AND s.session_date <= $3
GROUP BY s.id, a.tension_score
ORDER BY s.session_date DESC;
```

### 3. 統計情報の更新
```sql
-- 定期的な統計情報更新
ANALYZE chat_sessions;
ANALYZE messages;
ANALYZE analyses;
```

## バックアップ戦略

### 1. 日次バックアップ
```bash
pg_dump -h localhost -U kasaneha -d kasaneha_db --compress=9 > backup_$(date +%Y%m%d).sql.gz
```

### 2. ポイントインタイム復旧
```sql
-- WALアーカイビングの有効化
archive_mode = on
archive_command = 'cp %p /backup/archive/%f'
```

## セキュリティ考慮事項

### 1. 行レベルセキュリティ
```sql
-- ユーザーは自分のデータのみアクセス可能
ALTER TABLE chat_sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY user_data_policy ON chat_sessions
FOR ALL TO application_user
USING (user_id = current_setting('app.current_user_id')::uuid);
```

### 2. 暗号化
```sql
-- 機密データの暗号化（拡張機能使用）
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- パスワードハッシュ化
UPDATE users SET password_hash = crypt('password', gen_salt('bf', 10));
``` 