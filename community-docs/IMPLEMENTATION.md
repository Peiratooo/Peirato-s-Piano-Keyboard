# PP 钢琴社区：运行与实现说明

本文件是当前实现的入口。其他交接文档保留原包背景，不代表当前完成状态。

播放与跟弹琴键沿用 MIDI 轨道的左右手配色，跟弹提示被接受后仍保留按住音符的手别。双端 MIDI 发声力度使用 `round(原始 MIDI 力度 × 触键力度 / 127)`，播放、恢复与跳转后的持续音符均按当前设置计算；设为 127 保留原始力度，主音量独立控制。桌面内置音源和外部 MIDI 输出使用同一缩放后的力度。

## App 与网页界面统一

- App 为视觉来源；网页直接使用桌面端的 ClassicKeyboard、WaterfallCanvas、MidiWaterfallCanvas、MidiTransport、MidiPracticeControls、MidiLibraryPanel、RecordingControls、QuickPianoSettings、CommunityPanel、Chord 与 FeedbackBridge。
- 主窗口、完整设置及音源面板样式集中在 `shared/piano/`，配色、键盘布局、键盘映射标签与编辑器沿用桌面源码；删除网页原有的另一套琴键、瀑布、时间轴和主题实现。
- 网页的平台接入集中在 `web-toy/src/composables/usePianoPresentation.js`，连接现有 Web Audio/MIDI 引擎、IndexedDB、偏好和按键方案。没有引入 Wails 运行时或旧接口兼容分支。
- 网页保留琴房与内容审核入口；完整设置采用 App 的设置样式，浏览器管理的音频参数显示实际只读信息。操作系统窗口按钮由桌面平台提供。
- 琴键使用同一套 Pointer Events 处理鼠标与触屏；手机仅调整工具栏换行、弹窗宽度和可滚动区域，控件与主题保持一致。

## 本地启动

环境：Go 1.27.3、Node 24；桌面依赖现有 MIDI/音频构建环境。

```bash
cd server
go run . -inspect-db # 只读检查数据库版本、建表权限和表名
go run .             # 读取工作目录 .env，初始化表并启动 API
```

另一个终端：

```bash
cd web-toy
npm ci
npm run dev
```

也可在根目录执行 `./scripts/dev-community.sh` 同时启动二者。本地 Toy 地址为 http://127.0.0.1:5173，API 为 http://127.0.0.1:8080。

实际数据库配置保存在被 Git 忽略的 `server/.env`，权限 0600，不随源码或产物分发。仅服务端连接数据库。可提交的格式见 `server/.env.example`。后端使用 `DB_HOST/PORT/USER/PASSWORD/NAME/SSLMODE/TIMEZONE`，没有 SQLite 或旧 DSN 分支。

`ADMIN_USERNAME` 与 `ADMIN_PASSWORD` 首次启动创建管理员。当前本机配置已生成独立的管理员口令，保存在上述私有文件中。普通用户在界面注册自己的账号。

## 桌面构建与联动

保持项目现有 Wails 构建流程；生成绑定时 CLI 版本必须与根 `go.mod` 中的 Wails 一致：

```bash
go run github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.24 generate bindings -d frontend/bindings .
npm run build --prefix frontend
CGO_CFLAGS=-mmacosx-version-min=12.0 CGO_LDFLAGS=-mmacosx-version-min=12.0 MACOSX_DEPLOYMENT_TARGET=12.0 go build -tags production -o bin/pp_keyboard-community .
```

必须先完成绑定生成，再构建前端，最后构建 Go 桌面程序。

- 菜单与工具栏的“工坊”进入社区，提供工坊、我的练习、账号三个面板。
- 默认 API 地址为本机；运行环境变量 `PP_COMMUNITY_API_URL` 或构建变量 `main/service.CommunityAPIURL` 可指定服务器。
- 自建账号为默认登录；“通过 Toy 登录”打开服务端 `TOY_URL`，使用同一套社区账号确认，桌面自动领取独立会话。
- 桌面会话只保存在 Go 管理的 Application Support 中，文件名 `community-session.json`、权限 0600；不保存密码，不放入 WebView 存储。
- 会话有效期 7 天。失效后清除；连接失败时保留已缓存账号显示并提示离线，不伪造在线认证。
- 昵称与头像自行设置，不调用 Toy 的用户资料 API。头像为 PNG/JPEG、2MB 以内。
- 缓存曲目保留社区 ID 和内容哈希，私密/待审曲目按账号区分；公开缓存和本地原有曲目离线可用。

