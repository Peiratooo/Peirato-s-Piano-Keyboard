# Peirato's Piano 架构文档

本文档用于把项目从“黑箱”拆开：先说明整体结构，再按启动、演奏、MIDI、音源、练习和窗口通信这些流程解释关键函数在做什么。

## 1. 项目总览

Peirato's Piano 是一个 Wails v3 桌面应用：

- Go 后端负责桌面窗口、系统 MIDI 设备、本地音源合成、MIDI 文件解析和播放调度。
- Vue 3 前端负责 UI 渲染、键盘高亮、设置中心、MIDI 练习交互。
- Wails bindings 把 Go 的 `Keyboard` 服务暴露给前端，生成文件在 `frontend/bindings/main/service/`。
- Pinia store 是前端状态中心，定义在 `frontend/src/store/index.js`。
- Wails event 和 `windowBus` 共同负责后端到前端、主窗口到设置窗口 / MIDI 窗口的实时状态同步。

主要入口：

- `main.go`：程序入口，加载配置、初始化音源、启动 Wails。
- `service/app.go`：创建 Wails app、主窗口、设置中心窗口。
- `frontend/src/App.vue`：前端全局初始化，注册事件、加载配置和设备。
- `frontend/src/views/MainWindow.vue`：主钢琴窗口。
- `frontend/src/views/ControlCenterWindow.vue`：设置中心窗口。
- `frontend/src/views/MidiWindow.vue`：独立 MIDI 播放 / 练习长条窗口。

## 2. 启动流程

1. `main.go` 执行 `service.LoadConfig("1.1.6")`。
   - 读取用户配置目录里的 `config.json`。
   - 如果配置不存在，则用 `DefaultConfig` 创建默认配置。
   - 如果旧配置缺字段，则通过 `mergeConfigWithDefaults()` 补齐。

2. `main.go` 执行 `service.InitSoundFont(service.ResolveSoundFontPath())`。
   - `ResolveSoundFontPath()` 优先使用用户配置里的 `SoundFontPath`。
   - 路径无效时回退到 `./assets/Yamaha-Grand-Lite-v2.0.sf2`。
   - `InitSoundFont()` 调用 `LoadSoundFont()` 加载 sf2，再调用 `InitSpeaker()` 初始化本地扬声器。
   - 音源失败不会阻止窗口启动，只会进入“无内置发声”状态。

3. `service.Run(assets)` 创建 Wails 应用。
   - `application.NewService(&Keyboard{})` 把 `Keyboard` 的公开方法暴露给前端。
   - `AssetFileServerFS(assets)` 使用 `frontend/dist` 的嵌入式资源。
   - 主窗口 `PianoWin` 使用 URL `/`。
   - 后台启动 `ListenDevices()`，每 3 秒扫描一次 MIDI 设备。

4. 前端 `App.vue` mounted 后执行初始化。
   - `WML.Reload()` 启用 Wails 热重载支持。
   - `initKeyboardConfig()` 加载 `keyboard_config.json`、`case-1.json`、`ChordNames.json`。
   - `getConfig()` 从 Go 后端取配置并应用 CSS 变量。
   - `getMidiDevices()` 获取 MIDI 输入/输出设备快照。
   - `Keyboard.MidiListenerStart()` 启动当前输入设备监听。
   - 注册后端事件、窗口总线、电脑键盘事件和窗口 resize 事件。

## 3. 配置流程

配置模型在 `service/config.go`：

- `DefaultConfig`：默认颜色、键盘类型、音量、力度、MIDI 通道、音源路径。
- `UserConfig`：运行时全局配置，有 `configMu` 保护并发访问。
- `LoadConfig(version)`：读取、校验、补齐并保存配置。
- `SaveConfig(config)`：合并默认字段后写入磁盘。
- `GetUserConfig()`：返回当前配置快照。
- `SendConfig()`：Wails 绑定，前端启动时读取配置。
- `ReceiveConfig(config)`：Wails 绑定，前端设置变更后保存配置并发出 `configChanged`。
- `ResetConfig()`：恢复默认配置，保留当前版本号。

前端配置状态在 `store.config`。设置页修改配置后调用注入的 `changeConfig()`，最终走 `Keyboard.ReceiveConfig(store.config)`。

## 4. 主窗口演奏流程

电脑键盘演奏：

1. `App.vue` 的 `keyboardListener()` 监听全局 `keydown` / `keyup`。
2. `keydown` 根据 `store.keyMapping['case-1']` 找到 MIDI 音高。
3. 调用 `Keyboard.KeyboardPlay(midiKey)`。
4. 前端同时调用 `store.setKeyState(midiKey, true)`，让 UI 立即亮起。
5. `keyup` 调用 `Keyboard.KeyboardStop(midiKey)` 并清除前端按键状态。

