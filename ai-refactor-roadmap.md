# 300Bot AI 响应与记忆重构全量路线图

> 本文档为群聊/私聊 AI 响应与长期记忆能力的五阶段重构总方案。
> 阶段一已随本次提交实施，后续阶段按本文档逐步推进。

## 1. 背景与目标

当前机器人的 AI 应答与长期记忆存在两类核心缺陷：

1. **上下文不完整**：只有触发机器人（@、喊名字、私聊）的消息才进入会话上下文，普通群聊全部丢失，机器人对刚聊过的话题"失忆"。
2. **记忆只写不读**：L1 原始记录（MySQL `memory_raw_turns`）与 L2 语义记忆（Qdrant 向量库）的写入链路已建成并运行，但没有任何检索与注入 prompt 的读取链路。

同时并发模型存在结构性问题（单协程全局串行、阻塞式令牌获取卡死整个消息管道、共享 map 无锁并发读写）。

目标：建立"**全量上下文 + 长期记忆读写闭环 + 安全并发**"的 AI 应答体系。

## 2. 问题清单与处置状态

| 编号 | 问题                                                                                         | 处置                                                      |
| ---- | -------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| P1   | 上下文只包含触发机器人的消息                                                                 | 阶段二解决                                                |
| P2   | 记忆只写不读，无检索注入链路                                                                 | 阶段三解决                                                |
| P3   | 人格设定（DB Personality）未生效                                                             | **用户明确暂缓**，当前统一人设；待人设定型后切回 DB       |
| P4   | 单协程队列 + 阻塞式获取令牌，LLM 慢时卡死整个消息管道                                        | 阶段一解决                                                |
| P5   | sessions / gptSetting map 无锁并发读写（SetPersonality 与 AskForChatGPT 竞态）               | 阶段一解决                                                |
| P6   | LLM 调用无超时、失败静默无兜底                                                               | 阶段一解决（120s 超时 + 兜底文案）                        |
| P7   | 群聊发言者区分靠"我是'昵称'"前缀，脆弱                                                       | 阶段二配合双身份方案改善                                  |
| P8   | 批量总结时间阈值只在来新消息时被动触发，无定时扫描（`ListPendingMemoryOwnerStats` 无调用者） | 阶段五已解决（5 分钟 cron 补扫）                          |
| P9   | `MemoryFallbackToMysql` 只有日志没有实现                                                     | 阶段五已解决（`memory_summary_fallback` 表 + 定时回灌）   |
| P10  | 总结失败（status=failed）的回合无补偿机制                                                    | 阶段五已解决（补扫查询含 failed）                         |
| P11  | 用户画像只有私聊来源，群内用户事实全部归到 group 维度                                        | 阶段二/三已解决（记忆锚定 QQ 号）                         |
| P12  | raw turns 缺昵称，总结素材只有 `USER[QQ号]`                                                  | 阶段二解决（昵称 + QQ 号双身份）                          |
| P13  | CollectInput 为消息热路径同步 DB insert                                                      | 阶段五已解决（channel + 批量 insert）                     |
| P14  | `memory_raw_turns` 无清理策略；dedup_window 未真正生效                                       | 阶段五已解决（每日清理 + 写前点查询）                     |
| P15  | fmt.Println 全量打印请求/响应，日志噪音大                                                    | 阶段五已收敛（单行摘要 log.Printf）                       |
| P16  | 模型名硬编码（qwen3.5-plus / deepseek-r1）                                                   | 阶段五已入配置（chatModel/storyModel/memorySummaryModel） |
| P17  | `memory=3000`/`is_limit_memory` 常量失效                                                     | 阶段二随滑窗方案重设计                                    |

## 3. 五阶段总览

```text
阶段一 并发模型重构（地基）          —— 已实施
阶段二 上下文重建（全量群聊 + NapCat 补拉 + 双身份） —— 已实施
阶段三 记忆召回注入（Qdrant search + 超时预算 + 与排队重叠） —— 已实施
阶段四 LLM 交互 JSON 化（结构化输入/输出协议） —— 已实施
阶段五 收尾（定时扫描 / fallback / 清理 / 配置化 / 日志） —— 已实施（本次）
```

## 4. 阶段一：并发模型重构（已实施）

### 4.1 调度模型

- **会话内串行**：同一 session（群号或 QQ 号）的请求进同一 FIFO 队列，保证上下文读写顺序。
- **会话间并行**：不同群/用户并发执行。
- **全局上限**：信号量限制同时在途的 LLM 请求数。
- **非阻塞入队**：队列满即丢弃并回兜底文案，绝不阻塞消息处理管道（processLoop）。
- **池隔离**：交互聊天池（chatScheduler）与后台任务池（bgScheduler，涩图/绘图/修仙故事）分离，后台任务不挤占交互回复。

