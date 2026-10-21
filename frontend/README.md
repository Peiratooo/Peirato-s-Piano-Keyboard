# Peirato's Piano 前端

Vue 3、Pinia、Naive UI 和 Vite 实现主钢琴窗口、设置中心与 MIDI 中心。

在项目根目录运行 `wails3 task dev` 启动完整桌面开发环境；前端依赖通过 `npm ci` 安装。Wails 绑定由 `common:generate:bindings` 任务生成，不提交到仓库。

生产构建与安装包通过 Wails3 Manager 的 `release` 任务完成，配置和资源说明见根目录 README。单独检查前端构建可在本目录执行 `npm run build`，需要先生成绑定。
