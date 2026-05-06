# AI 长期记忆（向量数据库）建设建议与可行性分析

## 1. 目标与范围
- 目标：为机器人建立“长期记忆”能力，支持对“发言人”和“群聊”两类主体持续沉淀可检索记忆。
- 范围：先做“写入与储备”，在消息处理完成后追加“会话解读 + 记忆入库”；后续再接“读取与拼接”。
- 约束：尽量不影响当前消息主链路的实时响应，优先异步化。

## 2. 当前代码链路与挂载点

### 2.1 现有链路（简化）
```text
WebSocket上报 -> CheckWsMsg(post_type分发)
 -> message.CheckType
   -> private(msg) / group(msg)
      -> chatGPT.AddPlanPrivate / chatGPT.AddPlan 或其他业务分支
         -> AskForChatGPT
         -> send.SendPrivatePost / send.SendGroupPost
```

### 2.2 最佳挂载点（建议）
- 挂载点 A：`event/message/message.go` 的 `private(msg)`、`group(msg)` 末尾。
  - 用途：记录原始会话素材（用户输入、场景信息、命令路径）。
  - 优势：覆盖所有消息，不依赖是否命中 ChatGPT。
- 挂载点 B：`function/chatGPT/chatGPT.go` 的 `AddPlan`、`AddPlanPrivate` 中拿到模型回复后。
  - 用途：记录“问-答对”、对话结果和 token 信息。
  - 优势：能形成高价值“事实 + 偏好 +上下文结果”记忆。

建议做法：A + B 双写入，A 记录输入侧，B 记录输出侧，并通过 `session_id/message_id` 关联。

## 3. 向量数据库选型建议

### 3.1 选型结论
- 第一阶段推荐：`Qdrant`（独立服务，HTTP/gRPC 接入，运维和开发成本低，适合快速上线）。
- 备选：
  - `pgvector`：若团队已有 PostgreSQL 体系可选；当前项目主库是 MySQL，迁移成本更高。
  - `Milvus`：能力强但部署复杂，适合数据规模和检索复杂度明显上升后再升级。

### 3.2 结合当前项目的可行性
- 当前项目是 Go 单体服务，已存在较多外部 HTTP 调用模式，接入 Qdrant 风险低。
- 当前聊天并发控制偏串行（ChatGPT 队列容量 1），新增异步记忆写入不会放大并发复杂度。
- 现有配置读取集中在 `conf/config.go`，可平滑增加向量库配置项与开关项。

可行性结论：高可行，建议分阶段上线，先“稳定写入”，再做“召回拼接”。

## 4. 记忆模型设计（先储备）

### 4.1 记忆实体
- `UserMemory`（用户长期记忆）
  - 主体键：`user_id`
  - 场景键：`group_id`（可选，表示该记忆来源群）
  - 内容：偏好、身份信息、长期目标、稳定习惯、显式要求
- `GroupMemory`（群长期记忆）
  - 主体键：`group_id`
  - 内容：群规则、常见话题、群内约定、共享上下文

### 4.2 记忆分层（建议）
- `raw_turn`：原始回合（输入/输出），低权重，保留短期溯源。
- `episodic`：事件级摘要（一次或多次会话归纳）。
- `semantic`：稳定事实/偏好（高价值长期记忆）。

先做 `raw_turn + semantic` 即可，`episodic` 可作为第二阶段补充。

### 4.3 元数据字段（建议最小集）
- `memory_id`
- `scope`：`user` / `group`
- `user_id`、`group_id`
- `session_id`、`message_id`
- `source`：`private` / `group`
- `text`：用于向量化的文本
- `summary`：结构化摘要（可空）
- `tags`：如 `preference`, `fact`, `rule`, `profile`
- `importance`：1-5
- `created_at`、`last_access_at`
- `ttl`（可选）

## 5. 会话解读与写入流程设计

