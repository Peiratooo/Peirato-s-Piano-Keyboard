> 原始交接包参考文档。当前实现、配置和验收以 [IMPLEMENTATION.md](IMPLEMENTATION.md) 与 [VALIDATION.md](VALIDATION.md) 为准；下文的旧任务建议不是本次执行指令。

# 本地 Codex 接手说明 · PP Piano Community

**生成日期**：2026-10-28\
**状态**：已包含上一轮完整源码 + 本轮 Toy SDK 类型、仓库 Skill、产品路线文档；**并非已经通过生产编译的部署包**。

## 第一件事：读文档，不要重新设计整套项目

根目录打开后依次读取：`AGENTS.md` → 本文件 → `docs/PRODUCT_AND_ROADMAP.md` → `docs/TOY_SDK_INTEGRATION.md` → `.agents/skills/bilibili-toy/SKILL.md` → 相关源代码。涉及 Toy 开发时使用项目 Skill `bilibili-toy`。

**特别注意**：项目 `web-toy/types/toy.d.ts` 已包含作者提供的 Toy SDK 完整 API 类型（包括补充的共享存档），并通过 `jsconfig.json` 为 JavaScript 工程提供编辑器类型提示；**没有集成官方运行时 SDK 包**，运行时仍由 `web-toy/index.html` 引入 Toy SDK 脚本。

## 本次前置环境

| 项目 | 当前信息 |
|---|---|
| 源码 | `server/`、`web-toy/`、`deploy/`、`scripts/` 均存在 |
| Go | `go.mod` 使用 Go 1.23/GORM/Gorilla WebSocket；默认 SQLite 有 CGO/GCC 依赖 |
| 前端 | Vue 3 / Vite；`web-toy/package.json` 已有 `dev`、`build`、`test:timing` |
| SDK | index.html 已引入 `toy-sdk.js`；本次新增完整 ambient 类型、能力规范和 Codex Skill |
| 构建 | 以前的环境无法访问 Go/npm 模块源，**既没有被验证的 Go 二进制，也没有被验证的 Vite dist** |
| Git 及部署 | 不代表已推送 GitHub；没有线上发布授权；线上域名/Token/账号由用户另行提供 |

## 本地 Codex 的第一个里程碑：完成真正的 build + 端到端 MVP 验证

**目标**：先让现有工程“可编译、可启动、两人可实时弹奏”，再继续产品功能开发。不能重新生成一套同名模拟代码，也不能只做静态检查就宣称编译成功。

### 建议执行顺序

1. `node -v && npm -v && go version && gcc --version`，记录环境；检查完整源码与依赖。注意 `server/.env.example` **不会自动加载**，运行前需正确导出变量或用容器 env-file。
2. `cd web-toy && npm install && npm run test:timing && npm run build`；修复 Vue/Vite 打包错误和真实浏览器 console 错误。生成并提交 `package-lock.json`，后续 CI 用 `npm ci`。
3. `cd ../server && go mod tidy && go test ./... && go build -o ../release-local/pp-piano-api .`；修复 Go 编译、测试、逻辑错误；提交 `go.sum`。
4. 准备 **仅本机开发**环境 `CORS_ORIGINS=http://localhost:5173`、`ADDR=127.0.0.1:8080`、SQLite；启动 `go run .`、`npm run dev`。执行 `GET /api/health`。
5. 用 A、B 两个不同账号/浏览器验证：A→上台，B→排队；A 主动下场，B 接管；到期和断连宽限；观众无权发送琴键；前后台/切网清音、重连同步。
6. 验证个人私密房只有带凭证能进，随机只显示正在演奏的公开房；验证私密 MIDI 权限、公开审核、解析调号/小节、练习开始/暂停/结束与统计。
7. 在 B站 Toy 真实容器验收 SDK 载入、横屏沉浸、分享/二维码、外部 HTTPS/WSS、Web Audio 解锁；如有不支持功能，设置明确回退界面。
8. 通过后执行 `./scripts/build-release.sh` 或 Docker/GitHub Actions 构建，输出准确产物及 SHA256；提交可复现构建日志与剩余问题。

### 最先修的风险清单

- 完整后端与前端还没有可靠的在线依赖安装、构建记录。**P0 首要是跑通构建**，不是再增加功能。
- B站 Toy 容器对外部 HTTPS/WSS 的允许规则未知，必须真机确认。没有网络支持时房间业务不可用，保留本地钢琴。
- 后端 JWT/第三方授权不应凭空添加。原版用独立账号 Session；`toyOpenId` 非凭证，不能免注册伪登录。
- MIDI 音源表现、WebView 多指触控、页面隐藏、内存占用与网络抖动需要实测，不得宣称与桌面版完全一致。
- 当前服务端为单进程 Hub，部署多副本会破坏全局唯一舞台和排队公平。先单实例稳定运行。
- 审核/举报是 MVP：没有完善的申诉、封禁、违规存储清理、安全审计，正式公开运营前必须补足。

## 验收任务拆解（可以逐一交给 Codex）

| 任务 | 改动范围 | 验收结果 |
|---|---|---|
| T-01 真正构建工程 | Go npm 锁文件、脚本/CI | `go test ./...`、`go build`、`npm run build` 成功并有日志 |
| T-02 实时席位联机测试 | `server/ws.go`、`web-toy/src/services/community.js` | 双账号排队/到期/让位/断线/观众越权通过 |
| T-03 Toy 兼容与触控 | `web-toy`、Skill/SDK 封装 | Android/iOS/桌面逐项记录兼容结果及降级 |
| T-04 MIDI 权限和文件健壮性 | Go MIDI/文件 API | 私密不可越权，畸形文件无法拖垮服务 |
| T-05 演奏音频对齐 | audio.js/usePiano.js/瀑布 | 踏板、力度、重音和切后台没有永久卡音 |
| T-06 产品整合与体验 | 前端组件、交互 | 对标桌面版简洁键盘/瀑布流，不遮挡主舞台 |
| T-07 部署/运营准备 | deploy/ docs/ server | HTTPS/WSS、CORS、代理 IP、备份/恢复、审核值守可运行 |

## 复制到本地 Codex 的首条任务（可直接使用）

> 请先阅读根目录 AGENTS.md、docs/CODEX_HANDOFF.md、docs/PRODUCT_AND_ROADMAP.md、docs/TOY_SDK_INTEGRATION.md，以及 .agents/skills/bilibili-toy/SKILL.md。这个项目是 PP 钢琴键盘的 B站 Toy 网页社区版，保留现有 Vue 钢琴实现、Go/GORM 六张扁平表和大众琴房的 5 分钟排队规则，不要重写一套应用。请优先执行 T-01：在本地真实安装依赖、完整编译 Go 后端和 Vite 前端，运行全部自动测试，修复遇到的实际错误，生成 go.sum/package-lock.json，并说明测试日志、修改文件、可运行命令与未完成风险。之后执行 T-02 的双浏览器联机回归。没有明确授权不要部署公网或修改线上数据库。

## 下一阶段建议怎么继续

- T-01/T-02 完成后，按 `docs/PRODUCT_AND_ROADMAP.md` 的 P1 → P2 顺序推进。
- 每完成一项维护一份真实的 changelog/测试日志，不要将暂定设计写成已实现状态。
- 严格遵守 Toy SDK 支持范围，特别是用户数据与安全约束。Toy 的跨端类型变化优先更新 `types/toy.d.ts` + `docs/TOY_SDK_INTEGRATION.md`。
