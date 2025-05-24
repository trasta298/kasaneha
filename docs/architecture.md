# アーキテクチャ設計

## システム全体構成

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │    Backend      │    │   External      │
│   (Astro)       │    │    (Go/Chi)     │    │   Services      │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ PWA Application │◄──►│ REST API Server │◄──►│ Google Gemini   │
│ TypeScript      │    │ Chat Handler    │    │ API             │
│ Tailwind CSS    │    │ Analysis Engine │    │                 │
│ Service Worker  │    │ Auth Middleware │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   Database      │
                       │   (PostgreSQL)  │
                       ├─────────────────┤
                       │ Users           │
                       │ Chat Sessions   │
                       │ Messages        │
                       │ Analysis Data   │
                       │ Sentiment Scores│
                       └─────────────────┘
```

## コンポーネント詳細

### フロントエンド層

#### Astro Application
- **役割**: SPA（Single Page Application）として動作
- **主要機能**:
  - チャットインターフェース
  - カレンダー表示
  - グラフ・チャート表示
  - オフライン対応

#### PWA Features
- **Service Worker**: オフラインキャッシング、プッシュ通知
- **Manifest**: ホーム画面追加、ネイティブアプリライクな体験
- **Cache Strategy**: チャット履歴の本文データをローカルキャッシュ

### バックエンド層

#### API Server (Go + Chi)
- **役割**: RESTful APIサーバー
- **主要機能**:
  - チャットセッション管理
  - メッセージ送受信
  - 感情分析処理
  - データ永続化

#### AI Integration Module
- **役割**: Gemini APIとの統合
- **主要機能**:
  - 会話生成
  - 感情分析
  - サマリー生成
  - テンションスコア算出

### データ層

#### PostgreSQL Database
- **役割**: アプリケーションデータの永続化
- **設計原則**:
  - 正規化されたスキーマ
  - インデックス最適化
  - 日付ベースのパーティショニング

## データフロー

### 1. 日次チャット開始フロー
```
User → Frontend → Backend → Database
              ↓
    New Session Creation
              ↓
    Gemini AI → Generate Opening Message
```

### 2. メッセージ送信フロー
```
User Input → Frontend → Backend → Database (Save)
                    ↓
              Gemini API (Response Generation)
                    ↓
              Database (Save AI Response)
                    ↓
              Frontend (Display Response)
```

### 3. 分析・スコア算出フロー
```
Daily Chat Completion → Backend Analysis Engine
                    ↓
              Gemini API (Sentiment Analysis)
                    ↓
              Historical Data Comparison
                    ↓
              Tension Score Calculation
                    ↓
              Database (Save Analysis)
```

## セキュリティ考慮事項

### 認証・認可
- **JWT Token**: ステートレス認証
- **Session Management**: Redis使用（将来的な拡張）
- **Rate Limiting**: DDoS攻撃防止

### データ保護
- **暗号化**: 機密性の高いメッセージは暗号化保存
- **GDPR対応**: ユーザーデータの削除権利確保
- **API Key管理**: 環境変数での管理

## スケーラビリティ設計

### 水平スケーリング対応
- **ステートレス設計**: サーバー間でのセッション共有なし
- **データベース分散**: 日付ベースでのシャーディング準備
- **CDN対応**: 静的ファイルの配信最適化

### パフォーマンス最適化
- **接続プール**: データベース接続の効率化
- **キャッシング**: Redis使用（将来拡張）
- **非同期処理**: AI分析の非同期実行

## 監視・ログ設計

### ログ戦略
- **構造化ログ**: JSON形式での出力
- **ログレベル**: DEBUG, INFO, WARN, ERROR
- **分散トレーシング**: 将来的にOpenTelemetry導入

### メトリクス
- **アプリケーションメトリクス**: 
  - リクエスト数、レスポンス時間
  - Gemini API使用量
  - ユーザーアクティビティ
- **インフラメトリクス**: CPU、メモリ、ディスク使用量 