### 5.1 处理流程（写入阶段）
```mermaid
flowchart TD
  A[消息处理完成] --> B[构建记忆任务 MemoryTask]
  B --> C[异步队列 memoryWorker]
  C --> D[会话解读(提取偏好/事实/规则)]
  D --> E[Embedding 向量化]
  E --> F[向量库Upsert]
  F --> G[写入关系型日志(可选)]
```

### 5.2 会话解读策略（建议）
- 仅抽取“长期有价值内容”，过滤无意义闲聊。
- 优先提取：
  - 用户：称呼、偏好、禁忌、长期计划、稳定背景信息。
  - 群聊：群规、公共约定、反复出现的话题结论。
- 每条候选记忆给出 `importance`，低分可跳过写入。

### 5.3 幂等与去重
- 去重键建议：`hash(scope + user_id/group_id + normalized_text)`。
- 时间窗口去重：例如 24 小时内相似内容仅更新热度，不重复入库。
- 异步失败重试：指数退避 + 最大重试次数，避免阻塞主流程。

## 6. 工程落地方案（按阶段）

### Phase 1：储备能力（你当前要做的）
- 新增 `memory` 模块（建议目录）：
  - `function/memory/collector.go`：收集会话素材
  - `function/memory/interpreter.go`：会话解读
  - `function/memory/embedder.go`：向量化
  - `function/memory/repository.go`：向量库写入
  - `function/memory/worker.go`：异步队列与重试
- 在 `private/group` 与 `AddPlan/AddPlanPrivate` 增加异步 `EnqueueMemoryTask(...)`。
- 先只实现写入，不改现有回复逻辑。

### Phase 2：读取与拼接（后续）
- 新增 `RetrieveMemories(user_id, group_id, query)`：
  - TopK 召回用户记忆 + 群记忆；
  - 重排后拼接到系统提示词或上下文前缀。
- 增加预算控制：
  - 限制注入条数与总 token；
  - 优先高 `importance` + 新近相关条目。

### Phase 3：治理与评估
- 记忆质量评估：
  - 命中率、误召回率、用户反馈。
- 记忆生命周期：
  - 过期策略、冲突覆盖、人工清理接口。

## 7. 配置建议
- 在配置中新增：
  - `memoryEnabled`：总开关
  - `memoryAsyncQueueSize`：异步队列长度
  - `embeddingProvider`、`embeddingApiKey`、`embeddingModel`
  - `vectorDbType`（qdrant/milvus/...）
  - `vectorDbUrl`、`vectorDbApiKey`、`vectorCollection`
  - `memoryMinImportance`：最小入库阈值
- 建议提供降级策略：
  - 向量库不可用时只记关系型日志（或落本地队列）；
  - 主聊天流程不因记忆失败而失败。

## 8. 风险与应对
- 风险：写入延迟导致队列积压。
  - 应对：异步 worker + 限流 + 丢弃低优先级任务。
- 风险：低质量记忆污染。
  - 应对：解读规则 + importance 阈值 + 去重策略。
- 风险：隐私与合规。
  - 应对：敏感信息脱敏、可删除能力、按用户/群维度隔离。
- 风险：召回失准（未来阶段）。
  - 应对：分层召回（用户/群分开）+ 重排 + 在线反馈修正。

## 9. 与当前项目的适配性结论
- 技术可行：高。现有 Go 架构与消息链路天然适合插入“异步记忆写入”。
- 改造成本：中。主要是新增 `memory` 模块与少量挂点改造，不需重构主架构。
- 风险可控：高。通过“只加写入、不改主回复路径”可低风险上线。
- 建议优先级：高。先做“可追踪、可治理”的记忆储备，为后续检索拼接打基础。

## 10. 最小可执行清单（MVP）
- 完成向量库实例部署（推荐 Qdrant）。
- 打通 `EnqueueMemoryTask -> 解读 -> Embedding -> Upsert` 闭环。
- 私聊/群聊都能写入 `raw_turn` 和 `semantic` 两类记忆。
- 增加基础监控：写入成功率、队列长度、失败重试次数。
- 保证主消息处理链路零阻塞回退。

