package main

import (
	"embed"
	"log"
	"main/service"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	service.PRODUCTION = false // 生产环境模式默认开启，开发时可以改为 false 查看 DevTools。

	// 启动顺序一定要可控：配置 -> 音源 -> 窗口 -> MIDI 设备扫描。
	// 旧版本这里使用多个 goroutine 并发初始化，容易出现配置未加载、音源未准备好、窗口还没创建就发事件的问题。
	if err := service.LoadConfig("1.1.6"); err != nil {
		log.Println("加载配置失败，已使用默认配置:", err)
	}

	if err := service.InitSoundFont(service.ResolveSoundFontPath()); err != nil {
		// 音源加载失败时允许程序继续启动，用户后续可以在设置中心重新选择音源。
		log.Println("初始化音源失败，程序将以无内置发声模式启动:", err)
	}

	service.Run(assets)
}
