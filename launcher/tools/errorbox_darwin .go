//go:build darwin

package tools

import (
	"fmt"
	"os/exec"
)

func ThrowErrorBox(content string) {
	message := "1-请检查程序文件完整性\n2-请检查WebView2是否已安装\n" + content + "\n若问题仍未解决，请尝试重新安装或联系技术支持。"
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`display dialog "%s" with title "程序运行失败！" buttons {"OK"} with icon stop`, message))
	_ = cmd.Run() // 忽略错误，确保不中断主流程

	OpenWebViewPage()
}
