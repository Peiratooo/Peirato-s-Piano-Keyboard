
# 🎹 Peirato's Piano

✨ 一个基于 **Wails3 + Vue3** 开发的跨平台轻量桌面钢琴程序。  
支持 **MIDI 键盘** 与 **电脑打字键盘** 演奏，实时显示 **和弦名称**，并可自定义主题颜色。  

## 📖 功能特性

- 🎼 **实时钢琴键盘**：打开软件即可开始演奏  
- 🎹 **支持 MIDI 键盘**：自动识别外接 MIDI 设备  
- ⌨️ **键盘演奏模式**：使用电脑打字键盘也能弹琴  
- 💡 **按键亮起效果**：演奏时对应的钢琴键会高亮  
- 🎶 **和弦识别**：自动识别当前演奏的和弦并显示  
- 🎨 **主题自定义**：支持修改主题颜色，打造专属风格  

---

## 📸 截图预览

![](./screenshot.png)

## 📦 安装与运行

### 下载 Release
前往 [Releases](https://github.com/Peiratooo/Peirato-s-Piano/releases/tag/1.1.6) 下载最新版本，解压后运行即可。

### 从源码构建
```bash
# 克隆仓库
git clone https://github.com/Peiratooo/Peirato-s-Piano

# 同步依赖
go mod tidy

# 启动开发模式
wails3 dev

# 构建发布版本
wails3 task release
```

*注意：构建前需安装 [Wails 3](https://wails.io/) 和 Go 环境。*

## 🧭 架构与排障

项目的启动流程、核心函数职责、MIDI/音源/练习流程和常见问题排查见 [架构文档](./docs/ARCHITECTURE.md)。

---

## 📜 许可证

本项目基于 MIT License 开源。
你可以自由使用、修改、分发，但请保留版权信息。

------

## 💡 致谢

- [Wails](https://wails.io/) — 跨平台桌面应用框架。
- [Vue.js](https://vuejs.org/) — 前端框架。

> 🎶 如果觉得本项目对你有帮助，欢迎点个 ⭐ Star 支持一下！