### 4.2 代码落点

- 新增 `function/scheduler` 包：通用会话调度器（FIFO + 信号量 + 空闲回收 + panic recover），仅依赖标准库，带单测。
- `function/chatGPT/chatGPT.go`：
    - `sessions` / `gptSetting` 加 `sync.RWMutex`；
    - `AskForChatGPT` 改快照模式：锁内完成上下文读-改-写，锁外调用 LLM；
    - `AskForChatGPT` / `JustChatGpt` 增加 `context.WithTimeout`（默认 120s，`llmTimeoutSec` 可配置）；
    - `AddPlan` / `AddPlanPrivate` / `AddImgPlan` 改为调度器入队，新增丢弃/失败兜底文案。
- `image_dashscope.go`、`immortal.go` 迁入后台池。
- 删除旧的 `Glimit`（`function/chatGPT/goroutine.go`）。
- `conf/config.go` 新增可选项：`chatConcurrency`(默认3)、`chatQueueDepth`(默认8)、`bgConcurrency`(默认2)、`llmTimeoutSec`(默认120)。

## 5. 阶段二：上下文重建（已实施）

> 落点：`function/chatctx/window.go` 滑动窗口（`chatctx/backfill` NapCat 补拉子包）、`send/napcat.go` 只读 API 封装、`memory_raw_turns` 增 nickname 列、`AskForChatGPT` ambient 注入与双身份前缀。

### 5.1 数据源设计（推为主、拉为辅）

| 数据源             | 角色           | 说明                                                                 |
| ------------------ | -------------- | -------------------------------------------------------------------- |
| 内存滑动窗口       | 运行时主数据源 | 每群/每私聊维护最近 N 条全量消息（含发言人、时间戳），普通聊天也入窗 |
| NapCat 主动拉取    | 补洞           | 窗口空洞（重启/崩溃/漏收）时一次性补拉历史                           |
| `memory_raw_turns` | 持久兜底       | 已有的全量 L1 记录，兼作记忆管道原料                                 |

### 5.2 NapCat 能力接入

| API                      | 参数要点                                                      | 用途                           |
| ------------------------ | ------------------------------------------------------------- | ------------------------------ |
| `get_group_msg_history`  | `group_id`, `message_seq`(0=从最新), `count`, `reverse_order` | 群历史补拉                     |
| `get_friend_msg_history` | `user_id`, `count`（NapCat 扩展）                             | 私聊历史补拉                   |
| `get_group_member_info`  | `group_id`, `user_id`                                         | QQ 号 ↔ 当前昵称解析（加缓存） |
| `get_msg`                | `message_id`                                                  | 引用/回复场景补全单条消息      |

注意：NapCat 历史拉取依赖 QQ 客户端本地缓存，过老消息可能拉不到；只在窗口空洞时拉，不每次触发都拉。

### 5.3 双身份方案（P12）

- `memory_raw_turns` 新增 `nickname` 列，`CollectInput` 同时存昵称与 QQ 号。
- 总结素材格式：`用户[昵称](QQ:123456): xxx`。
- **QQ 号是唯一稳定身份键**：总结模型输出的记忆必须锚定 QQ 号，昵称仅作注释；昵称修改不导致记忆错乱。
- 滑动窗口同样以 QQ 号为键存储发言人。

### 5.4 上下文注入

`AskForChatGPT` 组装请求时，将窗口内非触发消息以"群友发言记录"形式注入（压缩为 system 段），保留 30 分钟超时语义但作用于窗口时间戳。

## 6. 阶段三：记忆召回注入（已实施）

> 落点：`function/memory/qdrant_repo.go`（Search/EmbedQuery/doJSONCtx）、`function/memory/recall.go` 触发时刻召回编排（2000ms 预算 + 两路并发 + 部分降级 + 可观测日志）、`function/memory/recall` 纯函数子包、`memory.local.json` 召回配置项。

### 6.1 检索实现

- `qdrant_repo.go` 新增 `Search(scope, ownerId, query, topK)`：`POST /collections/{c}/points/search`，payload filter 带 `user_id` / `group_id`。
- 触发时分别取 user 记忆 + group 记忆 top-k（3~5 条，带相似度阈值），拼成 system 段 `【关于对方的既有记忆】...`。

### 6.2 超时预算（两种超时严格分离）

| 超时     | 取值                | 语义                                                                                                     |
| -------- | ------------------- | -------------------------------------------------------------------------------------------------------- |
| 召回预算 | 默认 2000ms，可配置 | embedding 一次 + 两 collection **并发** search；部分降级（一路失败用另一路）；预算耗尽即弃，不带记忆继续 |
| 生成超时 | 默认 120s，可配置   | 只监控不激进截断；超时放弃本次并回兜底                                                                   |

