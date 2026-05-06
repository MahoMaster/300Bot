# 记忆存储（Qdrant）实现 Spec

## Why
当前机器人只有短期会话上下文，缺少可持续积累的长期记忆。用户已完成本地 Qdrant 开发环境部署，需要把群聊/私聊会话沉淀为可检索记忆资产。

## What Changes
- 新增记忆存储能力：记录会话原始回合并批量总结后入库 Qdrant。
- 覆盖私聊与群聊两类主体，形成 `user` 与 `group` 维度长期记忆。
- 复用现有 AI 配置能力（现有模型请求配置）用于会话总结，不额外引入新模型配置项。
- 新增记忆模块、配置加载、数据表与异步任务流程。
- 新增失败降级策略：Qdrant 不可用时不阻塞主聊天流程，保留原始回合记录。

## Impact
- Affected specs: 会话处理、异步任务、配置管理、数据存储
- Affected code: `conf/config.go`、`event/message/message.go`、`function/chatGPT/chatGPT.go`、`model/*`、`function/memory/*`、`main.go`（或初始化入口）

## ADDED Requirements
### Requirement: 会话原始记录
系统 SHALL 在群聊和私聊消息处理后记录原始会话回合（输入、上下文标识、可选输出）。

#### Scenario: 私聊记录成功
- **WHEN** 私聊消息完成一次处理
- **THEN** 系统写入一条 `scope=user` 的原始回合记录，包含 `user_id` 与 `session_id`

#### Scenario: 群聊记录成功
- **WHEN** 群聊消息完成一次处理
- **THEN** 系统写入一条 `scope=group` 的原始回合记录，包含 `group_id` 与消息标识

### Requirement: 分组批量总结
系统 SHALL 按会话分组（私聊按用户、群聊按群）在达到阈值后触发 AI 总结，并输出结构化记忆候选。

#### Scenario: 达到阈值触发总结
- **WHEN** 同一分组的回合数、字符数或等待时间达到配置阈值
- **THEN** 系统创建总结任务并调用 AI 生成记忆候选

### Requirement: Qdrant 入库
系统 SHALL 将通过门槛的记忆候选向量化后写入 Qdrant 对应集合。

#### Scenario: 用户记忆入库
- **WHEN** 候选记忆 `scope=user` 且通过置信度/重要度门槛
- **THEN** 系统向用户记忆集合写入向量点和元数据

#### Scenario: 群记忆入库
- **WHEN** 候选记忆 `scope=group` 且通过置信度/重要度门槛
- **THEN** 系统向群记忆集合写入向量点和元数据

### Requirement: 主链路非阻塞与降级
系统 SHALL 保证记忆流程失败不影响原有消息回复。

#### Scenario: Qdrant 不可用
- **WHEN** Qdrant 写入失败或超时
- **THEN** 系统记录失败并按重试/降级策略处理，聊天主流程继续返回

## MODIFIED Requirements
### Requirement: 配置加载方式
系统 SHALL 支持多 `.local` 文件加载并合并，至少包含基础配置、记忆配置、Qdrant 配置。

#### Scenario: 配置缺失
- **WHEN** `conf.local.json`、`memory.local.json`、`qdrant.local.json` 任一缺失
- **THEN** 系统启动失败并输出明确错误信息

### Requirement: ChatGPT 流程挂载记忆收集
系统 SHALL 在现有 `AddPlan`/`AddPlanPrivate` 成功响应后追加输出侧记忆收集，不改变现有回复文案与时序。

#### Scenario: 正常回复
- **WHEN** ChatGPT 回复成功并发送
- **THEN** 系统异步提交输出侧回合数据供后续总结与入库

## REMOVED Requirements
### Requirement: 无
**Reason**: 本次为增量能力建设，不移除现有功能。
**Migration**: 无需迁移。
