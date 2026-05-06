# Tasks
- [ ] Task 1: 完成配置拆分与加载实现
  - [ ] SubTask 1.1: 在 `conf` 中定义 `BaseConfig`、`MemoryConfig`、`QdrantConfig` 与聚合配置对象
  - [ ] SubTask 1.2: 实现 `conf.local.json`、`memory.local.json`、`qdrant.local.json` 的加载与校验
  - [ ] SubTask 1.3: 增加关键字段合法性校验（阈值、超时、向量维度）

- [ ] Task 2: 建立原始回合存储能力（L1）
  - [ ] SubTask 2.1: 新增原始回合数据模型与表操作（写入、标记状态、按分组读取）
  - [ ] SubTask 2.2: 在 `private/group` 处理后接入输入侧收集
  - [ ] SubTask 2.3: 在 `AddPlan/AddPlanPrivate` 成功发送后接入输出侧收集

- [ ] Task 3: 实现分组批量总结任务
  - [ ] SubTask 3.1: 实现按 `user/group` 聚合窗口判定（回合数/字符数/等待时长）
  - [ ] SubTask 3.2: 实现调用 AI 的总结器（复用现有 AI 配置）
  - [ ] SubTask 3.3: 解析总结结果并做重要度/置信度过滤

- [ ] Task 4: 实现 Qdrant 入库模块
  - [ ] SubTask 4.1: 实现 Qdrant client 初始化与集合探测
  - [ ] SubTask 4.2: 实现向量化、upsert、去重键策略与元数据写入
  - [ ] SubTask 4.3: 区分用户集合与群集合写入路径

- [ ] Task 5: 完成异步 Worker、重试与降级
  - [ ] SubTask 5.1: 实现记忆任务队列与 worker 池
  - [ ] SubTask 5.2: 实现失败重试与退避策略
  - [ ] SubTask 5.3: 实现 Qdrant 异常时降级（仅保留 L1，不阻塞主链路）

- [ ] Task 6: 验证与回归
  - [ ] SubTask 6.1: 验证私聊/群聊均可写入原始回合
  - [ ] SubTask 6.2: 验证阈值触发批量总结并成功入库 Qdrant
  - [ ] SubTask 6.3: 验证 Qdrant 不可用时主聊天流程正常
  - [ ] SubTask 6.4: 补充必要日志与最小可观测指标

# Task Dependencies
- Task 2 依赖 Task 1
- Task 3 依赖 Task 2
- Task 4 依赖 Task 1，可与 Task 3 并行推进
- Task 5 依赖 Task 3 与 Task 4
- Task 6 依赖 Task 5
