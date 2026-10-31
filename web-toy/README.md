# PP 钢琴键盘 · Toy

Vue 3 / Vite 网页端，以桌面 App 为视觉基准。琴键、瀑布、播放/跟弹控件、时间轴、曲库、录制面板、快捷设置、主题、按键方案编辑器、工坊与通知直接复用 `../frontend/src` 和 `../shared` 中的组件与样式。浏览器音频、本地存储和 MIDI 权限由网页服务层处理，不加载 Wails 运行时。

开发和生产构建前会从桌面端同步图标、琴键与和弦数据，因此需要保留完整仓库目录；Docker 构建也包含这些源码。

```bash
npm ci
npm run dev
npm run test:timing
npm run build
```

默认地址 http://127.0.0.1:5173，社区 API 默认为 http://127.0.0.1:8080。生产构建使用 `VITE_API_BASE_URL=https://你的接口域名`，Toy 资源使用相对路径。

使用本系统的账号密码。昵称和头像由用户设置，不调用 Toy 用户资料 API；Toy 登录是桌面的可选网页登录确认流程。

公开工坊、我的上传、我的收藏支持搜索、标签/难度筛选、排序分页、详情、试听/跟弹、上传编辑删除、下载、赞踩收藏和举报。管理员可审核作品、下架、恢复及处理举报；私密/待审文件只供有权限的账号访问。

本地曲目与草稿保存在 IndexedDB，社区原文件使用独立缓存库。Toy 云存储仅保存少量偏好，不保存账号 token、MIDI 或头像。

完整启动配置和 API 见 [实现说明](../community-docs/IMPLEMENTATION.md)，本轮真实验证见 [验证记录](../community-docs/VALIDATION.md)。真实 B站 Android/iOS 容器和外部 HTTPS/WSS 仍需实测。
