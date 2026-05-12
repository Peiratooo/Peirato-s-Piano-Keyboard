# Peirato's Piano 重构记录

## 本轮重构目标

本轮不是大规模推倒重写，而是围绕后续开发做“低风险整理”：

1. 稳定 MIDI 输入 / 输出 / 监听生命周期。
2. 稳定音源加载，预留用户导入 SoundFont。
3. 主窗口和设置中心窗口分离。
4. 和弦识别从 App.vue 临时逻辑中独立出来。
5. 为后续 MIDI 播放和跟弹模式预留结构。

## 已完成

### 后端

- `main.go` 改为按顺序初始化：配置 -> 音源 -> 窗口。
- `config.go` 增加配置锁、默认配置合并、`midiChannel`、`soundFontPath`。
- `soundfont.go` 增加音源加载错误处理，避免音源文件缺失时 panic。
- `soundfont.go` 对 Synth 的 Render / NoteOn / NoteOff 加锁，降低并发风险。
- `piano.go` 统一 MIDI 设备 ID，不再混用数组下标和 `port.Number()`。
- `piano.go` 修复 ChangeDevice 的 map 校验。
- `piano.go` 修复 MIDI listener 重复启动问题。
- `piano.go` 增加 `AllNotesOff()`，用于设备切换、退出、后续播放器停止。
- `piano.go` 增加 `pressedDown` / `pressedUp` 事件，让和弦识别可基于真实按键，而不是延音后的 activeKey。
- `app.go` 增加明确的 `Keyboard.OpenControlCenter()` 设置中心窗口入口。

### 前端

- `App.vue` 改为统一初始化层和事件桥接层。
- 新增 `views/MainWindow.vue`：负责主窗口钢琴展示和轻量操作。
- 新增 `views/ControlCenterWindow.vue`：负责设置中心窗口。
- `Setting.vue` 改为左侧菜单 + 右侧内容区，预留音源、MIDI 播放、跟弹模式。
- `ClassicKeyboard.vue` 改为明确的 press/release 流程，减少鼠标弹奏卡音风险。
- `Chord.vue` 改为使用 `services/chordEngine.js`，并加入 40ms 防抖。
- `store/index.js` 增加 `pressedKey`、`playbackKey`、`hintKey`、`wrongKey`，为和弦、播放、跟弹分层。
- `services/midiPlaybackController.js` 仅保留播放器状态常量，真实解析和播放调度统一由 Go / Wails 服务负责。

## 下一步建议

1. 在本地 Wails 环境重新生成 bindings。
2. 运行前端 `npm install && npm run build`。
3. 运行 Wails 开发模式，重点测试：设备扫描、设备切换、电脑键盘弹奏、鼠标弹奏、和弦识别、设置中心窗口。
4. 如果设置中心窗口需要严格单例，等确认当前 Wails v3 alpha 窗口 API 后补窗口关闭回调和聚焦逻辑。
5. 下一阶段实现 MIDI 播放 MVP。

## 本环境校验情况

已完成：

- `gofmt -w main.go service/*.go`
- 前端 `.js` 与 `.vue` 中 `<script setup>` 片段的 Node 语法检查

本容器未完成：

- `go test ./...`：当前容器 Go 为 1.23.2，而项目 `go.mod` 要求 Go 1.24.3。
- `npm run build`：当前压缩包未包含 `node_modules`，容器中没有 `vite`，且离线环境无法完整安装依赖。

建议在你的本地开发环境执行：

```bash
cd frontend
npm install
npm run build

cd ..
wails3 generate bindings
wails3 dev
```

---

## Step 2：MIDI 播放 MVP 与跟弹雏形

### 本轮目标

在不大改主窗口的前提下，把 MIDI 播放能力放进设置中心，让主窗口继续只负责钢琴展示和实时反馈。

### 已完成

- 新增 `services/windowBus.js`：用于主窗口和设置中心窗口同步播放高亮、跟弹提示等 UI 状态。
- MIDI 播放调度已迁移到 Go，前端只保留状态常量和 UI 渲染。
- 旧的 `MidiPlayerPanel.vue / FollowPlayPanel.vue` 已合并为 `MidiPracticePanel.vue`。
- `store/index.js` 增加 `player` 状态，集中保存 MIDI 播放文件、播放进度、状态和速率。
- `App.vue` 接入窗口通信事件，让设置中心播放 MIDI 时，主窗口琴键能同步高亮。
- `MainWindow.vue` 新增“停止所有声音”按钮，复用后端 `AllNotesOff` 能力。

### 设计说明

当前播放器统一使用 Go 侧 Wails 绑定完成调度和发声。修改 Go 暴露方法后，必须执行 `wails3 generate bindings`。

跟弹模式目前是第一版闭环：

1. 复用 MIDI 播放页导入的 notes。
2. 按 60ms 阈值把同时出现的音分成一组。
3. 当前组通过 `hintKey` 在主窗口闪烁提示。
4. 用户按对当前组后进入下一组。

