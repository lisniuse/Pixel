package tray

import (
	"log"
	"os"
	"path/filepath"

	"github.com/getlantern/systray"
	"github.com/lisniuse/pixel/internal/config"
	"github.com/lisniuse/pixel/internal/monitor"
	"github.com/lisniuse/pixel/internal/notify"
)

var (
	mon *monitor.Monitor
	cfg *config.Config
)

// Init must be called with the loaded config before systray.Run.
func Init(c *config.Config) {
	cfg = c
}

func OnReady() {
	systray.SetIcon(iconBytes())
	systray.SetTitle("Pixel")
	systray.SetTooltip("Pixel 桌面女仆 - 未监听")

	mToggle := systray.AddMenuItem("开始监听", "开启屏幕监听")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出 Pixel")

	mon = monitor.New(cfg)

	go func() {
		listening := false
		for {
			select {
			case <-mToggle.ClickedCh:
				if listening {
					mon.Stop()
					mToggle.SetTitle("开始监听")
					systray.SetTooltip("Pixel 桌面女仆 - 未监听")
					listening = false
					log.Println("tray: monitoring stopped")
				} else {
					// Warn if API key is not configured.
					if cfg.LLM.APIKey == "" {
						cfgDir, _ := config.Dir()
						cfgPath := filepath.Join(cfgDir, "config.json")
						notify.Show("Pixel - 需要配置", "请先填写 API Key：\n"+cfgPath)
						continue
					}
					if err := mon.Start(); err != nil {
						log.Printf("tray: start monitor: %v", err)
						notify.Show("Pixel 错误", "启动监听失败："+err.Error())
						continue
					}
					mToggle.SetTitle("停止监听")
					systray.SetTooltip("Pixel 桌面女仆 - 监听中")
					listening = true
					log.Println("tray: monitoring started")
				}
			case <-mQuit.ClickedCh:
				log.Println("tray: quit")
				os.Exit(0)
			}
		}
	}()
}

func OnExit() {
	if mon != nil {
		mon.Stop()
	}
}