鼠标演奏：

1. `ClassicKeyboard.vue` 负责渲染可见琴键。
2. `pressKey(key)` 设置 `store.pressedKey` / `store.activeKey`，并调用 `Keyboard.KeyboardPlay(key)`。
3. `releaseKey(key)` 调用 `Keyboard.KeyboardStop(key)`。
4. 鼠标拖过琴键时，`enterKey(key)` 会释放旧键并按下新键。

Go 后端发声：

- `KeyboardPlay()` 使用配置里的默认 velocity，并按当前输出目标发声。
- 输出目标互斥：`无` 不发音，`软件音源` 只触发内置音源，外部 MIDI 设备只发送 MIDI 消息。
- `KeyboardStop()` 按当前输出目标停止音符。
- `AllNotesOff()` 同时清理内置音源、外部 MIDI 输出和前端高亮。

## 5. MIDI 设备流程

设备状态在 `service/piano.go` 的 `Midis` 全局变量中：

- `InMidiPool`：输入设备列表。
- `OutMidiPool`：输出设备列表。
- `SelectedInDevice` / `SelectedOutDevice`：当前选中设备。
- `PedalStatus`：每个输入设备的踏板状态。
- `Listener.Started`：当前是否正在监听输入设备。

扫描流程：

- `ListenDevices()` 无限循环，每 3 秒调用 `ListenMidiDevices()`。
- `ListenMidiDevices()` 调用 `midi.GetInPorts()` / `midi.GetOutPorts()`，再分别交给 `CompareInDevices()` 和 `CompareOutDevices()`。
- `CompareInDevices()` 会添加新输入设备、删除已拔出设备，并在没有选择时自动选择最后一个输入设备。
- `CompareOutDevices()` 会打开新输出设备、删除已拔出设备，但不会自动选择系统 MIDI 输出；默认输出目标是软件音源。
- 扫描完成后通过 Wails 事件 `devices` 把快照推给前端。

输入监听：

- `MidiListenerStart()` 对当前输入设备调用 `midi.ListenTo()`。
- `handleMidiMessage()` 解析 NoteOn、NoteOff、ControlChange。
- NoteOn 会调用 `Keydown()` 内置发声，并发出 `down` 和 `pressedDown` 事件。
- NoteOff 会根据延音踏板状态决定是否马上 `Keyup()`，并发出 `pressedUp`，必要时发出 `up`。
- `handlePedalMessage()` 处理 64 延音、66 持音、67 柔音，并通过 `pedal` 事件推送前端。

前端接收：

- `App.vue` 的 `registerBackendEvents()` 监听 `down`、`up`、`pressedDown`、`pressedUp`、`pedal`、`devices`。
- `activeKey` 表示当前应该亮起或发声的音，受延音影响。
- `pressedKey` 表示用户手指真实按住的音，和弦识别优先使用它。

## 6. 音源流程

音源逻辑在 `service/soundfont.go`：

- `LoadSoundFont(path, sampleRate, bufferSize)` 读取并解析 `.sf2`，创建 `meltysynth.Synthesizer`。
- `InitSpeaker()` 使用 `beep/speaker` 初始化音频输出，并注册实时渲染 streamer。
- `Keydown(channel, key, velocity)` 调用 synthesizer 的 `NoteOn()`。
- `Keyup(channel, key)` 调用 `NoteOff()`。
- `AllSynthNotesOff()` 遍历 16 个 channel 和 128 个 key，确保内置音源不残留声音。

设置中心音源操作：

- `ImportSoundFontBase64(fileName, encoded)`：前端选择 `.sf2` 后以 base64 传给 Go，Go 保存到用户配置目录并加载。
- `SelectSoundFont(path)`：加载指定路径的音源，成功后写入配置。
- `ReloadSoundFont()`：按当前配置路径重新加载。
- `RestoreDefaultSoundFont()`：清空用户音源路径，回到默认音源。
- 音源变更后发出 `soundFontChanged` 和必要的 `configChanged`。

## 7. MIDI 文件解析与预览播放

MIDI 文件解析在 `service/midi_file.go`：

