# Developer Platform Backend

基于 Go + Gin 的极简后台，实现 Google 登录、JWT 会话与用户信息同步，使用 GORM + PostgreSQL 并带有可回滚的迁移机制。

## 功能亮点

- `.env` 配置，启动顺序为：加载配置 → 初始化数据库 → `models.RunMigrations()`
- GORM + `gorm.io/driver/postgres`，自定义迁移表，支持版本化 Up/Down
- Google OAuth 登录，回调后可获取头像、邮箱与昵称并完成 upsert
- Dockerfile 多阶段构建，可直接用于容器部署

## 快速开始

```bash
cp .env.example .env
# 根据实际环境填写数据库与 Google OAuth 相关变量

go run ./...
```

> 当前环境因网络受限未能提前下载 Go 依赖，请在可访问代理的环境下执行 `go mod tidy`。

### 环境变量说明（节选）

| 变量 | 说明 |
| --- | --- |
| `SERVER_PORT` | HTTP 监听端口，默认 `8080` |
| `DATABASE_URL` | 可选，若为空将由 `DB_HOST` 等字段拼接 |
| `GOOGLE_CLIENT_ID/SECRET` | Google OAuth 凭证 |
| `GOOGLE_REDIRECT_URL` | Google 回调地址，需与控制台一致 |
| `SESSION_STATE_NAME` | 存放 state 的 cookie 名称 |
| `JWT_SECRET` | 签发访问令牌所用密钥 |
| `JWT_EXPIRES_IN` | 令牌有效期，Go `time.ParseDuration` 格式，如 `24h` |
| `FRONTEND_REDIRECT_URL` | 登录成功后重定向到的前端地址，例如 `https://app.example.com/auth/success` |

更多字段可参考 `.env.example`。

## API

- `GET /healthz` 服务健康检查（公开）
- `GET /api/auth/google/login` 重定向到 Google 登录（公开）
- `GET /api/auth/google/callback` 处理回调，返回用户信息 + JWT（公开，Google 回调用）。若配置 `FRONTEND_REDIRECT_URL`，会带着 `payload=<Base64(JSON)>` 重定向到指定地址。
- `GET /api/auth/me` 查询当前登录用户（需要 `Authorization: Bearer <token>`）
- `GET /api/protected/ping` 受保护示例接口（需要 `Authorization: Bearer <token>`）
- `GET /api/api-keys` 列出当前用户的 API 密钥（需要 `Authorization: Bearer <token>`）
- `POST /api/api-keys` 创建密钥，密钥字符串会包含当前等级信息（需要 `Authorization: Bearer <token>`）
- `PUT /api/api-keys/:id` 更新密钥（重命名、重新生成、标记上次使用时间，需 `Authorization`）
- `DELETE /api/api-keys/:id` 删除密钥（需 `Authorization`）

> 更完整的字段与示例请参考 `docs/api.md`。
回调成功响应示例：

```json
{
  "token": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 86400
  },
  "user": {
    "email": "demo@google.com",
    "name": "Demo User",
    "avatar_url": "https://...",
    "level": 1
  },
  "synced_at": "2024-07-31T12:00:00Z"
}
```

受保护接口调用示例：

```bash
curl http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer <access_token>"
```

### 前端如何消费 `payload`

Google 登录成功后，后端会重定向至 `FRONTEND_REDIRECT_URL`，并在查询参数中附带 `payload`。该字段使用 URL 安全的 Base64 编码，内部 JSON 结构与上述回调示例一致。前端伪代码：

1. 从 `window.location.search` 读取 `payload`；
2. `JSON.parse(atob(payload))` 解码；
3. 将 `token.access_token`/`user` 写入前端状态，然后重定向到业务页面。

如需保持旧行为（直接在回调接口返回 JSON 而不跳转），可将 `FRONTEND_REDIRECT_URL` 留空，后端将回退为原始 JSON 响应。

### API Key 管理

- 创建密钥：`POST /api/api-keys`，请求体示例 `{"label": "server-1"}`。响应：

```json
{
  "key": {
    "id": 12,
    "label": "server-1",
    "key": "KF-1-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "level_snapshot": 1,
    "created_at": "2024-08-01T12:00:00Z",
    "last_used_at": null
  }
}
```

- 更新密钥：`PUT /api/api-keys/:id`，允许以下字段任选其一或组合：
  - `label`：字符串，修改名称；
  - `regenerate`：布尔值，若为 `true` 则依据当前等级重新生成密钥，`level_snapshot` 会同步；
  - `mark_used`：布尔值，若为 `true` 将 `last_used_at` 更新为当前时间。
- 删除：`DELETE /api/api-keys/:id`，无响应体。

后台会通过 JWT 中的用户身份校验，并自动将用户当前等级拼入密钥字符串（形式如 `KF-<level>-<uuid>`）。`last_used_at` 字段可在调用方在实际使用密钥后通过更新接口置为当前时间，用于审计或展示。

## Docker 构建

```bash
docker build -t developer-platform-backend .
docker run --env-file .env -p 8080:8080 developer-platform-backend
```

## 迁移管理

- 所有迁移定义在 `models/migration.go`
- `models.RunMigrations(db)`：应用未执行的版本
- `models.RollbackMigration(db, version)`：回滚指定版本
- `models.GetMigrationStatus(db)`：查看执行状态

添加字段或表时：

1. 更新对应 `models/*.go` 中的结构体
2. 在 `migrations` 切片新增更高版本的 `MigrationItem`
3. 在 `Up` 中使用 `db.AutoMigrate()` 或 `db.Migrator()` 操作，`Down` 写回滚逻辑
4. 重启服务会自动应用最新迁移

## 开发建议

- 建议配合 `air`、`reflex` 等工具实现热重载
- 生产部署时将 `GIN_MODE=release`，并配置 HTTPS 及可信任域名的 Cookie
