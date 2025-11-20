# API 说明

本文档列出当前服务可供前端/第三方调用的主要接口。所有带 `Authorization` 标记的接口均需在请求头中携带 `Authorization: Bearer <access_token>`，令牌来自 Google 登录回调返回的 `token.access_token`。

## 1. Google OAuth

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/auth/google/login` | 生成 state 并重定向至 Google 登录页 |
| `GET` | `/api/auth/google/callback` | Google 回调入口。成功后返回 JSON 或（若配置 `FRONTEND_REDIRECT_URL`）携带 `payload` 重定向到前端 |

回调 JSON 示例（`payload` 解码后的结构相同）：

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

## 2. 用户信息

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/auth/me` | **需 Authorization**。返回当前用户信息与 token 到期时间 |

响应示例：

```json
{
  "user": {
    "email": "demo@google.com",
    "name": "Demo User",
    "avatar_url": "https://...",
    "level": 2
  },
  "token_expires_at": "2024-08-02T12:00:00Z"
}
```

## 3. API 密钥管理

所有接口均需 Bearer Token。密钥字符串格式为 `KF-<level>-<uuid>`，其中 `<level>` 为生成时用户的等级快照。`last_used_at` 用于记录业务方调用时的时间戳，可通过更新接口置为当前时间。

### 3.1 列出密钥

- **方法**：`GET`
- **路径**：`/api/api-keys`
- **参数**：无
- **响应**：

```json
{
  "keys": [
    {
      "id": 12,
      "label": "server-1",
      "key": "KF-1-5f90e057-9d3c-4aa6-88db-0a7c729c22f9",
      "level_snapshot": 1,
      "created_at": "2024-08-01T12:00:00Z",
      "last_used_at": null
    }
  ]
}
```

### 3.2 创建密钥

- **方法**：`POST`
- **路径**：`/api/api-keys`
- **请求体**：

```json
{
  "label": "server-1" // 可选，默认 "default"
}
```

- **响应**：`201 Created`，结构与列表单项一致。

### 3.3 更新密钥

- **方法**：`PUT`
- **路径**：`/api/api-keys/:id`
- **请求体**（至少提供一个字段）：

```json
{
  "label": "new-name",     // 可选，修改名称
  "regenerate": true,      // true 时按当前等级重新生成密钥
  "mark_used": true        // true 时将 last_used_at 更新为当前时间
}
```

- **响应**：`200 OK`，返回更新后的密钥。

### 3.4 删除密钥

- **方法**：`DELETE`
- **路径**：`/api/api-keys/:id`
- **响应**：`204 No Content`

## 4. 受保护示例

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/protected/ping` | **需 Authorization**。示例接口，用于校验 token 是否有效 |

返回：

```json
{
  "message": "认证通过"
}
```
