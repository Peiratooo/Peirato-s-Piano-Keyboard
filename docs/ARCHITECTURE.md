# Peirato's Piano 1.3.0 架构

应用使用 Wails v3、Go 和 Vue 3。Go 后端负责配置、音频、MIDI、更新与桌面窗口；前端使用 Pinia 保存各窗口的状态，通过 Wails bindings 调用 `Keyboard` 服务。

## 启动与窗口

`main.go` 按顺序读取配置、注册嵌入式 Upright Piano KW 音源、初始化音源，再调用 `service.Run()`。音源初始化失败时记录错误，窗口仍可打开。

`service/app.go` 创建三个窗口：

| 窗口 | 路由 | 前端 |
| --- | --- | --- |
| 主钢琴窗口 | `/` | `views/MainWindow.vue` |
| 设置中心 | `/#/control` | `views/ControlCenterWindow.vue` |
| MIDI 中心 | `/#/midi` | `views/MidiWindow.vue` |

设置中心和 MIDI 中心各自保持单例，关闭时清除窗口引用。关闭 MIDI 中心会重置播放器；应用关闭时清理下载缓存并释放 MIDI 设备。

## 配置与版本

`service/config.go` 中的 `AppVersion` 固定当前运行版本为 `1.3.0`。`LoadConfig()` 从用户配置目录读取 `config.json`；首次启动创建默认配置，JSON 损坏时尝试 `.bak` 备份。配置使用当前字段，不包含版本分支或专门的旧配置合并器。

`SaveConfig()` 复制并校验数据，通过 `Revision` 拒绝过期写入，备份有效 JSON，再以同目录临时文件和平台原子替换写入。版本号始终使用 `AppVersion`，不能由前端配置覆盖。读取配置使用独立的 map 和切片快照。

`frontend/src/App.vue` 启动时调用 `SendConfig()`。设置修改串行保存，只提交待保存字段；失败时保留编辑。后端保存成功后向所有窗口发出 `configChanged`。原子保存、备份恢复、输入校验和音源失败处理属于正常运行保障。

## 演奏与 MIDI 设备

`frontend/src/services/computerKeyboard.js` 按当前 `activeKeyMapping` 处理电脑按键。按下时记住物理键对应的 MIDI 音，松键、失焦和卸载时释放；同音映射按计数管理。鼠标演奏由 `ClassicKeyboard.vue` 处理。

`service/piano.go` 扫描和监听 MIDI 设备，按设备、通道、音高管理活动音符和踏板状态。输入通过统一路由发送到软件音源或外部 MIDI 输出。电脑键盘使用 `KeyboardPlay()` / `KeyboardStop()`；`AllNotesOff()` 释放所有输出并清除可视状态。

`service/keymap.go` 管理多个按键方案，内置默认映射；导入和导出使用 `.pp-keymap.json`。前端不再加载静态 `case-*.json`。

## 音源

`service/soundfont.go` 管理嵌入式 SF2、自定义音源目录、MeltySynth 合成器和 beep 扬声器。当前音源由 `ActiveSoundFontID` 选择，音源记录保存在 `SoundFonts` 中。

切换音源与保存配置串行执行；失败时保留前一个可用音源。启动时当前自定义音源不可用则使用内置音源。音量控制软件合成器 master gain。

## MIDI 文件与跟弹

`service/midi.go` 保存轻量 MIDI 文件目录，通过 `gomidi/smf` 解析时间、轨道和事件，并推断左右手。运行时缓存按 LRU 淘汰，最多保留 16 首、总计 50 万事件；超大单首不缓存。

`service/player.go` 使用 `MidiPlaybackOptions` 管理播放模式、练习手、区间、速度、循环和分组窗口。播放器负责事件调度、通道与静音、暂停恢复、跳转和清音；跟弹步骤及命中判断均在 Go 后端执行。

`MidiWindow.vue` 通过 `LoadMidiByID()`、`StartMidiPlayback()`、`SwitchMidiPlayback()`、`PauseMidiPlayback()`、`ResumeMidiPlayback()`、`SeekMidiPlayback()` 等绑定操作播放器。

## 窗口同步

各 WebView 的 Pinia store 独立。所有共享状态由后端 Wails 事件同步，前端直接读取当前事件格式的 `event.data`：

- 配置与设备：`configChanged`、`devices`、`softwareSustain`。
- 用户演奏：`down`、`up`、`pressedDown`、`pressedUp`、`pedal`。
- MIDI 中心：`midiWindowState`、`midiPlayerState`、`midiPlaybackKey`、`midiFollowHint`、`midiVisualClear`。
- 全局清理和更新：`allNotesOff`、`updateState`。

用户真实按键、延音、播放高亮与左右手提示分别保存，渲染时聚合。事件、键盘和 resize 监听在前端卸载时取消。

## 软件更新

`service/release_update.go` 仅访问 `/release/get_info` 和 `/release/download`，使用 AppID `com.peirato.piano` 和运行平台区分发行版本。更新检查直接比较远端版本与 `AppVersion`。

下载按 release ID 固定版本，只计算清单所属文件的本地校验值。差分包校验 SHA-256 后进入 `ready` 状态；用户在关于页确认安装后，再次校验并调用随软件安装的 updater，随后退出主程序。下载、校验与安装状态通过 `updateState` 同步所有窗口。

`service/update.go` 管理共享更新状态和安装。Windows updater 位于主程序旁，macOS 位于 `Contents/MacOS/updater`。缺失时提示使用完整安装包修复。没有旧 product 接口回退、兼容启动器或过渡卸载流程。

## 构建与打包

正式安装包统一由 Wails3 Manager 管理：

- `build/config.yml`：应用标识、名称、版本和开发配置。
- `Taskfile.yml` 的 `release`：编译主程序，并通过 `common:stage:runtime` 准备预编译 updater 和 Windows DLL。
- `builder/packaging.json`：当前 Windows Manager 配置，使用相对路径和默认启动入口。
- `builder/windows/inno.iss`：Manager 生成的临时安装脚本，不作为手工维护入口。
- `build/darwin/Taskfile.yml` 的 `create:app:bundle`：供 Manager 组装 macOS 应用包；在签名前放入通用 updater。
- `builder/release/`：Manager 的发行产物目录。

macOS 在原生系统中通过 Manager 打包，附加资源列表使用 macOS 资源，移除 Windows DLL 和 updater.exe；应用包任务已包含 updater。

`assets/updater/` 保留预编译更新器。只有升级固定的 updater 依赖时才使用 `go run ./cmd/updater-builder` 重新生成；`cmd/release-inspect` 可检查 Manager 生成的安装包清单。这些工具不负责生成安装器。
