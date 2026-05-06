# AI 长期记忆实施方案、影响清单与 Qdrant 部署步骤

## 1. 目标
- 在不影响现有聊天主流程响应的前提下，新增“长期记忆储备能力”。
- 先完成写入链路：消息记录 -> 批量总结 -> 向量化 -> Qdrant 入库。
- 记忆主体覆盖两类：`用户(user)`、`群(group)`。

## 2. 实施方案（具体到模块）

### 2.1 总体设计（两层）
- `L1 原始记录层`（必做）：
  - 每条消息先落轻量原始记录（可存 MySQL 新表）。
  - 不调用 AI，总是成功，便于回溯与重放。
- `L2 语义记忆层`（必做）：
  - 按窗口批量触发 AI 总结，提取长期价值信息。
  - 通过 embedding 向量化后写入 Qdrant。

说明：L1 保底，L2 增强；即使 L2 失败也不影响主聊天能力。

### 2.2 代码改造点（建议）

#### A. 配置层（拆分多个 `.local` 文件）
- 配置文件拆分建议（都带 `.local`）：
  - `conf/conf.local.json`：保留现有基础配置（端口、数据库、机器人等）。
  - `conf/memory.local.json`：只放记忆策略与批处理参数。
  - `conf/qdrant.local.json`：只放向量库连接与集合参数。
- 代码结构建议：
  - 在 `conf/config.go` 中拆成三个结构体：`BaseConfig`、`MemoryConfig`、`QdrantConfig`。
  - 暴露聚合对象：`AppConfig{Base, Memory, Qdrant}`。
  - 启动时按固定顺序加载三个文件并合并，任何一个缺失都直接报错退出。
- 字段归属建议：
  - `MemoryConfig`：
    - `MemoryEnabled bool`
    - `MemoryRawStoreEnabled bool`
    - `MemoryBatchEnabled bool`
    - `MemoryBatchMaxTurns int`
    - `MemoryBatchMaxChars int`
    - `MemoryBatchMaxWaitSec int`
    - `MemoryAsyncQueueSize int`
    - `MemoryWorkerCount int`
    - `MemoryRetryTimes int`
    - `MemoryMinImportance int`
    - `MemoryDedupWindowSec int`
    - `MemoryFallbackToMysql bool`
    - `EmbeddingProvider string`
    - `EmbeddingApiKey string`
    - `EmbeddingModel string`
    - `EmbeddingApiUrl string`
  - `QdrantConfig`：
    - `QdrantUrl string`
    - `QdrantApiKey string`
    - `QdrantCollectionUser string`
    - `QdrantCollectionGroup string`
    - `QdrantVectorSize int`
    - `QdrantDistance string`
    - `QdrantTimeoutMs int`

#### B. 数据层（新增 model）
- 新增 `model/memory_raw.go`：
  - 表 `memory_raw_turns`（建议字段）：
    - `id`, `scope`, `user_id`, `group_id`, `session_id`, `message_id`
    - `source`, `input_text`, `reply_text`, `created_at`
    - `status`（pending/summarized/failed）
- 新增 `model/memory_job.go`：
  - 批处理任务状态管理，支持重试与幂等。

#### C. 业务层（新增 `function/memory/`）
- `types.go`：定义 `MemoryTask`, `MemorySummary`, `MemoryPoint`。
- `collector.go`：采集回合素材（输入/输出）。
- `batcher.go`：按阈值聚合（回合数/字数/时间）。
- `interpreter.go`：调用 AI 生成结构化总结。
- `embedder.go`：对总结文本生成向量。
- `qdrant_repo.go`：Qdrant upsert/search/delete。
- `dedup.go`：内容归一化 + hash 去重。
- `worker.go`：异步队列、重试、失败降级。

#### D. 挂载点改造（现有文件）
- `event/message/message.go`
  - `private(msg)` 末尾调用 `memory.CollectInput(...)`。
  - `group(msg)` 末尾调用 `memory.CollectInput(...)`。