### 6.3 与排队重叠

召回在**触发判定时刻**（入队时）即发起，而非轮到执行时；等真正调 LLM 时召回大概率已完成，外网通讯耗时被排队等待吸收。

### 6.4 可观测

每次召回打日志：命中条数、相似度、实际注入内容摘要，用于观察记忆质量与调参。

## 7. 阶段四：LLM 交互 JSON 化（已实施）

> 落点：`AskForChatGPT` 输入组装为结构化 JSON system 段（`chatctx.SnapshotJSON` + 召回 hits + 会话历史）；输出固定 schema 由 `function/memory/inline` 子包解析（失败兜底整段当 reply）；`chatJsonMode`/`chatMemoryInlineEnabled` 开关入 `conf.local.json`；回复 memory 候选经 `EnqueueInlineCandidates` 直入记忆队列。

### 7.1 输入侧

上下文组装改为结构化 JSON（发言人身份[昵称+QQ]、时间、历史消息、召回记忆），放入 system 段。与阶段二/三的注入格式统一。

### 7.2 输出侧

固定 schema（与现有记忆总结器的 JSON 输出模式统一）：

```json
{
	"should_reply": true,
	"reply": "...",
	"memory": ["顺带提取的记忆候选"]
}
```

收益：

- `should_reply` 为群聊"自主决定是否接话"留扩展位；
- 回复时顺手产出记忆候选，与批量总结互补（高频轻量 + 低频深度）；
- 为后续工具调用留扩展位。

### 7.3 风险控制

- 优先 `response_format: json_object`（实施时验证百炼 OpenAI 兼容端点对 qwen 模型的支持）；
- **解析失败兜底**：整段文本当作 reply 直接发送，绝不让用户等空；
- 保持 `enable_thinking: false`（noThinkingTransport 已就位）。

## 8. 阶段五：收尾项（已实施）

> 落点：`function/memory/scan.go`（ScanPendingOwners/CleanupSummarizedTurns）、`function/memory/fallback.go`（SaveFallback/BackfillFallback）、`function/memory/raw_writer.go`（异步批量写入器）、`interval.go` 三个 cron、`model/memory_fallback.go` 新表、`qdrant_repo.go` pointFresh 写前去重。

1. **定时补扫**：`interval.go` 注册 5 分钟 cron，调用现成的 `ListPendingMemoryOwnerStats`，对超时未总结与 `failed` 状态的 owner 重新触发 `TryBatchSummarizeOwner`。
    > 落点：`scan.go::ScanPendingOwners`（单轮至多触发 50 owner）+ `model/memory_raw.go` 查询状态扩展为 `in ('pending','failed')`。
2. **MySQL fallback 落实**：Qdrant 写入重试耗尽后落 MySQL summary 表，恢复后回灌。
    > 落点：`worker.go` 重试耗尽调用 `SaveFallback` 写 `memory_summary_fallback` 表；10 分钟 cron `BackfillFallback` 串行回灌（单轮≤20 条，失败即停）。
3. **数据清理**：`memory_raw_turns` 已 summarized 记录定期清理（如保留 30 天）。
    > 落点：每日 4 点 cron `CleanupSummarizedTurns`，`memoryRawRetentionDays` 可配置（默认 30），5000/批循环删除。
4. **模型名入配置**：聊天/总结/修仙故事模型名移入 `conf.local.json`。
    > 落点：`chatModel`/`storyModel`（BaseConfig）+ `memorySummaryModel`（MemoryConfig），代码内补默认值。
5. **日志收敛**：全量请求/响应打印改为摘要 `log.Printf`。
    > 落点：`chatGPT.go` 请求/响应各一条单行摘要（model/messages/cost_ms/tokens/preview）；`model/memory_raw.go` 错误打印同步收敛。
6. **CollectInput 异步化**：内存 channel + 批量 insert，主链路零 DB 阻塞。
    > 落点：`raw_writer.go` 单 worker（攒满 `memoryRawBatchSize` 或 1s flush），`CollectInput` 仅非阻塞入队；总结触发改在批量落库后。

另落实 P14 后半：`qdrant_repo.go::pointFresh` 写前点查询，dedup_window 窗口内重复记忆直接跳过（不重复 embed、不刷时间戳）。

## 9. 阶段一验证清单（真机）

1. 群聊 @机器人 连发 3 条：回复按顺序到达、上下文连贯（会话内串行）。
2. 两个不同群同时 @机器人：可并行回复（会话间并行）。
3. "来张涩图"与 @聊天 同时触发：互不阻塞（池隔离）。
4. "设置人格 xxx" 后立刻 @提问：不 panic（锁生效）。
5. 日志无频繁 `queue full`；LLM 卡死 120s 后释放并回兜底文案。