- `ParseMidiFileBase64(fileName, encoded)` 是 Wails 入口，解 base64 后调用 `ParseMidiFileBytes()`。
- `ParseMidiFileBytes()` 实现轻量 SMF 解析器：
  - 读取 `MThd` 文件头、format、track count、PPQ。
  - 逐轨读取 `MTrk`。
  - `parseMidiTrack()` 解析 tempo、track name、NoteOn、NoteOff 和 running status。
  - `normalizeTempoMap()` 整理 tempo 事件。
  - `ticksToSeconds()` 把 tick 转成秒。
  - 输出稳定的 `MidiFileInfo`、`MidiTrackInfo`、`MidiNote`。

预览播放在 `service/playback.go`：

- `LoadMidiFileBase64()` 解析 MIDI，并缓存到 `playback` runtime。
- `StartMidiPlayback()` 开始播放。
- `PauseMidiPlayback()` 暂停并释放当前活动音符。
- `StopMidiPlayback()` 停止、重置进度并清音。
- `SeekMidiPlayback(seconds)` 跳转进度。
- `SetMidiPlaybackRate(rate)` 设置播放速度，范围限制在 0.25 到 3。

播放调度：

- `midiPlaybackRuntime.run()` 每 10ms 计算当前播放时间。
- `collectPlaybackWork()` 找出该按下的音符和该释放的音符。
- 到点音符通过 `KeyboardPlayWithVelocity()` 发声。
- 结束音符通过 `KeyboardStop()` 停止。
- 通过 `midiPlayerState` 推送进度，通过 `playbackKey` 推送播放高亮。

前端预览 UI：

- `MidiWindow.vue` 的 `handleFileChange()` 读取文件并调用 `Keyboard.LoadMidiFileBase64()`。
- `playPreview()`、`pausePreview()`、`stopPreview()`、`seekPreview()`、`changePreviewRate()` 分别调用对应 Go 方法。
- `backendMidiService.js` 负责把 Go 返回的状态适配到 Pinia store。

## 8. 跟弹练习流程

练习计划生成在 `service/midi_file.go`：

- `BuildFollowPracticePlan(notes, options)` 是 Wails 入口。
- 根据 `threshold` 把开始时间接近的音符合并成一步。
- `inferTrackHands()` 根据轨道平均音高推断左手/右手。
- `normalizePracticeHand()` 处理练习手选择。
- 返回 `FollowPracticePlan`，包含：
  - `Steps`：每一步要练的音和自动播放的音。
  - `Assignments`：轨道到左右手的推断结果。
  - `AvailableHands`：前端可选练习模式。
  - `TotalPracticeNotes` / `TotalAutoPlayNotes`：统计信息。

前端跟弹逻辑在 `MidiWindow.vue`：

- `rebuildPracticePlan()` 根据当前 MIDI notes、区间、练习手生成计划。
- `startFollow()` 停止预览，清理旧状态，从第一步开始。
- `showCurrentFollowStep()` 设置当前步骤提示键，并调用 `playFollowAutoNotes()` 播放非练习声部。
- `watch(store.pressedKey)` 检查用户是否按对当前步骤所有音。
- `completeCurrentStep()` 命中后清提示并进入下一步。
- `stopFollow()` 停止自动伴奏并清理提示、错键、高亮。

自动伴奏：

- 前端调用 `Keyboard.PlayFollowAutoNotes(notes, playbackRate)`。
- Go 侧 `followAutoPlayRuntime.play()` 为每个自动音符启动 goroutine，按 duration 自动停止。
- `StopFollowAutoNotes()` 取消上一组自动伴奏，只清播放高亮，不清用户按键。

## 9. 多窗口同步

项目有多个 WebView：

- 主窗口：`PianoWin`，路由 `/`。
- 设置中心：`ControlWin`，路由 `/#/control`。
- MIDI 独立窗口：`MidiWin`，路由 `/#/midi`。

多个 WebView 的 Pinia 状态不是天然共享的，所以用两套通信：

- Go 到所有窗口：Wails `App.Event.Emit()`。
- 主窗口和设置中心之间：`frontend/src/services/windowBus.js`。

`windowBus` 优先使用 `BroadcastChannel`，不可用时退化为 `localStorage` 事件。它同步这些 UI 状态：

- `playback:key` / `playback:clear`：播放高亮。
- `hint:set` / `hint:clear`：跟弹提示键。
- `wrong:key`：错键提示。
- `all:clear`：清理全部前端键盘状态。

`service/app.go` 的 `OpenControlCenter()` 现在采用单例策略：

- 如果设置中心窗口已存在，则 `Show()` 并 `Focus()`。
- 否则创建窗口并加入 Wails app。

## 10. 已知质量点与后续建议

已处理：

