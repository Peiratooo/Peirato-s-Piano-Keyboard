> 原始交接包参考文档。当前实现、配置和验收以 [IMPLEMENTATION.md](IMPLEMENTATION.md) 与 [VALIDATION.md](VALIDATION.md) 为准；下文的旧任务建议不是本次执行指令。

# B站 Toy SDK 集成参考（给本地 Codex）

> 来源：项目作者在 2026-10-28 会话提供的 **Toy JS SDK 类型声明** 和 **2026-09-18 能力清单**。该文件是本项目的集成参考，不声称是 B站的官方在线文档。**源资料自身存在差异，接入必须以真实 Toy 容器测试和平台正式说明为准。**

## 1. 当前项目如何接入

- `web-toy/index.html` 的 `<head>` 已加入：`<script src="//s1.hdslb.com/bfs/seed/toy/app/sdk/toy-sdk.js" defer></script>`。生产建议在 HTTPS 页使用 HTTPS SDK URL；检查 Toy 的 CSP 和网络策略。
- **完整 Ambient TypeScript 类型**：`web-toy/types/toy.d.ts`；`web-toy/jsconfig.json` 将其纳入项目，供 JavaScript/JSDoc 编辑器补全。
- **封装入口**：`web-toy/src/services/toy.js`，已有 `support`、`watchToyContainer`、`immerseToy`、`shareToy`、`saveToyPreferences`、`loadToyPreferences`。新增调用应先复用封装。
- SDK 可能未加载或被浏览器阻止。`window.toy` 在类型中被设为可选；所有 SDK 调用均需检测存在和使用 try/catch。
- SDK 没有 MIDI 文件仓库、服务器端用户鉴权、WebSocket 房间或乐器合成器；这些使用本项目自己的 Go API、WebSocket 和 Web Audio。

## 2. 方法速查

| 能力 | B站 App 内 | Web | 要点 |
|---|---|---|---|
| `isSupport` | 支持 | 支持 | 优先做特性探测，失败降级 |
| `navigate` | 支持 | 支持 | 仅用户手势触发；站内视频/动态/作者/Toy |
| `saveImageToAlbum` | 支持 | 不支持 | base64 字符串上限 5,242,880；较大图优先 URL |
| `share` | 支持 | 不支持 | 只接受当前 Toy 的相对路径 |
| `getQrCode` | 支持 | 支持 | `path` 相对 Toy，`size` 80–1024 整数 |
| `closeBrowser` | 支持 | 不支持 | 不能把关闭失败当错误页面 |
| `getUserProfile` | 支持 | 支持（环境限制） | 用户确认；`toyOpenId` 不是凭证 |
| `getAuthorProfile` | 支持 | 支持 | 返回 `status`，`status==='ok'` 再读 `data` |
| `getAuthorVideos` | 支持 | 支持 | 只能当前 Toy 作者相关视频，1–50 项 |
| `getAuthorRelation` | 支持 | 支持（端外手机有限制） | 返回 `status`，访客可能未授权 |
| `getVideoUserActions` | 支持 | 支持（端外手机有限制） | 批量参阅 `items[].status` |
| `getCloudStorage` / `setCloudStorage` / `removeCloudStorage` | 支持 | 支持 | 按用户+Toy 隔离，小文本存档；需登录 |
| `getSharedStorage` / `setSharedStorage` / `removeSharedStorage` | 支持 | 支持 | 跨获授权 Toy 共享，先在创作中心创建 storageId |
| `submitScore` / `getRankList` / `getMyRank` | 支持 | 支持 | 高分榜，非实时排队；游客仅可读榜 |
| `checkInputSafety` | 支持 | 支持 | 公开展示前检查；`unavailable` 视为未通过 |
| `requestCamera` / `requestMicrophone` | 支持 | 支持 | 用户手势+业务授权+系统授权，返回实时流 |
| `stopMedia` | 支持 | 支持 | 使用结束必须释放原 `MediaStream` |
| `onContainerChange` / `getContainerState` / `setContainerMode` | 支持 | 不支持 | 方向、沉浸、安全区，App 内才可调用 |

> 注意：即使“Web 支持”，在端外手机浏览器也可能返回 `unsupported` 并引导打开 B站 App。以上表格是作者提供文档的概括，不能替代真实设备验证。

## 3. 关键限制与示例

### 用户资料和安全

`await window.toy.getUserProfile()` 可返回 `avatar`、`nickname` 和可选 `toyOpenId`。首次需要用户确认时由平台弹固定授权窗口。SDK 不提供 B站 UID/MID、真实登录令牌或供自建 Go 服务端验签的证明：**不能将 `toyOpenId` 或昵称当作 API 鉴权**。当前项目继续使用独立账号、服务端 Session 和 Bearer token。用户头像可在明确授权后用于展示，避免记录 toyOpenId 到日志/埋点。

### 云存储及共享存档

- 单 Toy 默认 **最多 128 个 key-value**；key 匹配 `[A-Za-z0-9_-]{1,128}`，不能以 `__` 开头；每个 value UTF-8 ≤1024 字节。
- `getCloudStorage()` 不传 keys 读取当前用户的全部可见值；`setCloudStorage({ preferences: JSON.stringify(...) })` 一次批量写入；失败是 Promise reject。
- `getSharedStorage(storageId, keys?)` / `setSharedStorage(storageId, items)` / `removeSharedStorage(storageId, keys)`：需要在 B站创作中心创建的获授权 `st_...` 标识，跨 Toy 同玩家共享；**不是面向所有人的服务器数据库**。
- 不可放入：MIDI 二进制、完整录音、密码、token、房间实时状态、排行榜实时分数流。
- 所有用户共享 Toy 的频率配额；低频、批量操作，307044 时退避并告知用户，不要自动高频重试。

