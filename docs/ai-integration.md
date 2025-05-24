# AI統合設計

## Gemini AI統合概要

Google Gemini APIを使用して、自然な会話、感情分析、テンションスコア算出を実現します。

## go-genai SDK統合

### 1. クライアント設定

```go
package ai

import (
    "context"
    "log"
    "google.golang.org/genai"
)

type GeminiClient struct {
    client *genai.Client
    model  string
}

func NewGeminiClient(apiKey string) (*GeminiClient, error) {
    client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
        APIKey:  apiKey,
        Backend: genai.BackendGeminiAPI,
    })
    if err != nil {
        return nil, err
    }

    return &GeminiClient{
        client: client,
        model:  "gemini-2.0-flash", // 最新の高速モデル使用
    }, nil
}
```

### 2. プロンプト設計

#### 会話生成プロンプト
```go
const conversationSystemPrompt = `
あなたは親しみやすく、共感力の高い日記の相談相手です。
ユーザーが今日起きた出来事について自然に話せるよう、優しく聞き出してください。

## 人格設定
- 名前: かさね（kasane）
- 年齢: 25歳くらいの印象
- 性格: 温かく、聞き上手で、ユーザーの気持ちに寄り添う
- 話し方: 丁寧語だが親しみやすく、適度に絵文字を使用

## 会話のガイドライン
1. ユーザーの話に共感を示す
2. 具体的な出来事や感情を引き出す質問をする
3. 批判的にならず、受容的な態度を保つ
4. 必要に応じて励ましや助言を与える
5. 会話が途切れないよう、適切な質問や相づちを入れる

## 会話の流れ例
1. 挨拶と今日の調子を聞く
2. 印象的だった出来事を聞く
3. その時の感情や考えを深掘りする
4. 他に話したいことがないか確認する
5. 今日の振り返りで締めくくる

現在の日付: {{.Date}}
時間帯: {{.TimeOfDay}}
ユーザー名: {{.UserName}}
`

const firstMessageTemplate = `
こんにちは{{.UserName}}さん！😊

今日も一日お疲れさまでした。
{{.TimeOfDay}}はいかがお過ごしでしたか？

今日印象に残った出来事や、気になったことがあれば
ぜひ聞かせてくださいね✨
`
```

#### 感情分析プロンプト
```go
const emotionAnalysisPrompt = `
以下の会話ログから、ユーザーの感情状態を分析してください。

## 分析項目
1. 基本感情（happiness, sadness, anger, fear, surprise, disgust）のスコア（0-1）
2. 主要感情の特定
3. 信頼度スコア（0-1）
4. 感情の詳細説明

## 出力形式（JSON）
{
  "primary_emotion": "感情名",
  "emotions": {
    "happiness": 0.8,
    "sadness": 0.1,
    "anger": 0.05,
    "fear": 0.05,
    "surprise": 0.0,
    "disgust": 0.0
  },
  "confidence": 0.85,
  "explanation": "感情分析の根拠説明"
}

## 会話ログ
{{.ConversationLog}}
`
```

#### テンションスコア算出プロンプト
```go
const tensionScorePrompt = `
以下のユーザーの過去のチャット履歴とその感情分析結果を基に、
今日のテンションスコアを0-100で算出してください。

## スコア基準
- 0-20: 非常に低い（深刻な落ち込み）
- 21-40: 低い（やや沈んでいる）  
- 41-60: 普通（平常状態）
- 61-80: 高い（元気で前向き）
- 81-100: 非常に高い（とても良い状態）

## 考慮要素
1. 感情の種類と強度
2. 出来事への反応
3. 将来への言及
4. 人間関係の状況
5. 仕事・活動への取り組み

## 履歴データ（過去30日分）
{{.HistoricalData}}

## 今日の分析結果
{{.TodayAnalysis}}

## 出力形式（JSON）
{
  "tension_score": 75,
  "relative_score": 10,
  "reasoning": "スコア算出の理由",
  "key_factors": ["影響した主要因子のリスト"]
}
`
```

### 3. AI処理フロー

#### 会話生成処理
```go
func (g *GeminiClient) GenerateResponse(ctx context.Context, req ConversationRequest) (*ConversationResponse, error) {
    // システムプロンプトの準備
    systemPrompt := templates.Execute(conversationSystemPrompt, map[string]interface{}{
        "Date":      req.Date,
        "TimeOfDay": req.TimeOfDay,
        "UserName":  req.UserName,
    })

    // 会話履歴の構築
    messages := []*genai.Content{
        {
            Parts: []*genai.Part{{Text: systemPrompt}},
            Role:  "system",
        },
    }

    // 過去のメッセージを追加
    for _, msg := range req.ConversationHistory {
        role := "user"
        if msg.Sender == "ai" {
            role = "model"
        }
        messages = append(messages, &genai.Content{
            Parts: []*genai.Part{{Text: msg.Content}},
            Role:  role,
        })
    }

    // 現在のユーザーメッセージを追加
    messages = append(messages, &genai.Content{
        Parts: []*genai.Part{{Text: req.UserMessage}},
        Role:  "user",
    })

    // Gemini API呼び出し
    response, err := g.client.Models.GenerateContent(ctx, g.model, messages, &genai.GenerateContentConfig{
        Temperature:     0.7,
        MaxOutputTokens: 500,
        TopK:           40,
        TopP:           0.9,
    })

    if err != nil {
        return nil, fmt.Errorf("failed to generate response: %w", err)
    }

    return &ConversationResponse{
        Content:   response.Candidates[0].Content.Parts[0].Text,
        Timestamp: time.Now(),
    }, nil
}
```

