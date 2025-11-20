# Repository Guidelines

## 项目结构与模块组织
主入口 `main.go` 负责加载 `config`、初始化 GORM、注册 `routes` 并暴露 HTTP 服务。`config/` 解析 `.env`，`controllers/` 聚合请求处理，`services/` 封装 OAuth 与业务逻辑，`models/` 定义数据结构及 `migration.go` 的 Up/Down 逻辑，`routes/` 将 Gin 分组到 `/healthz` 与 `/api/auth`。公共配置样例位于 `.env.example`，Docker 入口在 `Dockerfile`。

## 构建、测试与开发命令
```bash
cp .env.example .env        # 初始化环境变量
go mod tidy                 # 在可联网环境补齐依赖
go run ./...                # 本地启动并自动执行迁移
go test ./...               # 运行全部 Go 单测
docker build -t developer-platform-backend .   # 容器构建
docker run --env-file .env -p 8080:8080 developer-platform-backend
```
开发调试可配合 `air` 或 `reflex` 监听 `controllers`、`services`、`routes` 目录。

## 代码风格与命名约定
保持 `gofmt -w .` 与 `goimports` 结果，必要时运行 `golangci-lint run`。导出类型、函数使用 `PascalCase`，本地变量与接收者使用 `camelCase`，常量与环境变量使用 `SCREAMING_SNAKE_CASE`。Gin 路由分组按资源命名 (`auth`, `healthz`)，迁移版本号需递增并在 `models/migration.go` 内集中维护。

## 测试规范
尚未有 `_test.go`，新增功能时使用 Go 的 testing 包，并将文件放在与被测模块相同的目录中，例如 `services/google_service_test.go`。命名遵循 `TestFunction_Scenario`，覆盖所有 GORM 交互及 OAuth 分支。CI 或手动提交前应运行 `go test ./...` 并附带伪造配置，以确保迁移不会对共享数据库造成副作用。

## 提交与 Pull Request
遵循类 Conventional Commits，例如 `feat: add google callback logging`、`fix: rollback migration on failure`。每个 PR 须包含：变更概要、关联 issue 链接、必要时的 API 响应示例（JSON）或终端截图、迁移说明以及手动验证步骤（含 `go test ./...` 和可复现的 `curl` 请求）。禁止提交 `.env`、本地 SQLite 或其它敏感文件，公共配置请放入 `.env.example` 并更新 README。

## 安全与配置提示
OAuth 与数据库凭证仅放 `.env`，部署时通过容器的 `--env-file` 或 CI secret 注入。若启用 Cookie 登录，生产必须设置 `GIN_MODE=release`、可信域名、HTTPS 反向代理，并在 `config` 中启用 `SESSION_STATE_NAME` 以抵御 CSRF。数据库迁移建议在 staging 先行验证 `models.RunMigrations` 与 `models.RollbackMigration`，再在生产执行。
