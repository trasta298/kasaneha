# Kasaneha AI日記アプリ

<div align="center">

![Kasaneha Logo](frontend/public/favicon.svg)

**AIと一緒に心の軌跡を記録する次世代日記アプリ**

[![Tech Stack](https://skillicons.dev/icons?i=go,ts,react,astro,tailwind,postgres,docker)](https://skillicons.dev)

</div>

## 🌟 概要

**Kasaneha**は、Google Gemini AIを搭載した革新的な日記アプリケーションです。AIアシスタント「かさね」との自然な対話を通じて、あなたの心の状態を分析し、感情の可視化とメンタルヘルスケアをサポートします。

## ✨ 主要機能

### 🤖 AI対話システム
- **Google Gemini AI**との自然な日本語対話
- 毎日のセッション管理（1日1スレッド）
- リアルタイムメッセージング体験

### 📊 感情分析・可視化
- **感情分析**: 対話内容から感情状態を自動分析
- **テンションスコア**: 0-100スケールでの心の状態数値化
- **インサイト生成**: AIによる傾向分析とアドバイス
- **統計・グラフ**: 長期的な感情推移の可視化

### 📱 PWA対応
- **オフライン機能**: Service Workerによるキャッシング
- **ネイティブ体験**: ホーム画面インストール対応
- **レスポンシブデザイン**: モバイル・デスクトップ最適化

### 🔒 セキュリティ
- **JWT認証**: セキュアなユーザー管理
- **データ保護**: 暗号化とプライバシー保護
- **CORS対応**: 安全なAPI通信

## 🏗️ 技術スタック

### バックエンド
- **言語**: Go 1.21+
- **フレームワーク**: Chi Router
- **データベース**: PostgreSQL
- **AI**: Google Gemini API
- **認証**: JWT

### フロントエンド
- **フレームワーク**: Astro 5.x + React 18
- **言語**: TypeScript
- **スタイリング**: Tailwind CSS
- **状態管理**: nanostores
- **PWA**: @vite-pwa/astro

### インフラ
- **コンテナ**: Docker + Docker Compose
- **プロキシ**: Nginx
- **開発**: Hot Reload対応

## 🚀 クイックスタート

### 前提条件
- Docker & Docker Compose
- Google Gemini API キー

### 1. リポジトリクローン
```bash
git clone https://github.com/your-org/kasaneha.git
cd kasaneha
```

### 2. 環境変数設定

#### バックエンド環境変数
```bash
# backend/.env
GEMINI_API_KEY=your_gemini_api_key_here
DATABASE_URL=postgres://kasaneha:password@postgres:5432/kasaneha_db?sslmode=disable
JWT_SECRET=your_jwt_secret_here
HOST=0.0.0.0
PORT=8080
```

#### フロントエンド環境変数
```bash
# フロントエンドからアクセスするAPIのベースURL
# ローカル開発時
PUBLIC_API_BASE_URL=http://localhost:8080

# プロダクション環境の例
# PUBLIC_API_BASE_URL=https://api.kasaneha.example.com
```

**注意**: フロントエンドはSPAなので、ブラウザから直接APIサーバーにアクセスします。
- ローカル開発: `http://localhost:8080`
- プロダクション: 実際のAPIサーバーのURL

### 3. アプリケーション起動
```bash
docker-compose up -d
```

### 4. アクセス
- **フロントエンド**: http://localhost:4321
- **バックエンドAPI**: http://localhost:8080
- **データベース**: localhost:5432

## 📁 プロジェクト構造

```
kasaneha/
├── backend/                 # Go バックエンド
│   ├── cmd/api/            # メインアプリケーション
│   ├── internal/           # ビジネスロジック
│   │   ├── ai/            # Gemini AI統合
│   │   ├── handler/       # HTTPハンドラー
│   │   ├── service/       # サービス層
│   │   ├── repository/    # データアクセス層
│   │   └── types/         # 型定義
│   └── migrations/        # データベースマイグレーション
├── frontend/               # Astro + React フロントエンド
│   ├── src/
│   │   ├── pages/         # ページコンポーネント
│   │   ├── components/    # 再利用可能コンポーネント
│   │   ├── stores/        # 状態管理
│   │   ├── api/           # API クライアント
│   │   └── types/         # TypeScript 型定義
│   └── public/            # 静的アセット
├── docker-compose.yml     # 開発環境設定
└── README.md
```

## 🛠️ API エンドポイント

### 認証
- `POST /api/v1/auth/register` - ユーザー登録
- `POST /api/v1/auth/login` - ログイン
- `GET /api/v1/auth/me` - ユーザー情報取得

### チャットセッション
- `GET /api/v1/sessions/today` - 今日のセッション取得
- `POST /api/v1/sessions` - 新規セッション作成
- `GET /api/v1/sessions` - セッション履歴
- `GET /api/v1/sessions/:id/messages` - メッセージ一覧
- `POST /api/v1/sessions/:id/messages` - メッセージ送信
- `PUT /api/v1/sessions/:id/complete` - セッション完了

### 分析・統計
- `GET /api/v1/sessions/:id/analysis` - セッション分析結果
- `POST /api/v1/sessions/:id/analysis` - 分析実行
- `GET /api/v1/analysis/scores` - テンションスコア履歴
- `GET /api/v1/analysis/insights` - 分析インサイト
- `GET /api/v1/calendar/:year/:month` - カレンダーデータ

## 📱 画面構成

| 画面 | 機能 | 説明 |
|------|------|------|
| **ログイン/登録** | 認証 | ユーザー認証・アカウント作成 |
| **ダッシュボード** | 概要表示 | インサイト・統計概要・クイックアクション |
| **チャット** | AI対話 | かさねとのリアルタイム対話 |
| **分析結果** | 可視化 | 感情分析・グラフ・統計詳細 |
| **履歴** | 振り返り | カレンダー・セッション一覧 |
| **設定** | 管理 | プロフィール・アプリ設定・データ管理 |

## 🎯 開発フロー

### 開発環境
```bash
# バックエンド開発
cd backend
go run cmd/api/main.go

# フロントエンド開発
cd frontend
npm run dev

# データベース
docker-compose up db -d
```

### ビルド
```bash
# フロントエンド
cd frontend
npm run build

# バックエンド
cd backend
go build -o bin/api cmd/api/main.go

# Docker
docker-compose build
```

## 📊 技術的特徴

### 🔧 バックエンド
- **クリーンアーキテクチャ**: レイヤー分離設計
- **依存性注入**: テスタブルな設計
- **エラーハンドリング**: 包括的エラー処理
- **ログ管理**: 構造化ログ出力

### 🎨 フロントエンド
- **サーバーサイドレンダリング**: Astroによる高速描画
- **型安全性**: TypeScript完全対応
- **状態管理**: nanostoresによる軽量管理
- **コンポーネント設計**: 再利用可能な設計

### 🤖 AI統合
- **自然言語処理**: Gemini AIによる高度な理解
- **感情分析**: 多角的な感情状態評価
- **傾向分析**: 時系列データからの洞察抽出
- **パーソナライゼーション**: ユーザー適応型応答

## 🔐 セキュリティ

- **認証**: JWT トークンベース認証
- **認可**: ロールベースアクセス制御
- **データ保護**: 機密情報の暗号化
- **CORS**: 適切なクロスオリジン設定
- **入力検証**: XSS・SQLインジェクション対策

## 📈 パフォーマンス

- **フロントエンド**: Vite最適化、コード分割
- **バックエンド**: Go高性能、並行処理
- **データベース**: インデックス最適化
- **キャッシング**: Service Worker、Redis対応準備済み

## 🌍 国際化対応

- **言語**: 日本語優先設計
- **タイムゾーン**: 複数タイムゾーン対応
- **文字エンコーディング**: UTF-8完全対応

## 📋 ライセンス

このプロジェクトは MIT ライセンスの下で公開されています。詳細は [LICENSE](LICENSE) ファイルをご確認ください。

## 🤝 コントリビューション

コントリビューションを歓迎します！

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📞 サポート

- **Issues**: [GitHub Issues](https://github.com/your-org/kasaneha/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/kasaneha/discussions)
- **Email**: support@kasaneha.com

---

<div align="center">

**🎉 Kasaneha AI日記アプリ - あなたの心の成長をサポートします 🎉**

Made with ❤️ by the Kasaneha Team

</div> 