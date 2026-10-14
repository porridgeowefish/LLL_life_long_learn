package ocr

import "runtime"

type LocalCapability struct {
	Platform   string `json:"platform"`
	ProviderID string `json:"providerId"`
	Name       string `json:"name"`
	Available  bool   `json:"available"`
	SetupHint  string `json:"setupHint,omitempty"`
}

func localIdentity() (string, string) {
	switch runtime.GOOS {
	case "windows":
		return "windows-local-ocr", "windows-zh-Hans"
	case "darwin":
		return "macos-local-ocr", "vision-zh-Hans"
	case "linux":
		return "linux-local-ocr", "tesseract-chi_sim"
	default:
		return "system-local-ocr", ""
	}
}

func IsLocal(p PageProvider) bool { return p.Kind == "windows-local" || p.Kind == "system-local" }

func LocalPageCapability() LocalCapability {
	id, _ := localIdentity()
	cap := LocalCapability{Platform: runtime.GOOS, ProviderID: id, Available: LocalPageAvailable()}
	switch runtime.GOOS {
	case "windows":
		cap.Name = "Windows 系统 OCR"
		cap.SetupHint = "请在 Windows 语言设置中安装简体中文 OCR 语言包，然后重启应用。"
	case "darwin":
		cap.Name = "macOS Vision"
		cap.SetupHint = "免费 OCR 需要支持简体中文的系统 Vision 和安装包自带组件。请更新 macOS（推荐 13 或更新版本）或重新安装应用。"
	case "linux":
		cap.Name = "Tesseract"
		cap.SetupHint = "请安装 Tesseract 和简体中文 chi_sim 语言包。Ubuntu/Debian：sudo apt install tesseract-ocr tesseract-ocr-chi-sim；其它发行版请使用对应软件包。安装后刷新识别状态。"
	default:
		cap.Name = "本地 OCR"
		cap.SetupHint = "当前系统暂不支持本地 OCR，可配置云端文档识别连接。"
	}
	if cap.Available {
		cap.SetupHint = ""
	}
	return cap
}