- `function/chatGPT/chatGPT.go`
  - `AddPlan` / `AddPlanPrivate` 成功发送后调用 `memory.CollectOutput(...)`。
  - 将输入输出通过 `session_id + message_id` 关联。

#### E. 启动初始化（建议）
- 在 `main.go` 或 `init` 流程增加：
  - `memory.Init(config)`（初始化队列、worker、qdrant client）
  - `memory.Start()`（后台批处理任务）

### 2.3 批量总结策略（你提出的方案落地）
- 触发条件（满足任一触发）：
  - `turn_count >= MemoryBatchMaxTurns`
  - `char_count >= MemoryBatchMaxChars`
  - `now - first_turn_time >= MemoryBatchMaxWaitSec`
- 聚合维度：
  - 私聊：`scope=user, user_id`
  - 群聊：`scope=group, group_id`（并记录主要发言 user_id 列表）
- 总结输出建议 JSON：
  - `facts[]`, `preferences[]`, `rules[]`, `goals[]`
  - `importance`（1-5）
  - `confidence`（0-1）
- 入库门槛：
  - `importance >= MemoryMinImportance`
  - `confidence >= 0.65`（建议默认）

## 3. 影响清单（改造评估）

### 3.1 功能影响
- 新增记忆写入链路，不改变原有命令与回复行为。
- 群聊与私聊都将生成可检索长期记忆资产。

### 3.2 性能影响
- 正向影响：主链路保持同步响应，记忆流程走异步。
- 负向风险：批处理高峰可能造成队列积压。
- 应对：
  - `MemoryAsyncQueueSize` 限制；
  - 低优先级任务可丢弃；
  - worker 水平扩展。

### 3.3 稳定性影响
- 风险：Qdrant/Embedding 服务故障导致写入失败。
- 应对：
  - 重试 + 指数退避；
  - `MemoryFallbackToMysql=true`；
  - 不阻塞聊天回复。

### 3.4 数据与合规影响
- 新增长期存储，需明确脱敏策略和删除策略。
- 建议：
  - 对手机号/邮箱等敏感字段脱敏；
  - 提供按 `user_id/group_id` 清理接口；
  - 记录审计日志（谁写入/删除了什么）。

### 3.5 运维影响
- 新增 Qdrant 服务和备份需求。
- 新增监控指标：
  - `memory_queue_len`
  - `memory_batch_success_total`
  - `memory_batch_fail_total`
  - `qdrant_upsert_latency_ms`
  - `embedding_latency_ms`

### 3.6 测试影响
- 必测用例：
  - 单条消息写入 L1 成功；
  - 达到阈值触发批量总结；
  - Qdrant 不可用时自动降级；
  - 去重生效（重复内容仅更新热度）。

## 4. 配置模板（拆分版）

### 4.1 `conf/conf.local.json`（基础配置，沿用现有）
```json
{
  "name": "300Bot",
  "port": "9999",
  "apiPort": "6097",
  "apiUrl": "127.0.0.1",
  "botQQ": "2466001518",
  "botName": "叁柏",
  "manager": "675559614",
  "databaseHost": "ip:3306",
  "databaseUser": "root",
  "databasePassword": "root",
  "botDatabaseName": "chat_bot",
  "heroDatabaseName": "300heros",
  "immortalbaseName": "luma_immortal",
  "chatGPTkey": "",
  "wetherApiCode": "",
  "moneyList": []
}
```

### 4.2 `conf/memory.local.json`（记忆与批处理）
```json
{
  "memoryEnabled": true,
  "memoryRawStoreEnabled": true,
  "memoryBatchEnabled": true,
  "memoryBatchMaxTurns": 24,
  "memoryBatchMaxChars": 4000,
  "memoryBatchMaxWaitSec": 900,
  "memoryAsyncQueueSize": 2000,
  "memoryWorkerCount": 4,
  "memoryRetryTimes": 3,
  "memoryMinImportance": 3,
  "memoryDedupWindowSec": 86400,
  "memoryFallbackToMysql": true,
  "embeddingProvider": "openai_compatible",
  "embeddingApiKey": "",
  "embeddingModel": "text-embedding-3-small",
  "embeddingApiUrl": "https://api.openai.com/v1/embeddings"
}
```