#### 感情分析処理
```go
func (g *GeminiClient) AnalyzeEmotion(ctx context.Context, conversationLog string) (*EmotionAnalysis, error) {
    prompt := strings.Replace(emotionAnalysisPrompt, "{{.ConversationLog}}", conversationLog, 1)

    messages := []*genai.Content{
        {
            Parts: []*genai.Part{{Text: prompt}},
            Role:  "user",
        },
    }

    response, err := g.client.Models.GenerateContent(ctx, g.model, messages, &genai.GenerateContentConfig{
        Temperature:     0.3, // 分析は一貫性重視
        MaxOutputTokens: 300,
        ResponseMIMEType: "application/json",
    })

    if err != nil {
        return nil, fmt.Errorf("failed to analyze emotion: %w", err)
    }

    var analysis EmotionAnalysis
    if err := json.Unmarshal([]byte(response.Candidates[0].Content.Parts[0].Text), &analysis); err != nil {
        return nil, fmt.Errorf("failed to parse emotion analysis: %w", err)
    }

    return &analysis, nil
}
```

## データ構造定義

### リクエスト・レスポンス型
```go
type ConversationRequest struct {
    UserMessage         string    `json:"user_message"`
    ConversationHistory []Message `json:"conversation_history"`
    Date                string    `json:"date"`
    TimeOfDay          string    `json:"time_of_day"`
    UserName           string    `json:"user_name"`
}

type ConversationResponse struct {
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
}

type EmotionAnalysis struct {
    PrimaryEmotion string             `json:"primary_emotion"`
    Emotions       map[string]float64 `json:"emotions"`
    Confidence     float64            `json:"confidence"`
    Explanation    string             `json:"explanation"`
}

type TensionScoreAnalysis struct {
    TensionScore   int      `json:"tension_score"`
    RelativeScore  int      `json:"relative_score"`
    Reasoning      string   `json:"reasoning"`
    KeyFactors     []string `json:"key_factors"`
}
```

## エラーハンドリング

### API制限対応
```go
type RateLimiter struct {
    requests chan struct{}
    ticker   *time.Ticker
}

func NewRateLimiter(rps int) *RateLimiter {
    rl := &RateLimiter{
        requests: make(chan struct{}, rps),
        ticker:   time.NewTicker(time.Second / time.Duration(rps)),
    }
    
    go func() {
        for range rl.ticker.C {
            select {
            case rl.requests <- struct{}{}:
            default:
            }
        }
    }()
    
    return rl
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
    select {
    case <-rl.requests:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### リトライ機能
```go
func (g *GeminiClient) callWithRetry(ctx context.Context, fn func() error) error {
    backoff := []time.Duration{
        1 * time.Second,
        2 * time.Second,
        5 * time.Second,
        10 * time.Second,
    }
    
    for i, delay := range backoff {
        if err := fn(); err != nil {
            if i == len(backoff)-1 {
                return err
            }
            
            select {
            case <-time.After(delay):
                continue
            case <-ctx.Done():
                return ctx.Err()
            }
        }
        return nil
    }
    return nil
}
```

## パフォーマンス最適化

### 1. バッチ処理
複数の分析を同時に実行する場合の最適化

```go
func (g *GeminiClient) BatchAnalyze(ctx context.Context, sessions []SessionData) ([]AnalysisResult, error) {
    semaphore := make(chan struct{}, 5) // 並行数制限
    results := make([]AnalysisResult, len(sessions))
    var wg sync.WaitGroup
    
    for i, session := range sessions {
        wg.Add(1)
        go func(index int, data SessionData) {
            defer wg.Done()
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            result, err := g.AnalyzeSession(ctx, data)
            if err != nil {
                log.Printf("Analysis failed for session %s: %v", data.ID, err)
                return
            }
            results[index] = result
        }(i, session)
    }
    
    wg.Wait()
    return results, nil
}
```

### 2. キャッシング戦略
```go
type AnalysisCache struct {
    cache map[string]CacheEntry
    mutex sync.RWMutex
    ttl   time.Duration
}

type CacheEntry struct {
    Data      interface{}
    ExpiresAt time.Time
}

func (c *AnalysisCache) Get(key string) (interface{}, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    
    entry, exists := c.cache[key]
    if !exists || time.Now().After(entry.ExpiresAt) {
        return nil, false
    }
    
    return entry.Data, true
}
```

## 監視・ログ

### APIメトリクス
```go
type AIMetrics struct {
    RequestCount    prometheus.Counter
    RequestDuration prometheus.Histogram
    ErrorCount      prometheus.Counter
    TokenUsage      prometheus.Counter
}

func NewAIMetrics() *AIMetrics {
    return &AIMetrics{
        RequestCount: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "ai_requests_total",
            Help: "Total number of AI API requests",
        }),
        // ... other metrics
    }
}
```

### ログ記録
```go
func (g *GeminiClient) logAPICall(ctx context.Context, operation string, duration time.Duration, err error) {
    log := logrus.WithFields(logrus.Fields{
        "operation":     operation,
        "duration_ms":   duration.Milliseconds(),
        "model":         g.model,
        "trace_id":      getTraceID(ctx),
    })
    
    if err != nil {
        log.WithError(err).Error("AI API call failed")
    } else {
        log.Info("AI API call completed")
    }
}
``` 