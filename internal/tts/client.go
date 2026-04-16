// Package tts calls the Volcengine HTTP TTS API and returns raw audio bytes.
// API docs: https://www.volcengine.com/docs/6561/79823
package tts

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lisniuse/pixel/internal/config"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

type ttsResponse struct {
	ReqID     string `json:"reqid"`
	Code      int    `json:"code"`
	Message   string `json:"Message"`
	Operation string `json:"operation"`
	Sequence  int    `json:"sequence"`
	Data      string `json:"data"` // base64-encoded audio
}

// Synthesize converts text to speech and returns raw audio bytes
// in the format specified by cfg.Encoding (default: wav).
func Synthesize(cfg *config.TTSConfig, text string) ([]byte, error) {
	if cfg.AppID == "" || cfg.BearerToken == "" {
		return nil, fmt.Errorf("tts: app_id 或 bearer_token 未填写，请编辑配置文件")
	}

	payload := map[string]map[string]interface{}{
		"app": {
			"appid":   cfg.AppID,
			"token":   "access_token", // fixed dummy value per API spec
			"cluster": cfg.Cluster,
		},
		"user": {
			"uid": "pixel",
		},
		"audio": {
			"voice_type":   cfg.VoiceType,
			"encoding":     cfg.Encoding,
			"speed_ratio":  cfg.SpeedRatio,
			"volume_ratio": cfg.VolumeRatio,
			"pitch_ratio":  cfg.PitchRatio,
		},
		"request": {
			"reqid":     newReqID(),
			"text":      text,
			"text_type": "plain",
			"operation": "query",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("tts: marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, cfg.BaseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("tts: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer;%s", cfg.BearerToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tts: do request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tts: read body: %w", err)
	}

	var result ttsResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("tts: unmarshal (HTTP %d): %w\nbody: %s", resp.StatusCode, err, raw)
	}
	if result.Code != 3000 {
		return nil, fmt.Errorf("tts: API error code=%d message=%s", result.Code, result.Message)
	}

	audio, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		return nil, fmt.Errorf("tts: decode audio: %w", err)
	}
	return audio, nil
}

// newReqID returns a random UUID v4 string without external dependencies.
func newReqID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
