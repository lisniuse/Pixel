package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const appDirName = ".desktop-pixel"

// LLMProvider identifies the API wire format to use.
type LLMProvider string

const (
	ProviderAnthropic LLMProvider = "anthropic"
	ProviderOpenAI    LLMProvider = "openai"
)

// LLMConfig holds every LLM-related setting.
type LLMConfig struct {
	// Provider selects the API format: "anthropic" or "openai".
	Provider LLMProvider `json:"provider"`
	// BaseURL is the API root, e.g. "https://api.anthropic.com" or
	// "https://api.openai.com/v1".  Trailing slashes are ignored.
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

// TTSConfig holds Volcengine TTS settings.
// Docs: https://www.volcengine.com/docs/6561/79823
type TTSConfig struct {
	// BaseURL is the TTS endpoint.
	BaseURL string `json:"base_url"`
	// AppID is the application ID obtained from the Volcengine console.
	AppID string `json:"app_id"`
	// BearerToken is the access token used in the Authorization header.
	BearerToken string `json:"bearer_token"`
	// Cluster is the service cluster name, e.g. "volcano_tts".
	Cluster string `json:"cluster"`
	// VoiceType is the speaker code, e.g. "BV700_streaming".
	VoiceType string `json:"voice_type"`
	// Encoding is the audio format: "wav" | "mp3" | "pcm" | "ogg_opus".
	Encoding    string  `json:"encoding"`
	SpeedRatio  float64 `json:"speed_ratio"`
	VolumeRatio float64 `json:"volume_ratio"`
	PitchRatio  float64 `json:"pitch_ratio"`
}

// Config is the full application configuration.
type Config struct {
	// IntervalSeconds is how often (in seconds) to take a screenshot.
	IntervalSeconds int `json:"interval_seconds"`
	// NotifyEnabled controls whether a system dialog pops up with the reply.
	NotifyEnabled bool `json:"notify_enabled"`
	// TTSEnabled controls whether the reply is spoken aloud via TTS.
	TTSEnabled bool      `json:"tts_enabled"`
	LLM        LLMConfig `json:"llm"`
	TTS        TTSConfig `json:"tts"`
	// Prompts is a pool of system prompts. One is chosen at random each tick.
	Prompts []string `json:"prompts"`
}

func defaultConfig() *Config {
	return &Config{
		IntervalSeconds: 10,
		NotifyEnabled:   true,
		TTSEnabled:      false,
		LLM: LLMConfig{
			Provider: ProviderAnthropic,
			BaseURL:  "https://api.anthropic.com",
			APIKey:   "",
			Model:    "claude-opus-4-6",
		},
		TTS: TTSConfig{
			BaseURL:     "https://openspeech.bytedance.com/api/v1/tts",
			AppID:       "",
			BearerToken: "",
			Cluster:     "volcano_tts",
			VoiceType:   "BV700_streaming",
			Encoding:    "wav",
			SpeedRatio:  1.0,
			VolumeRatio: 1.0,
			PitchRatio:  1.0,
		},
		Prompts: []string{
			// 温柔体贴型
			`你是一个可爱的女仆机器人，名叫"小像素"。你正在守护你的主人使用电脑。
性格温柔、体贴、略带俏皮。根据主人屏幕内容，发送一句关心或鼓励的话。
要求：不超过60个字，只回复那一句话，可用"呢~""哦~""嘛~"等语气词。`,

			// 元气活泼型
			`你是元气满满的女仆机器人"小像素"！你超级关心你的主人~
根据主人现在的电脑屏幕，用活泼开朗的语气给主人打气或提个小建议。
要求：不超过60个字，只回复那一句话，可以加"！"和可爱的颜文字。`,

			// 俏皮毒舌型
			`你是略带毒舌但内心超温柔的女仆机器人"小像素"。
看看主人在干什么，用俏皮又不失关心的方式评论一下，或者给个小建议。
要求：不超过60个字，只回复那一句话，语气可以稍微调侃但不能过分。`,

			// 健康提醒型
			`你是专注主人健康的女仆机器人"小像素"。
观察主人的屏幕，重点关注他的用眼健康、坐姿、休息和饮水情况，给出一句温柔提醒。
要求：不超过60个字，只回复那一句话，语气关切自然。`,

			// 夸夸鼓励型
			`你是专门给主人打气的女仆机器人"小像素"，擅长发现主人做得好的地方并真诚夸奖。
根据主人屏幕内容，送上一句真诚的鼓励或夸奖。
要求：不超过60个字，只回复那一句话，让主人感受到被认可。`,
		},
	}
}

// Dir returns the platform-appropriate app directory (~/.desktop-pixel).
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, appDirName), nil
}

// Load reads ~/.desktop-pixel/config.json, creating a default one on first run.
// It also ensures ~/.desktop-pixel/logs/ exists.
func Load() (*Config, error) {
	d, err := Dir()
	if err != nil {
		return nil, err
	}

	// Ensure both the config dir and the logs sub-dir are present.
	if err := os.MkdirAll(filepath.Join(d, "logs"), 0o755); err != nil {
		return nil, fmt.Errorf("create app dirs: %w", err)
	}

	cfgPath := filepath.Join(d, "config.json")
	data, err := os.ReadFile(cfgPath)
	if os.IsNotExist(err) {
		def := defaultConfig()
		if werr := writeFile(cfgPath, def); werr != nil {
			return nil, fmt.Errorf("write default config: %w", werr)
		}
		return def, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Unmarshal on top of defaults so any missing keys keep their default values.
	cfg := defaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 10
	}
	return cfg, nil
}

// SetupLogging redirects the standard logger to a daily rotating log file
// under ~/.desktop-pixel/logs/. Returns the open file so the caller can
// defer Close().
func SetupLogging() (*os.File, error) {
	d, err := Dir()
	if err != nil {
		return nil, err
	}
	logsDir := filepath.Join(d, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, err
	}

	name := "pixel-" + time.Now().Format("2006-01-02") + ".log"
	f, err := os.OpenFile(filepath.Join(logsDir, name),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	return f, nil
}

func writeFile(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600) // owner-readable only
}