### 下一步建议

1. 本地运行设置中心，验证 MIDI 文件导入、播放、暂停、停止和倍速控制。
2. 验证两个窗口之间的琴键高亮同步是否稳定。
3. 重新生成 Wails bindings，确保新增的 Go 方法在前端可直接调用。
4. 下一阶段可以继续完善跟弹模式：错误按键提示、轨道筛选、左右手练习、练习进度和重置逻辑。

## Step 3 - Wails 绑定服务与 Go 后端化

本阶段目标：把适合后端处理的 MIDI 解析、播放调度、跟弹分组和音源管理逐步迁移到 Go，通过 Wails3 绑定提供给前端。

### 已完成

1. 新增 `service/midi_file.go`
   - Go 侧轻量 Standard MIDI File 解析器。
   - 支持 `MThd / MTrk`、PPQ 时间、tempo meta event、running status、note on/off 配对。
   - 输出统一的 `MidiFileInfo / MidiTrackInfo / MidiNote`，前端不再依赖第三方库结构。
   - 暴露 Wails 方法：
     - `ParseMidiFileBase64(fileName, encoded)`
     - `LoadMidiFileBase64(fileName, encoded)`
     - `BuildFollowGroups(notes, threshold)`

2. 新增 `service/playback.go`
   - Go 侧 MIDI 播放器调度器。
   - 支持播放、暂停、停止、Seek、倍速。
   - 播放时通过后端直接调用 `KeyboardPlayWithVelocity / KeyboardStop`。
   - 向前端推送：
     - `midiPlayerLoaded`
     - `midiPlayerState`
     - `playbackKey`
     - `playbackClear`

3. 扩展 `service/soundfont.go`
   - 新增音源管理 Wails 方法：
     - `SelectSoundFont(path)`
     - `ReloadSoundFont()`
     - `RestoreDefaultSoundFont()`
   - 音源加载成功后才写入配置，避免坏路径污染下次启动。

4. 优化 `service/piano.go`
   - `KeyboardPlayWithVelocity` 现在使用 “MIDI velocity × 用户音量” 计算本地音源力度。
   - MIDI 文件播放可以保留自身力度，同时主窗口音量滑块仍然生效。

5. 新增 `frontend/src/services/backendMidiService.js`
   - 前端优先调用 Go 后端 Wails 绑定。
   - 前端不再保留前端解析/播放回退逻辑，必须通过 Wails 绑定调用 Go 服务。

6. 更新设置中心
   - MIDI 播放页优先使用 Go 后端播放器。
   - 跟弹模式优先使用 Go 后端分组。
   - 音源管理页支持重新加载和恢复默认音源。

### 当前设计原则

- 前端只负责 UI、交互、展示状态。
- MIDI 文件解析、播放调度、跟弹分组、音源切换逐步迁到 Go。
- 前端只走正式 Wails 绑定路径，避免两套播放逻辑长期并存。

### 本地需要执行

```bash
wails3 generate bindings
cd frontend
npm install
npm run build
cd ..
wails3 dev
```

### 后续建议

1. 把文件选择也迁移到 Go / Wails 原生 Dialog。
2. 跟弹模式的错误判定、进度控制继续放到 Go 侧。
3. 增加 track 过滤播放，让后端播放器只播放选中的轨道。
4. 增加循环播放、区间练习、速度渐进练习。

## Step 4 - 后端服务迁移与跟弹练习增强

本轮目标：按桌面软件思路继续把高计算/状态核心迁到 Go，同时保留前端选择文件的交互方式。

### 已完成

1. **文件导入策略调整**
   - MIDI 文件继续由前端 `<input type="file">` 选择，再以 base64 传给 Go 解析。
   - SF2 音源也改成同样方式：前端选择 `.sf2`，后端保存到用户配置目录下的 `soundfonts/`，加载成功后写入配置。
   - 不使用 Wails 原生 Dialog，保持 MIDI 和 SF2 导入方式一致。

2. **跟弹模式升级**
   - 新增 Go 侧 `BuildFollowPracticePlan`。
   - 跟弹区间只作用于练习计划，不影响普通 MIDI 播放器。
   - 支持练习声部：单手 / 双手 / 只练右手 / 只练左手。
   - 多轨 MIDI 会根据轨道平均音高推断左右手：平均音高低的一半为左手，高的一半为右手。
   - 单轨 MIDI 会强制进入 `single`，前端只允许单手练习。
   - 练右手时左手自动播放，练左手时右手自动播放。
   - 前端负责显示步骤、判断用户按键、错误提示和步骤推进。

3. **Go 侧跟弹自动声部播放**
   - 新增 `PlayFollowAutoNotes`，用于播放非练习声部。
   - 自动播放音符会触发主窗口 `playbackKey` 高亮，并按音符 duration 自动停止。

