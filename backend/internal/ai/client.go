package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/trasta298/kasaneha/backend/pkg/timeutil"
	"google.golang.org/genai"
)

// Client wraps the Gemini AI client
type Client struct {
	client *genai.Client
	model  string
}

// NewClient creates a new AI client
func NewClient(apiKey, model string) (*Client, error) {
	fmt.Println("DEBUG: NewClient called with apiKey:", apiKey, "and model:", model)
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is required")
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &Client{
		client: client,
		model:  model,
	}, nil
}

// ConversationRequest represents a request for conversation generation
type ConversationRequest struct {
	UserMessage         string    `json:"user_message"`
	ConversationHistory []Message `json:"conversation_history"`
	Date                string    `json:"date"`
	TimeOfDay           string    `json:"time_of_day"`
	UserName            string    `json:"user_name"`
}

// ConversationResponse represents a response from conversation generation
type ConversationResponse struct {
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Message represents a message in conversation history
type Message struct {
	Content string `json:"content"`
	Sender  string `json:"sender"`
}

// EmotionAnalysis represents the result of emotion analysis
type EmotionAnalysis struct {
	PrimaryEmotion string             `json:"primary_emotion"`
	Emotions       map[string]float64 `json:"emotions"`
	Confidence     float64            `json:"confidence"`
	Explanation    string             `json:"explanation"`
}

// TensionScoreAnalysis represents the result of tension score analysis
type TensionScoreAnalysis struct {
	TensionScore  int      `json:"tension_score"`
	RelativeScore int      `json:"relative_score"`
	Reasoning     string   `json:"reasoning"`
	KeyFactors    []string `json:"key_factors"`
}

// Helper function to convert float to *float32
func float32Ptr(f float64) *float32 {
	result := float32(f)
	return &result
}

// Helper function to convert int to *int32
func int32Ptr(i int) *int32 {
	result := int32(i)
	return &result
}

// GenerateResponse generates an AI response for a conversation
func (c *Client) GenerateResponse(ctx context.Context, req ConversationRequest) (*ConversationResponse, error) {
	systemPrompt := c.buildConversationSystemPrompt(req)

	// Build conversation history
	messages := []*genai.Content{}

	// Add system prompt
	messages = append(messages, &genai.Content{
		Parts: []*genai.Part{{Text: systemPrompt}},
		Role:  "user", // Gemini treats system messages as user messages
	})

	// Add conversation history
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

	// Add current user message
	messages = append(messages, &genai.Content{
		Parts: []*genai.Part{{Text: req.UserMessage}},
		Role:  "user",
	})

	// Generate response
	response, err := c.client.Models.GenerateContent(ctx, c.model, messages, &genai.GenerateContentConfig{
		Temperature:     float32Ptr(0.7),
		MaxOutputTokens: 500,
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: false,
			ThinkingBudget:  int32Ptr(0),
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response generated")
	}

	return &ConversationResponse{
		Content:   response.Candidates[0].Content.Parts[0].Text,
		Timestamp: timeutil.NowJST(),
	}, nil
}

// AnalyzeEmotion analyzes emotions from conversation log
func (c *Client) AnalyzeEmotion(ctx context.Context, conversationLog string) (*EmotionAnalysis, error) {
	prompt := c.buildEmotionAnalysisPrompt(conversationLog)

	messages := []*genai.Content{
		{
			Parts: []*genai.Part{{Text: prompt}},
			Role:  "user",
		},
	}

	response, err := c.client.Models.GenerateContent(ctx, c.model, messages, &genai.GenerateContentConfig{
		Temperature:      float32Ptr(0.3), // Lower temperature for more consistent analysis
		MaxOutputTokens:  2000,
		ResponseMIMEType: "application/json",
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: true,
			ThinkingBudget:  int32Ptr(1000),
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to analyze emotion: %w", err)
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no analysis generated")
	}

	var responseText string
	for _, part := range response.Candidates[0].Content.Parts {
		if !part.Thought {
			responseText = part.Text
			break
		}
	}

	if responseText == "" {
		return nil, fmt.Errorf("no non-thought response found")
	}

	var analysis EmotionAnalysis
	if err := json.Unmarshal([]byte(responseText), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse emotion analysis: %w", err)
	}

	return &analysis, nil
}

// CalculateTensionScore calculates tension score based on analysis and history
func (c *Client) CalculateTensionScore(ctx context.Context, todayAnalysis *EmotionAnalysis, historicalData string) (*TensionScoreAnalysis, error) {
	prompt := c.buildTensionScorePrompt(todayAnalysis, historicalData)

	messages := []*genai.Content{
		{
			Parts: []*genai.Part{{Text: prompt}},
			Role:  "user",
		},
	}

	response, err := c.client.Models.GenerateContent(ctx, c.model, messages, &genai.GenerateContentConfig{
		Temperature:      float32Ptr(0.3),
		MaxOutputTokens:  2000,
		ResponseMIMEType: "application/json",
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: true,
			ThinkingBudget:  int32Ptr(1000),
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate tension score: %w", err)
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no score analysis generated")
	}

	var analysis TensionScoreAnalysis
	var responseText string
	for _, part := range response.Candidates[0].Content.Parts {
		if !part.Thought {
			responseText = part.Text
			break
		}
	}
	if err := json.Unmarshal([]byte(responseText), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse tension score analysis: %w", err)
	}

	return &analysis, nil
}

// GenerateFirstMessage generates the initial message for a new chat session
func (c *Client) GenerateFirstMessage(ctx context.Context, userName, date, timeOfDay string) (*ConversationResponse, error) {
	prompt := c.buildFirstMessagePrompt(userName, date, timeOfDay)

	messages := []*genai.Content{
		{
			Parts: []*genai.Part{{Text: prompt}},
			Role:  "user",
		},
	}

	response, err := c.client.Models.GenerateContent(ctx, c.model, messages, &genai.GenerateContentConfig{
		Temperature:     float32Ptr(0.7),
		MaxOutputTokens: 1000,
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: false,
			ThinkingBudget:  int32Ptr(0),
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate first message: %w", err)
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no first message generated")
	}

	return &ConversationResponse{
		Content:   response.Candidates[0].Content.Parts[0].Text,
		Timestamp: timeutil.NowJST(),
	}, nil
}

func (c *Client) buildConversationSystemPrompt(req ConversationRequest) string {
	template := `あなたは親しみやすく、共感力の高い日記の相談相手です。
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

現在の日付: %s
時間帯: %s
ユーザー名: %s`

	return fmt.Sprintf(template, req.Date, req.TimeOfDay, req.UserName)
}

func (c *Client) buildFirstMessagePrompt(userName, date, timeOfDay string) string {
	template := `あなたはかさねという親しみやすいAIです。ユーザーの日記の相談相手として、今日の会話を始めてください。

ユーザー名: %s
現在の日付: %s  
時間帯: %s

以下のような感じで温かく挨拶し、今日の出来事について聞いてください：
- 親しみやすい挨拶
- 今日の調子を聞く
- 何か印象的な出来事があったか聞く
- 適度に絵文字を使用

150文字程度で簡潔にお願いします。`

	return fmt.Sprintf(template, userName, date, timeOfDay)
}

func (c *Client) buildEmotionAnalysisPrompt(conversationLog string) string {
	template := `以下の会話ログから、ユーザーの感情状態を分析してください。

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
%s`

	return fmt.Sprintf(template, conversationLog)
}

func (c *Client) buildTensionScorePrompt(todayAnalysis *EmotionAnalysis, historicalData string) string {
	template := `以下のユーザーの今日の感情分析結果と過去のデータを基に、
今日のテンションスコアを0-100で算出してください。

## スコア基準
- 0-20: 非常に低い（深刻な落ち込み）
- 21-40: 低い（やや沈んでいる）  
- 41-60: 普通（平常状態）
- 61-80: 高い（元気で前向き）
- 81-100: 非常に高い（とても良い状態）

## 今日の感情分析結果
主要感情: %s
感情スコア: %v
信頼度: %.2f

## 履歴データ（過去30日分）
%s

## 出力形式（JSON）
{
  "tension_score": 75,
  "relative_score": 10,
  "reasoning": "スコア算出の理由",
  "key_factors": ["影響した主要因子のリスト"]
}`

	emotionsJSON, _ := json.Marshal(todayAnalysis.Emotions)
	return fmt.Sprintf(template, todayAnalysis.PrimaryEmotion, string(emotionsJSON), todayAnalysis.Confidence, historicalData)
}