## 工坊与接口

`GET /api/midis`：

- `scope=public|mine|favorites`；后两项需要登录。
- `q` 搜索标题、作曲者、上传者昵称和标签；`tag` 精确筛选，`difficulty=1..5`。
- `sort=latest|likes|favorites|views|plays|downloads`，默认最新。
- `page` 默认 1，`pageSize` 默认 20、最大 50；响应 `{items,total,page,pageSize}`。

详情和列表包含六项公开统计、`myReaction`、`myFavorite`；私人状态只对应当前登录者。

- `PUT /api/midis/{id}/interaction`：`{reaction:-1|0|1,favorite:boolean}`，按目标状态保存。
- `POST /api/midis/{id}/events`：`{type:"view"|"play"|"download"}`，可见性检查通过后累计。
- 主动打开详情计浏览；从停止成功开始计播放，暂停恢复/循环不重复计；显式导出文件计下载，缓存抓取不计。
- 原文件由 `GET /api/midis/{id}/file` 鉴权提供，不直接暴露磁盘目录。
- 上传最大 8MB；个人最多 100 首/100MB。上传后服务端解析音乐元数据，无调号时显示未标注。
- 上传、编辑、删除、公开/私密切换均由所有者操作。公开资料变化重新待审；下架状态不能通过切换私密绕过。
- 管理员在 Toy 中审核、试听、拒绝、解除下架和处理举报；不通过管理员曲库列出普通私密作品。

账号资料接口：`GET/PATCH /api/auth/me`、`POST /api/auth/avatar`。
桌面授权接口：`POST /api/auth/desktop/start|confirm|poll`。授权 5 分钟、轮询 5 秒、领取一次失效；授权状态存单实例内存，服务重启后须重新发起。

数据库共七张扁平表：`users`、`sessions`、`rooms`、`midi_files`、`practice_logs`、`reports`、`midi_interactions`。赞踩收藏由互动记录汇总，浏览/播放/下载使用曲目计数列；没有新建社交关系或分布式服务。

## 练习记录与边界

双端通过同一个生命周期模块记录工坊曲目的练习；使用服务端时间，20 秒心跳、最多累计最近 40 秒；暂停、停止、换曲及退出结束记录。离线期间不补写推测时长。“继续练习”载入相应曲目和最近模式。

Toy 保留单实例琴房和 WebSocket 功能，桌面本轮未接入实时琴房。没有新增歌单、评论、关注、按日访客去重或推荐算法。

## 验证与产物

现有测试：`cd server && go test ./...`；`cd web-toy && npm run test:timing`。没有新增测试文件或 GitHub Actions。

`./scripts/build-release.sh` 生成当前目标系统的 API 和 Toy 静态文件至 `release/`。指定 `GOOS=linux GOARCH=amd64` 可交叉构建 Linux API（PostgreSQL 不依赖 CGO）。桌面仍通过现有 Wails 构建。

本轮真实验证明细见 `VALIDATION.md`。真实 Android/iOS B站容器、硬件 MIDI 和公网 HTTPS/WSS 未经本机结果替代验收。

## 图一工坊布局

App 与 Toy 使用 shared/community/CommunityWorkspace.vue 的完整工作区，账号使用独立的 AccountDialog；曲库范围从侧栏切换。WorkshopPanel 提供无封面信息卡片和曲目详情，WorkspacePlayer 连接已有播放会话。试听保留网格，跟弹返回原钢琴界面。

强调色全部读取现有 getPianoUiTheme 产生的 CSS 变量和 Naive UI 配置；切换设置主题会同步更新主按钮、选中项、卡片描边及播放进度。没有新增独立配色或 MIDI 封面字段。宽屏四列、一般桌面三列/两列、手机竖屏单列；搜索、标签、难度、排序和分页保留真实 API 行为。
