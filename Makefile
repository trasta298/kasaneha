# Kasaneha Project Makefile

.PHONY: help dev build test clean batch-build batch-run batch-dry-run

# デフォルトターゲット
help:
	@echo "Kasaneha Project Commands:"
	@echo "  dev              - Start development environment"
	@echo "  build            - Build production containers"
	@echo "  test             - Run tests"
	@echo "  clean            - Clean up containers and volumes"
	@echo "  batch-build      - Build batch processor"
	@echo "  batch-run        - Run batch analysis"
	@echo "  batch-dry-run    - Run batch analysis in dry-run mode"
	@echo "  batch-logs       - Show batch processing logs"

# 開発環境の起動
dev:
	docker compose up --build

# 本番環境のビルド
build:
	docker compose -f compose.yaml build

# テストの実行
test:
	cd backend && go test ./...

# クリーンアップ
clean:
	docker compose down -v
	docker system prune -f

# バッチプロセッサーのビルド
batch-build:
	cd backend && go build -o bin/batch ./cmd/batch

# バッチ分析の実行（本番環境）
batch-run:
	docker compose exec batch-scheduler /app/scripts/daily-analysis.sh

# バッチ分析のドライラン（本番環境）
batch-dry-run:
	docker compose exec batch-scheduler /app/batch --dry-run --min-messages=2

# バッチ処理のローカル実行（開発環境）
batch-local:
	cd backend && \
	export $$(cat ../.env | xargs) && \
	go run ./cmd/batch --min-messages=2

# バッチ処理のローカルドライラン（開発環境）
batch-local-dry:
	cd backend && \
	export $$(cat ../.env | xargs) && \
	go run ./cmd/batch --dry-run --min-messages=2

# バッチログの確認
batch-logs:
	docker compose logs batch-scheduler

# バッチログのリアルタイム監視
batch-logs-follow:
	docker compose logs -f batch-scheduler

# データベースのマイグレーション
migrate:
	docker compose exec backend go run ./cmd/migrate

# データベースのリセット
db-reset:
	docker compose down postgres
	docker volume rm kasaneha_postgres_data
	docker compose up -d postgres
	sleep 10
	$(MAKE) migrate

# 全サービスの再起動
restart:
	docker compose restart

# バッチサービスのみ再起動
batch-restart:
	docker compose restart batch-scheduler

# 開発用: バッチ処理の即座実行
batch-now:
	@echo "Running batch analysis immediately..."
	docker compose exec batch-scheduler /app/batch --min-messages=1

# 環境変数のテンプレート作成
env-template:
	@echo "Creating .env template..."
	@echo "# Kasaneha Environment Variables" > .env.template
	@echo "GEMINI_API_KEY=your_gemini_api_key_here" >> .env.template
	@echo "PUBLIC_API_BASE_URL=http://localhost:8080/api/v1" >> .env.template
	@echo "MIN_MESSAGES=2" >> .env.template
	@echo "WEBHOOK_URL=" >> .env.template
	@echo ".env.template created. Copy to .env and fill in your values." 