### 4.3 `conf/qdrant.local.json`（向量库）
```json
{
  "qdrantUrl": "http://127.0.0.1:6333",
  "qdrantApiKey": "",
  "qdrantCollectionUser": "mem_user_v1",
  "qdrantCollectionGroup": "mem_group_v1",
  "qdrantVectorSize": 1536,
  "qdrantDistance": "Cosine",
  "qdrantTimeoutMs": 3000
}
```

### 4.4 加载与校验规则（建议）
- 所有 `.local` 文件必须存在，否则启动失败。
- 程序启动时做强校验：
  - `qdrantVectorSize` 必须与 `embeddingModel` 维度一致。
  - `memoryBatchMaxTurns`、`memoryBatchMaxChars`、`memoryBatchMaxWaitSec` 均需 > 0。
  - `memoryWorkerCount` 不建议超过 CPU 核心数的 2 倍。

## 5. Qdrant 部署步骤

### 5.1 方案一：Docker（推荐）

#### Linux / Windows Docker Desktop
```bash
docker run -d \
  --name qdrant \
  -p 6333:6333 \
  -p 6334:6334 \
  -v qdrant_storage:/qdrant/storage \
  qdrant/qdrant:latest
```

健康检查：
```bash
curl http://127.0.0.1:6333/
```

预期返回包含服务信息的 JSON。

### 5.2 方案二：docker-compose（生产更推荐）
```yaml
version: "3.8"
services:
  qdrant:
    image: qdrant/qdrant:latest
    container_name: qdrant
    restart: always
    ports:
      - "6333:6333"
      - "6334:6334"
    volumes:
      - ./qdrant_storage:/qdrant/storage
```

启动：
```bash
docker compose up -d
```

### 5.3 初始化 Collection（示例）

用户记忆库：
```bash
curl -X PUT "http://127.0.0.1:6333/collections/mem_user_v1" \
  -H "Content-Type: application/json" \
  -d '{
    "vectors": {
      "size": 1536,
      "distance": "Cosine"
    }
  }'
```

群记忆库：
```bash
curl -X PUT "http://127.0.0.1:6333/collections/mem_group_v1" \
  -H "Content-Type: application/json" \
  -d '{
    "vectors": {
      "size": 1536,
      "distance": "Cosine"
    }
  }'
```

查看集合：
```bash
curl "http://127.0.0.1:6333/collections"
```

### 5.4 可选安全配置（生产）
- 开启 API Key（通过环境变量配置 Qdrant）。
- 将 6333/6334 仅暴露到内网。
- 加反向代理和 TLS（如 Nginx）。
- 定期备份 `storage` 卷目录。

## 6. 上线与回滚建议

### 6.1 上线顺序
- 步骤 1：先部署 Qdrant，完成健康检查。
- 步骤 2：发布代码但先关闭 `memoryEnabled`（灰度）。
- 步骤 3：开启 `memoryRawStoreEnabled`，观察 24h。
- 步骤 4：开启 `memoryBatchEnabled`，逐步提高 worker 数量。
- 步骤 5：验证稳定后，进入读取拼接阶段开发。

### 6.2 回滚策略
- 业务回滚：关闭 `memoryEnabled` 即可停用记忆链路。
- 数据回滚：保留 L1 原始记录，允许后续重新批处理补偿。
- 服务回滚：Qdrant 故障不影响主回复流程。

## 7. 本阶段交付标准（Done Definition）
- 已完成配置项接入与读取。
- 已完成 L1 原始记录写入。
- 已完成按阈值批量总结与 Qdrant upsert。
- 已完成失败重试和降级策略。
- 已具备基础监控与日志定位能力。
