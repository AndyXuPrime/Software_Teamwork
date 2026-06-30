# AI Gateway 默认 Profile Seed 与 Token Hash 运行手册

本文档说明如何在本地或 CI 环境中初始化 AI Gateway 的服务访问凭证和三类默认 Model Profile（chat、embedding、rerank），以满足 QA、Knowledge、Document 等下游服务的联调需求。

---

## 1. 生成服务令牌哈希（X-Service-Token）

AI Gateway 使用 SHA-256 哈希验证入站请求的 `X-Service-Token` 头，**不存储明文令牌**。

### 1.1 生成令牌

选择一个随机令牌字符串（生产环境请用密码管理器生成，≥32 位随机字符）：

```bash
# 示例：本地 / CI 使用固定测试令牌（仅非生产环境）
SERVICE_TOKEN="local-dev-service-token-$(openssl rand -hex 16)"
echo "token: $SERVICE_TOKEN"
```

### 1.2 计算哈希并写入配置

```bash
TOKEN_HASH="sha256:$(echo -n "$SERVICE_TOKEN" | sha256sum | awk '{print $1}')"
echo "config value: $TOKEN_HASH"
```

将 `$TOKEN_HASH` 写入 AI Gateway 的环境变量 `AI_GATEWAY_SERVICE_TOKEN_HASHES`（或对应的 `.env` / Kubernetes Secret）。支持逗号分隔多个哈希，用于轮换令牌。

> **注意**：`sha256sum` 的 `-n` 参数确保不附带换行符；如果使用 `openssl`：
> ```bash
> TOKEN_HASH="sha256:$(echo -n "$SERVICE_TOKEN" | openssl dgst -sha256 | awk '{print $2}')"
> ```

---

## 2. 创建默认 Model Profile

所有请求均需携带：

```
X-Service-Token: <明文令牌>
X-Caller-Service: gateway          # 管理接口使用 gateway
Content-Type: application/json
```

将 `AI_GATEWAY_BASE_URL` 设置为 AI Gateway 服务地址。默认监听端口为 `8086`（由 `AI_GATEWAY_HTTP_ADDR` 控制），本地示例：

```bash
AI_GATEWAY_BASE_URL="http://localhost:8086"
```

### 2.1 默认 Chat Profile

```bash
curl -s -X POST "$AI_GATEWAY_BASE_URL/internal/v1/model-profiles" \
  -H "X-Service-Token: $SERVICE_TOKEN" \
  -H "X-Caller-Service: gateway" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "default-chat",
    "purpose": "chat",
    "provider": "siliconflow",
    "baseUrl": "https://api.siliconflow.cn/v1",
    "model": "Qwen/Qwen2.5-72B-Instruct",
    "apiKey": "'"$SILICONFLOW_API_KEY"'",
    "enabled": true,
    "isDefault": true,
    "supportsStreaming": true,
    "timeoutMs": 60000
  }' | jq .
```

### 2.2 默认 Embedding Profile

```bash
curl -s -X POST "$AI_GATEWAY_BASE_URL/internal/v1/model-profiles" \
  -H "X-Service-Token: $SERVICE_TOKEN" \
  -H "X-Caller-Service: gateway" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "default-embedding",
    "purpose": "embedding",
    "provider": "siliconflow",
    "baseUrl": "https://api.siliconflow.cn/v1",
    "model": "BAAI/bge-m3",
    "apiKey": "'"$SILICONFLOW_API_KEY"'",
    "enabled": true,
    "isDefault": true,
    "dimensions": 1024,
    "timeoutMs": 30000
  }' | jq .
```

### 2.3 默认 Rerank Profile

```bash
curl -s -X POST "$AI_GATEWAY_BASE_URL/internal/v1/model-profiles" \
  -H "X-Service-Token: $SERVICE_TOKEN" \
  -H "X-Caller-Service: gateway" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "default-rerank",
    "purpose": "rerank",
    "provider": "siliconflow",
    "baseUrl": "https://api.siliconflow.cn/v1",
    "model": "BAAI/bge-reranker-v2-m3",
    "apiKey": "'"$SILICONFLOW_API_KEY"'",
    "enabled": true,
    "isDefault": true,
    "topN": 5,
    "timeoutMs": 30000
  }' | jq .
```

---

## 3. 验证 Readiness

```bash
curl -s "$AI_GATEWAY_BASE_URL/readyz" | jq .
```

期望响应（所有三类 profile 就绪）：

```json
{
  "data": {
    "status": "ok",
    "checks": [
      { "name": "config_store", "status": "ok" },
      { "name": "chat_profile", "status": "ok" },
      { "name": "embedding_profile", "status": "ok" },
      { "name": "rerank_profile", "status": "ok" }
    ]
  },
  "requestId": "..."
}
```

如果任一 profile 缺失，`status` 为 `"degraded"`，对应的 `check.status` 为 `"missing"`。

---

## 4. 无真实密钥的 CI / 本地环境（fake provider）

CI 环境通常没有 SiliconFlow API Key。所有 provider smoke 测试均使用 `httptest.Server` 作为受控 fake provider，**无需真实密钥即可运行**：

```bash
cd services/ai-gateway
go test ./...
```

若需要对真实 provider 做手动 smoke 验证，直接使用第 6 节的 curl 示例即可。

---

## 5. 安全边界

| 禁止行为 | 原因 |
|----------|------|
| 将 `apiKey` 写入日志或响应 | API Key 仅加密存储，永远不出现在 API 响应中 |
| 在 `defaultParameters` 中包含 `api_key`、`token`、`secret` 等敏感键 | 服务层会拒绝含敏感键的 JSON 对象 |
| 将 `baseUrl` 设置为包含凭证或敏感 query 参数的 URL | 服务层校验会拒绝此类 URL |
| 将 provider 原始错误体转发给调用方 | 所有 provider 错误均经过归一化，不透传原始响应 |

---

## 6. 下游服务接入示例

QA / Knowledge / Document 服务接入 AI Gateway 时，在请求头中携带：

```
X-Service-Token: <服务令牌明文>
X-Caller-Service: qa            # 或 knowledge / document
X-Request-Id: <上游请求 ID>     # 可选，用于全链路追踪
```

Chat 请求示例（不指定 profile，使用默认 chat profile）：

```bash
curl -s -X POST "$AI_GATEWAY_BASE_URL/internal/v1/chat/completions" \
  -H "X-Service-Token: $SERVICE_TOKEN" \
  -H "X-Caller-Service: qa" \
  -H "Content-Type: application/json" \
  -d '{"model":"Qwen/Qwen2.5-72B-Instruct","messages":[{"role":"user","content":"你好"}]}'
```

Embedding 请求示例：

```bash
curl -s -X POST "$AI_GATEWAY_BASE_URL/internal/v1/embeddings" \
  -H "X-Service-Token: $SERVICE_TOKEN" \
  -H "X-Caller-Service: knowledge" \
  -H "Content-Type: application/json" \
  -d '{"model":"BAAI/bge-m3","input":["文档片段内容"]}'
```

Rerank 请求示例：

```bash
curl -s -X POST "$AI_GATEWAY_BASE_URL/internal/v1/rerankings" \
  -H "X-Service-Token: $SERVICE_TOKEN" \
  -H "X-Caller-Service: knowledge" \
  -H "Content-Type: application/json" \
  -d '{"model":"BAAI/bge-reranker-v2-m3","query":"用户问题","documents":[{"id":"chunk-1","text":"文档内容"}]}'
```