- 502：Vite dev server 固定绑定 `0.0.0.0`，同时兼容 Wails 注入的 `localhost` 和 Windows 下的 IPv4 代理连接。
- 全局监听清理：`App.vue` 的 Wails events、windowBus、键盘和 resize 监听都在卸载时清理。
- 设置中心重复窗口：改为单例窗口。
- 冗余依赖：移除了未使用的 `axios` 和已迁移掉的 `@tonejs/midi`。

建议后续小步处理：

- `PedalSingal` 应改为 `PedalSignal`，但这会影响生成 bindings，需要统一改 Go、前端模型和生成文件。
- `Sythesizer` 应改为 `Synthesizer`。
- `keybordType` 应改为 `keyboardTypeOptions`，目前前端多个组件依赖该字段。
- `ListenDevices()` 是无限循环，当前依赖进程退出结束；如后续做后台测试或可重启服务，建议加 context/cancel。
- `ControlWin` 关闭后的生命周期仍受 Wails v3 alpha 行为影响；升级 Wails 后可以接窗口关闭回调，把 `ControlWin` 明确置空。

## 11. 排障手册

### Wails 窗口显示 502

检查点：

1. 单独运行 `npm run dev -- --host 0.0.0.0 --port 9245 --strictPort`。
2. 确认 `http://localhost:9245/` 和 `http://127.0.0.1:9245/` 都返回 200。
3. 再运行 `wails3 dev -config ./build/config.yml -port 9245`。
4. 如果仍失败，检查是否已有旧的 `node.exe`、`wails3.exe`、`bin/app.exe` 占用或悬挂。
5. 本机曾观察到 `localhost:9245` 与 `127.0.0.1:9245` 可访问性不一致，所以两个地址都要测。
6. `0.0.0.0` 只用于开发模式；如果处在不可信网络，优先断开外部网络或调整防火墙。

### 音源加载失败

检查点：

1. 默认音源是否存在：`assets/Yamaha-Grand-Lite-v2.0.sf2`。
2. 用户配置里的 `soundFontPath` 是否还存在。
3. 设置中心音源页的 `soundFontInfo.error`。
4. 失败后程序仍可打开，但本地内置发声不可用。

### MIDI 设备不可用

检查点：

1. 设置中心 MIDI 设备页是否出现输入/输出设备。
2. 外部设备是否被其他软件独占。
3. 切换输入设备时前端会先 `MidiListenerStop()`，再 `ChangeDevice()`，最后 `MidiListenerStart()`。
4. 切换输出设备时只 `AllNotesOff()`，不会重启输入监听。

### 卡音或残留高亮

检查点：

1. 主窗口停止按钮会调用 `Keyboard.AllNotesOff()`。
2. MIDI 预览停止会调用 `StopMidiPlayback()`，最终也会清音。
3. 跟弹停止会调用 `StopFollowAutoNotes()`、清前端提示和播放高亮。
4. 延音踏板会延迟 NoteOff，高亮是否残留要区分真实延音和错误状态。

### Bindings 失配

如果修改了 Go 暴露给前端的结构或方法：

1. 运行 `wails3 generate bindings`。
2. 检查 `frontend/bindings/main/service/` 是否更新。
3. 再运行 `npm run build`。
4. 不要手改 bindings 生成文件。

## 11. MIDI 独立窗口产品逻辑

MIDI 播放 / 练习现在只存在于 `frontend/src/views/MidiWindow.vue`，不再挂在设置面板里。窗口由 `service/app.go` 创建为 `MidiWin`，形态是长条状独立窗口。

左侧目录由 `MidiDirectory.vue` 渲染，数据来自 `useMidiLibrary.js`。目录只保存 MIDI 的系统绝对路径，持久化 key 为 `peirato-piano-midi-library-v1`。移除记录只删除 localStorage 记录，不会删除真实文件。

右侧控制区分两层：

- 播放模式：直接调用 Go 播放器 `SeekMidiPlayback / StartMidiPlayback / PauseMidiPlayback / StopMidiPlayback / SetMidiPlaybackRate`。
- 练习模式：调用 `BuildFollowPracticePlan` 生成跟弹计划，再由前端根据用户真实按键 `pressedKey` 推进步骤。非练习声部调用 `PlayFollowAutoNotes` 自动播放。

底部进度条由 `AnchorProgress.vue` 管理。左锚点永远不能超过右锚点，右锚点永远不能小于左锚点。普通播放到右锚点会停止；练习模式生成计划时只取锚点范围内的音符。

需要注意：Go 侧新增的 `CheckMidiPathExists` 和 `LoadMidiFileFromPath` 必须重新执行 `wails3 generate bindings` 后，前端才能通过绝对路径重新加载历史 MIDI 记录。
