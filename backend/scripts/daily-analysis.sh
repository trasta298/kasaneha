#!/bin/sh

# 日次分析バッチスクリプト
# 一日の終わりにアクティブなセッションの分析を実行

set -eu
echo "run $(date)"

# タイムゾーンの設定
export TZ=Asia/Tokyo

# ログファイルの設定
LOG_DIR="/var/log/kasaneha"
LOG_FILE="$LOG_DIR/daily-analysis-$(date +%Y%m%d).log"

# ログディレクトリが存在しない場合は作成
mkdir -p "$LOG_DIR"

# ログローテーション（7日分保持）
find "$LOG_DIR" -name "daily-analysis-*.log" -mtime +7 -delete

echo "$(date '+%Y-%m-%d %H:%M:%S JST') - Starting daily analysis batch" >> "$LOG_FILE"

# バッチコマンドのパス
BATCH_CMD="/app/batch"

# 環境変数の確認
if [ -z "$DATABASE_URL" ] || [ -z "$GEMINI_API_KEY" ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S JST') - ERROR: Required environment variables are not set" >> "$LOG_FILE"
    exit 1
fi

# 最小メッセージ数（デフォルト: 2）
MIN_MESSAGES=${MIN_MESSAGES:-2}

echo "$(date '+%Y-%m-%d %H:%M:%S JST') - Running analysis for sessions with at least $MIN_MESSAGES messages" >> "$LOG_FILE"

# バッチ分析の実行
if "$BATCH_CMD" -min-messages="$MIN_MESSAGES" >> "$LOG_FILE" 2>&1; then
    echo "$(date '+%Y-%m-%d %H:%M:%S JST') - Daily analysis completed successfully" >> "$LOG_FILE"
    
    # 成功時の通知（オプション）
    if [ -n "$WEBHOOK_URL" ]; then
        curl -X POST "$WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "{\"text\": \"Daily analysis completed successfully at $(date '+%Y-%m-%d %H:%M:%S JST')\"}" \
            >> "$LOG_FILE" 2>&1 || true
    fi
else
    EXIT_CODE=$?
    echo "$(date '+%Y-%m-%d %H:%M:%S JST') - ERROR: Daily analysis failed with exit code $EXIT_CODE" >> "$LOG_FILE"
    
    # 失敗時の通知（オプション）
    if [ -n "$WEBHOOK_URL" ]; then
        curl -X POST "$WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "{\"text\": \"⚠️ Daily analysis failed at $(date '+%Y-%m-%d %H:%M:%S JST') with exit code $EXIT_CODE\"}" \
            >> "$LOG_FILE" 2>&1 || true
    fi
    
    exit $EXIT_CODE
fi

echo "$(date '+%Y-%m-%d %H:%M:%S JST') - Daily analysis batch finished" >> "$LOG_FILE" 