package llm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lisniuse/pixel/internal/config"
)

const (
	maxTokens = 256
	userText  = "这是主人现在的电脑屏幕，请根据屏幕内容发送一句关心的话。"
)

var httpClient = &http.Client{Timeout: 60 * time.Second}

// Ask sends imgBytes (PNG) to the configured LLM and returns the reply.
// prompt is the system prompt chosen for this request.
func Ask(cfg *config.LLMConfig, prompt string, imgBytes []byte) (string, error) {
	if cfg.APIKey == "" {
		return "", fmt.Errorf("api_key 未填写，请编辑配置文件")
	}
	switch cfg.Provider {
	case config.ProviderAnthropic:
		return askAnthropic(cfg, prompt, imgBytes)
	case config.ProviderOpenAI:
		return askOpenAI(cfg, prompt, imgBytes)
	default:
		return "", fmt.Errorf("未知 provider %q，请使用 \"anthropic\" 或 \"openai\"", cfg.Provider)
	}
}

// ── Anthropic ────────────────────────────────────────────────────────────────

type anthropicRequest struct {
	Model     string            `json:"model"`
	MaxTokens int               `json:"max_tokens"`
	System    []anthropicSysMsg `json:"system"`
	Messages  []anthropicMsg    `json:"messages"`
}

type anthropicSysMsg struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}

type cacheControl struct {
	Type string `json:"type"`
}

type anthropicMsg struct {
	Role    string          `json:"role"`
	Content []anthropicPart `json:"content"`
}

type anthropicPart struct {
	Type   string       `json:"type"`
	Text   string       `json:"text,omitempty"`
	Source *imgSource   `json:"source,omitempty"`
}

type imgSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func askAnthropic(cfg *config.LLMConfig, prompt string, imgBytes []byte) (string, error) {
	endpoint := strings.TrimRight(cfg.BaseURL, "/") + "/v1/messages"

	payload := anthropicRequest{
		Model:     cfg.Model,
		MaxTokens: maxTokens,
		System: []anthropicSysMsg{
			{
				Type:         "text",
				Text:         prompt,
				CacheControl: &cacheControl{Type: "ephemeral"}, // prompt caching
			},
		},
		Messages: []anthropicMsg{
			{
				Role: "user",
				Content: []anthropicPart{
					{
						Type: "image",
						Source: &imgSource{
							Type:      "base64",
							MediaType: "image/png",
							Data:      base64.StdEncoding.EncodeToString(imgBytes),
						},
					},
					{Type: "text", Text: userText},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("x-api-key", cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "prompt-caching-2024-07-31")
	req.Header.Set("content-type", "application/json")

	return doRequest(req, func(b []byte) (string, error) {
		var r anthropicResponse
		if err := json.Unmarshal(b, &r); err != nil {
			return "", err
		}
		if r.Error != nil {
			return "", fmt.Errorf("[%s] %s", r.Error.Type, r.Error.Message)
		}
		if len(r.Content) == 0 {
			return "", fmt.Errorf("empty content")
		}
		return r.Content[0].Text, nil
	})
}

// ── OpenAI-compatible ─────────────────────────────────────────────────────────

type openaiRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	Messages  []openaiMsg   `json:"messages"`
}

type openaiMsg struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []openaiPart
}

type openaiPart struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *openaiImgURL `json:"image_url,omitempty"`
}

type openaiImgURL struct {
	URL string `json:"url"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func askOpenAI(cfg *config.LLMConfig, prompt string, imgBytes []byte) (string, error) {
	endpoint := strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions"
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imgBytes)

	payload := openaiRequest{
		Model:     cfg.Model,
		MaxTokens: maxTokens,
		Messages: []openaiMsg{
			{Role: "system", Content: prompt},
			{
				Role: "user",
				Content: []openaiPart{
					{Type: "image_url", ImageURL: &openaiImgURL{URL: dataURL}},
					{Type: "text", Text: userText},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	return doRequest(req, func(b []byte) (string, error) {
		var r openaiResponse
		if err := json.Unmarshal(b, &r); err != nil {
			return "", err
		}
		if r.Error != nil {
			return "", fmt.Errorf("%s", r.Error.Message)
		}
		if len(r.Choices) == 0 {
			return "", fmt.Errorf("empty choices")
		}
		return r.Choices[0].Message.Content, nil
	})
}

// ── shared helper ─────────────────────────────────────────────────────────────

func doRequest(req *http.Request, parse func([]byte) (string, error)) (string, error) {
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	text, err := parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse response (HTTP %d): %w\nbody: %s", resp.StatusCode, err, raw)
	}
	return text, nil
}
