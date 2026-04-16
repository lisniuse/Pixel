# Pixel 桌面女仆

> 一个运行在系统托盘里的 AI 桌面智能体。每隔一段时间截取屏幕，发给大模型，然后以女仆的口吻弹窗或语音播报一句关心的话。

![platform](https://img.shields.io/badge/platform-Windows-blue)
![language](https://img.shields.io/badge/language-Go-00ADD8)

---

## 功能

- **系统托盘**：粉色爱心图标，右键菜单开启/停止监听，强制退出
- **定时截图**：每隔 N 秒截取主屏幕（可配置），自动缩放到 ≤ 1568px
- **AI 分析**：把截图发给大模型，从 prompt 池中随机抽一条系统提示词
- **弹窗通知**：用原生 Windows 对话框弹出女仆的话（可关闭）
- **语音播报**：接入火山引擎 TTS，用 `winmm.dll` 直接播放 WAV（可关闭）
- **配置文件**：`~/.desktop-pixel/config.json`，首次运行自动生成默认配置
- **日志**：每天一个文件，存放在 `~/.desktop-pixel/logs/`

---

## 快速开始

### 1. 编译

```powershell
# 安装 rsrc 工具（仅第一次需要）
go install github.com/akavel/rsrc@latest

# 一键构建（生成图标 + EXE 资源 + 编译）
.\build.ps1
```

产物在 `bin/pixel.exe`。

### 2. 配置

首次运行 `pixel.exe` 后，自动在 `C:\Users\{你的用户名}\.desktop-pixel\` 生成配置文件。

用任意编辑器打开 `config.json`，填写以下关键字段：

```json
{
  "interval_seconds": 10,
  "notify_enabled": true,
  "tts_enabled": false,
  "llm": {
    "provider": "anthropic",
    "base_url": "https://api.anthropic.com",
    "api_key": "你的 API Key",
    "model": "claude-opus-4-6"
  },
  "tts": {
    "base_url": "https://openspeech.bytedance.com/api/v1/tts",
    "app_id": "火山引擎 APP ID",
    "bearer_token": "火山引擎 Access Token",
    "cluster": "volcano_tts",
    "voice_type": "BV700_V2_streaming",
    "encoding": "wav",
    "speed_ratio": 1.0,
    "volume_ratio": 1.0,
    "pitch_ratio": 1.0
  },
  "prompts": [
    "你是一个可爱的女仆机器人，名叫\"小像素\"。根据主人屏幕内容发送一句关心的话，不超过60字，只回复那一句。"
  ]
}
```

### 3. 运行

双击 `bin/pixel.exe`，托盘出现粉色爱心图标，右键 → **开始监听** 即可。

---

## 配置说明

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `interval_seconds` | 截图间隔（秒） | `10` |
| `notify_enabled` | 是否弹出系统对话框 | `true` |
| `tts_enabled` | 是否语音播报 | `false` |
| `llm.provider` | API 格式：`anthropic` 或 `openai` | `anthropic` |
| `llm.base_url` | API 根地址 | `https://api.anthropic.com` |
| `llm.api_key` | API Key | 空（必填） |
| `llm.model` | 模型名称 | `claude-opus-4-6` |
| `tts.app_id` | 火山引擎应用 ID | 空 |
| `tts.bearer_token` | 火山引擎 Access Token | 空 |
| `tts.cluster` | 集群名称 | `volcano_tts` |
| `tts.voice_type` | 音色代号 | `BV700_V2_streaming` |
| `prompts` | 系统提示词池，每次随机抽一条 | 内置5条 |

### 使用 OpenAI 兼容接口

```json
"llm": {
  "provider": "openai",
  "base_url": "https://api.openai.com/v1",
  "api_key": "sk-...",
  "model": "gpt-4o"
}
```

### 推荐可爱女声音色

| 音色名 | voice_type |
|--------|-----------|
| 灿灿 2.0（推荐，支持撒娇/娇媚/傲娇） | `BV700_V2_streaming` |
| 小萝莉（萌系可爱） | `BV064_streaming` |
| 甜美小源 | `BV405_streaming` |
| 活泼女声 | `BV005_streaming` |

---

## 目录结构

```
Pixel/
├── assets/
│   └── icon.svg                  # 托盘图标 SVG 源文件
├── bin/
│   └── pixel.exe                 # 编译产物
├── cmd/pixel/
│   └── main.go                   # 程序入口
├── internal/
│   ├── audio/
│   │   └── player_windows.go     # winmm.dll WAV 播放
│   ├── config/
│   │   └── config.go             # 配置加载 / 日志初始化
│   ├── llm/
│   │   └── client.go             # Anthropic / OpenAI 接口
│   ├── monitor/
│   │   └── monitor.go            # 定时截图循环
│   ├── notify/
│   │   └── notify_windows.go     # Windows MessageBox 弹窗
│   ├── screenshot/
│   │   └── screenshot.go         # GDI 截图 + 缩放
│   ├── tray/
│   │   ├── assets/icon.ico       # 生成的多分辨率 ICO
│   │   ├── icon.go               # embed ICO
│   │   └── tray.go               # 托盘菜单逻辑
│   └── tts/
│       └── client.go             # 火山引擎 TTS HTTP 接口
├── tools/genicon/                 # 独立工具：SVG → PNG → ICO
│   ├── go.mod
│   └── main.go
├── build.ps1                      # 一键构建脚本
└── go.mod
```

---

## 开发

```powershell
# 只编译（跳过图标生成）
go build -ldflags="-H windowsgui" -o bin/pixel.exe ./cmd/pixel/

# 重新生成托盘图标（修改 assets/icon.svg 后执行）
Push-Location tools/genicon
go run . -svg ../../assets/icon.svg -out ../../internal/tray/assets/icon.ico
Pop-Location

# 重新生成 EXE 图标资源
rsrc -arch amd64 -ico internal/tray/assets/icon.ico -o cmd/pixel/resource.syso
```

---

## 依赖

| 包 | 用途 |
|----|------|
| `github.com/getlantern/systray` | 系统托盘 |
| `github.com/kbinani/screenshot` | 屏幕截图 |
| `github.com/srwiley/oksvg` + `rasterx` | SVG 渲染（图标生成工具） |
| `golang.org/x/sys` | Windows API |

TTS 和 LLM 均通过标准 HTTP 调用，无额外 SDK 依赖。

---

## License

MIT
