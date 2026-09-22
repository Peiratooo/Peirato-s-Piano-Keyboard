//go:build darwin

package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RunApp(app string) {
	// 获取可执行文件绝对路径
	exePath := getExecutablePath(app)
	// 验证文件存在
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		ThrowErrorBox(fmt.Sprintf("可执行文件不存在: %s", exePath))
		return
	}
	// 设置执行权限
	if err := os.Chmod(exePath, 0755); err != nil {
		ThrowErrorBox(fmt.Sprintf("权限设置失败: %v", err))
		return
	}
	cmd := exec.Command(exePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// 关键设置：让b4core成为新的进程组领导
	//cmd.SysProcAttr = &syscall.SysProcAttr{
	//	Setpgid: true,
	//}
	// 设置工作目录为应用包目录
	if isAppBundle() {
		cmd.Dir = filepath.Dir(filepath.Dir(filepath.Dir(exePath)))
	}
	if err := cmd.Start(); err != nil {
		ThrowErrorBox(fmt.Sprintf("启动失败: %v", err))
		return
	}
	_ = cmd.Process.Release()
	// 立即退出启动器（此时b4core会继续运行）
	//fmt.Println("启动器: 已启动b4core，即将退出")
	//os.Exit(0)
}

// 获取当前工作目录
func getCurrentDir() string {
	dir, _ := os.Getwd()
	return dir
}

// 检查是否在.app包内运行
func isAppBundle() bool {
	exe, _ := os.Executable()
	return strings.Contains(exe, ".app/Contents/MacOS")
}

// 获取可执行文件路径
func getExecutablePath(app string) string {
	// 1. 尝试从当前目录查找（开发环境）
	devPath := filepath.Join(getCurrentDir(), app)
	if _, err := os.Stat(devPath); err == nil {
		return devPath
	}
	// 2. 处理.app包情况
	exe, _ := os.Executable()
	if strings.Contains(exe, ".app/Contents/MacOS") {
		return filepath.Join(
			filepath.Dir(filepath.Dir(filepath.Dir(exe))),
			"Contents",
			"MacOS",
			app,
		)
	}
	return filepath.Join(getCurrentDir(), app)
}
