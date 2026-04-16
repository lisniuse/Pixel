package monitor

import (
	"context"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/lisniuse/pixel/internal/audio"
	"github.com/lisniuse/pixel/internal/config"
	"github.com/lisniuse/pixel/internal/llm"
	"github.com/lisniuse/pixel/internal/notify"
	"github.com/lisniuse/pixel/internal/screenshot"
	"github.com/lisniuse/pixel/internal/tts"
)

// Monitor manages the screenshot-and-notify loop.
type Monitor struct {
	cfg    *config.Config
	cancel context.CancelFunc
	done   chan struct{}
	busy   atomic.Int32 // 1 while an LLM call is in-flight
}

func New(cfg *config.Config) *Monitor {
	return &Monitor{cfg: cfg}
}

func (m *Monitor) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.done = make(chan struct{})
	go m.loop(ctx)
	return nil
}

func (m *Monitor) Stop() {
	if m.cancel == nil {
		return
	}
	m.cancel()
	m.cancel = nil
	<-m.done
	m.done = nil
}

func (m *Monitor) loop(ctx context.Context) {
	defer close(m.done)

	d := time.Duration(m.cfg.IntervalSeconds) * time.Second
	log.Printf("monitor: interval=%s model=%s provider=%s", d, m.cfg.LLM.Model, m.cfg.LLM.Provider)

	// Fire immediately on start, then on every tick.
	m.process()

	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.process()
		}
	}
}

func (m *Monitor) process() {
	// Skip if the previous LLM call hasn't finished yet.
	if !m.busy.CompareAndSwap(0, 1) {
		log.Println("monitor: previous call still running, skipping tick")
		return
	}
	defer m.busy.Store(0)

	imgBytes, err := screenshot.Capture()
	if err != nil {
		log.Printf("monitor: screenshot: %v", err)
		return
	}

	prompt := pickPrompt(m.cfg.Prompts)
	log.Printf("monitor: using prompt #%d", promptIndex(m.cfg.Prompts, prompt))
	reply, err := llm.Ask(&m.cfg.LLM, prompt, imgBytes)
	if err != nil {
		log.Printf("monitor: llm: %v", err)
		return
	}

	log.Printf("monitor: maid says: %s", reply)

	if m.cfg.NotifyEnabled {
		notify.Show("来自小像素女仆的消息 ♡", reply)
	}

	if m.cfg.TTSEnabled {
		wavData, err := tts.Synthesize(&m.cfg.TTS, reply)
		if err != nil {
			log.Printf("monitor: tts: %v", err)
		} else {
			audio.Play(wavData)
		}
	}
}

// pickPrompt randomly selects one prompt from the pool.
// Falls back to a built-in prompt when the pool is empty.
func pickPrompt(prompts []string) string {
	if len(prompts) == 0 {
		return `你是可爱的女仆机器人"小像素"，根据主人的电脑屏幕发送一句不超过60字的关心话语。只回复那一句话。`
	}
	return prompts[rand.Intn(len(prompts))]
}

func promptIndex(prompts []string, p string) int {
	for i, v := range prompts {
		if v == p {
			return i
		}
	}
	return -1
}