4. **依赖版本更新**
   - 更新 `go.mod` 中 Wails v3 到 `v3.0.0-alpha.87`。
   - 更新前端 `package.json` 里的核心依赖版本。
   - 删除旧的 `package-lock.json`，需要本地重新 `npm install` 生成新的 lockfile。

### 后续建议

1. 本地执行 `wails3 generate bindings`，让新增的 Go 方法生成到前端 bindings。
2. 本地执行 `npm install` 重新生成 `package-lock.json`。
3. 重点测试：
   - 导入 SF2 音源后是否能正常发声；
   - 多轨 MIDI 的左右手推断是否符合常见文件；
   - 单轨 MIDI 是否正确禁用左右手选择；
   - 跟弹区间起止时间是否正确；
   - 练右手时左手声部是否自动播放。

## Step 5 - 合并 MIDI 预览与跟弹练习

本轮目标：按实际产品使用流程，把“播放”和“跟弹”合并到同一个练习入口里。播放不再作为独立模块理解，而是导入 MIDI 后的“整首预览”；预览后再进入跟弹练习。

### 已完成

1. **设置中心菜单合并**
   - 原来的 `MIDI 播放` 与 `跟弹模式` 合并为 `MIDI 练习`。
   - 新增 `frontend/src/components/settings/MidiPracticePanel.vue`。
   - `Setting.vue` 只挂载一个练习中心页面，减少用户在两个菜单之间切换。

2. **练习中心流程重组**
   - 页面流程变成：导入 MIDI -> 预览整首 -> 跟弹练习。
   - 预览播放继续复用 Go 后端播放器：播放 / 暂停 / 停止 / Seek / 倍速。
   - 跟弹练习继续复用 Go 后端练习计划：区间、左右手、非练习声部自动播放。
   - 播放器不做区间播放，区间只存在于跟弹练习里。

3. **状态互斥更清晰**
   - 开始预览时会停止跟弹。
   - 开始跟弹时会停止预览。
   - 切换跟弹步骤时会清掉上一组提示和错误状态。
   - 停止跟弹时会清理 hintKey / wrongKey / playbackKey，避免主窗口残留高亮。

4. **Go 侧自动伴奏增加停止能力**
   - `PlayFollowAutoNotes` 现在由一个可取消的 runtime 管理。
   - 新增 `StopFollowAutoNotes()`，用于上一步 / 下一步 / 停止跟弹时取消上一组自动声部。
   - 这里不会触发全局 `AllNotesOff`，避免切换步骤时误清理下一组提示键。

### 当前设计说明

- `MIDI 练习` 是用户入口。
- `预览播放` 是为了让用户先听整首 MIDI。
- `跟弹练习` 是为了让用户按区间、左右手策略练习。
- 主窗口仍然只负责钢琴视觉反馈，不承担播放器 UI。
- Go 负责解析、预览播放、练习计划和自动声部播放；前端负责选择文件、展示状态和判断用户按键。

### 下一步建议

1. 给练习中心增加“小节/时间轴”选择体验，而不是只输入起止秒数。
2. 支持手动修正左右手轨道归属。
3. 增加练习统计：正确音、错误音、完成时间、重练次数。
4. 增加练习结束页。


## Step 6 - 跟弹补丁与绑定正式化

本阶段目标：补齐跟弹练习的几个关键体验点，并删除为了兼容“未生成 bindings”的前端回退逻辑。

### 已完成

1. 跟弹步骤完成反馈
   - 当前步骤全部按对后，会出现轻量成功提示。
   - 完成后短暂停留，再进入下一步，减少状态跳变带来的突兀感。

2. 重新开始当前练习
   - `MidiPracticePanel.vue` 增加“重新开始当前练习”按钮。
   - 保留当前 MIDI、区间、左右手设置，从第 1 步重新开始。

3. 重新导入 MIDI 时强制停止旧练习
   - 导入新 MIDI 前会先停止预览、停止跟弹自动伴奏、清空提示键和错误键，并调用 `AllNotesOff()`。
   - 避免旧 MIDI 的自动声部、播放高亮或跟弹提示残留到新文件。

4. 正式绑定模式
   - 删除旧的 `MidiPlayerPanel.vue / FollowPlayPanel.vue`。
   - 删除前端 MIDI 解析和前端播放器回退实现。
   - `MainWindow.vue` 改为直接调用 `Keyboard.OpenControlCenter()` 和 `Keyboard.AllNotesOff()`。
   - `OpenUrl()` 回归外部链接职责，不再承载内部协议。

### 注意

这一步之后，前端依赖新的 Wails 绑定。拉取本版本后，请务必执行：

```bash
wails3 generate bindings
```

否则旧的 `frontend/bindings` 里不会包含 `OpenControlCenter / AllNotesOff / LoadMidiFileBase64 / BuildFollowPracticePlan / PlayFollowAutoNotes` 等新方法。
