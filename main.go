package main

import (
	"embed"
	"log"
	"main/service"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed soundfont/UprightPianoKW.sf2
var defaultSoundFont []byte

func main() {
	//service.PRODUCTION = false // 生产环境模式默认开启，开发时可以改为 false 查看 DevTools。

	// 启动顺序一定要可控：配置 -> 音源 -> 窗口 -> MIDI 设备扫描。
	if err := service.LoadConfig(); err != nil {
		log.Println("加载配置失败，已使用默认配置:", err)
	}
	service.SetEmbeddedDefaultSoundFont(defaultSoundFont)

	if err := service.InitSoundFontFromConfig(); err != nil {
		// 音源加载失败时允许程序继续启动，用户后续可以在设置中心重新选择音源。
		log.Println("初始化音源失败，程序将以无内置发声模式启动:", err)
	}

	service.Run(assets)
}
