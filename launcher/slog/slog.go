package slog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logFilePath string
	logMutex    sync.Mutex
)

func init() {
	logFilePath = filepath.Join(GetAppRoot(), "app.log")
}

func GetAppRoot() string {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("获取程序路径失败: %v\n", err)
		return "." // 回退到当前目录
	}
	return filepath.Dir(exePath)
}

func P(a ...any) {
	message := fmt.Sprint(a...)
	now := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] %s\n", now, message)

	// 打印到控制台
	fmt.Print(logLine)

	// 使用互斥锁保证多 goroutine 写入安全
	logMutex.Lock()
	defer logMutex.Unlock()

	// 打开文件，追加写入，如果不存在则创建
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("打开日志文件失败: %v\n", err)
		return
	}
	defer f.Close()

	_, err = f.WriteString(logLine)
	if err != nil {
		fmt.Printf("写入日志文件失败: %v\n", err)
	}
}