### 分享与图片

- `share({ path: 'index.html?room=lobby' })`：App 内用户主动点击才触发，路径相对 `/toy/<slug>/`，不可绝对 URL 或 `../` 跳出。
- `getQrCode({ path: 'index.html', size: 320 })`：端内端外均可用，二维码内容仍限制为当前 Toy 内页面。
- `saveImageToAlbum({base64Data})`：App 内支持，base64 字符长度 ≤5MB；建议 ≤2MB，大图优先 URL。浏览器需用户点击后的 `<a download>` 或 Blob URL。

### 容器/安全区

1. `isSupport('onContainerChange')`，若支持先注册监听，保存取消函数。
2. 在用户主动点击“横屏沉浸”时 `setContainerMode({orientation:'landscape',immersive:true})`；手机横屏同时需要沉浸。
3. **不要用 Promise resolve 判定生效**，等待下一次 `onContainerChange(state)` 确认。
4. 使用返回的 `state.viewport` 和 `state.safeArea` 布局，`changedFields` 只用于提示哪些字段变化；在 Web 端使用 CSS responsive + `env(safe-area-inset-*)` 兜底。
5. 组件销毁时调用取消函数；不要重复注册全局监听。

### 安全文本

```js
async function checkPublicTitle(title) {
  const api = window.toy
  if (!api || !await api.isSupport('checkInputSafety')) {
    throw new Error('当前无法进行公开内容检测')
  }
  const result = await api.checkInputSafety({text: title, scene: 'midi_title'})
  if (result.status !== 'ok' || !result.passed) {
    throw new Error(result.status === 'rejected' ? '内容未通过检测' : '暂时无法验证，请稍后再试')
  }
}
```

限制：`text` 非空、UTF-8 ≤16384 字节，`scene` 只能 `[A-Za-z0-9_.:-]` 且 ≤64 字节。只在提交时检测，不能每次输入都调用。Toy 检测只能做**前置防护**：Go API 仍需审查公开 MIDI 标题/描述/标签、举报处理与管理员审核。

### 排行榜

- `submitScore({ board, score })` 必须登录且提交的是**绝对成绩**，服务器仅保留同榜高分，不累加。范围 `-16777216..16777215` 整数。
- `getRankList({ board, period: 'all'|'month'|'week'|'day', limit })` 游客可读；`getMyRank` 需登录，判断 `ranked` 不看 `score` 是否为 0。
- **冲突须核实**：作者提供的旧类型声明写 `board` 仅 1–3，新能力清单写 1–5。项目声明标注此差异；未在平台证实前业务端只使用 1–3，且首版并不依赖 Toy 排行榜。
- 排行榜不能用于琴房排队、计时、实时同步。

### 媒体输入

- `requestCamera({ facingMode:'user'|'environment' })`、`requestMicrophone()` 只能用户手势触发，返回**实时 `MediaStream` 而不是文件**。麦克风录制用 `MediaRecorder`；分析音量用 Web Audio `AnalyserNode`。
- SDK 授权 + 系统权限均可能拒绝，处理 `BusinessDenied`、`NotAllowedError`、`NotFoundError` 等。
- 结束使用必须 `await window.toy.stopMedia(stream)`。琴房当前只同步 MIDI 事件，**没有必要申请麦克风或摄像头**。

## 4. 类型和文档差异说明

1. **原始 .d.ts** 没有声明 `getSharedStorage` 系列，但更新能力清单补充了这 3 个方法。已在 `web-toy/types/toy.d.ts` 中合并，运行时仍 `isSupport` 判断。
2. **排行榜 board** 存在 1–3 与 1–5 的冲突，类型中标了注释，不应推断服务端可用上限。
3. `window.toy` 在作者原始声明中为必填；项目为了支持本地 Vite / SDK 载入失败，在类型中改为 **可选**。
4. 原始声明只对 `onContainerChange` 给出同步取消函数，其余方法 Promise；项目保持一致。
5. 这套 SDK 不提供外部 WSS/HTTPS 的连接授权。Toy 对第三方服务的 CSP/域名白名单及 iOS/Android WebView 兼容性**仍需要平台与真机验收**。

## 5. 本地 Codex 验收清单

- [ ] 不在没有 SDK 的普通浏览器中崩溃；本地钢琴仍可用。
- [ ] 用户触发分享/横屏/媒体权限，并处理拒绝和端外不支持。
- [ ] 所有云存储读写限额、字节长度、失败与限流被正确处理。
- [ ] `checkInputSafety.status !== 'ok'` 时不公开发布文本。
- [ ] 不把 `toyOpenId` 当服务端身份；不会在 URL/日志暴露 token。
- [ ] Android / iOS B站 App 与桌面 Web 分别做真实联机及声音测试，并记录设备及结果。
- [ ] `cd web-toy && npm run test:timing && npm run build` 能通过；所有不通过都提供具体日志。
