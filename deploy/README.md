# 社区服务部署配置

当前交付只完成本机联调，没有部署到公网。启动、接口和验证说明见 `../community-docs/IMPLEMENTATION.md`。

- API 单实例运行，使用 PostgreSQL；通过环境变量或工作目录的 `.env` 配置。
- API 和 Toy 分别设置 HTTPS / WSS 域名、准确的 `CORS_ORIGINS` 与 `TOY_URL`。
- Web 构建时指定 `VITE_API_BASE_URL`；桌面构建可通过 `-ldflags '-X main/service.CommunityAPIURL=https://api.example.com'` 配置，也可在运行时设置 `PP_COMMUNITY_API_URL`。
- 管理员由 `ADMIN_USERNAME` / `ADMIN_PASSWORD` 初次创建，不会自动提升已有普通账号。
- 持久保存并备份 PostgreSQL 及 `DATA_DIR`（MIDI、头像）；不要将数据目录作为静态文件目录公开。
- Dockerfile 提供 `toy-dist`、`api-binary`、`server` 三个目标；本机未运行 Docker 构建。

```bash
docker build --target server -t pp-piano-api -f deploy/Dockerfile .
docker run --env-file /secure/path/server.env -p 127.0.0.1:8080:8080 -v /secure/piano-data:/data pp-piano-api
```

真实 B站 Toy 的域名白名单、SDK、Web Audio 和触摸表现需要在实际容器中验收。
