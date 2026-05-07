# DashScope 文生图替换“涩图”计划

## Summary

- 目标：将群聊关键词 `涩图/来张涩图/整点二次元` 的行为从“随机图库”改为“调用 DashScope 文生图并回图”。
- 触发方式：保留原关键词；支持可选扩展参数（如 `涩图 海边`），参数会拼接到基础提示词后。
- 合规策略：采用“中度性感、非裸露、无性暗示”的基础提示词与负向提示词，确保在合规范围内实现“伪色图”效果。
- 配置策略：新增专用 DashScope Key 配置字段，不复用 `chatGPTkey`。
- 输出策略：默认 `1024x1024`；失败时返回简短错误并附 `request_id`（若有）。

## Current State Analysis

- 关键词分发位于 `event/message/keywords.go`，当前命中后调用 `img.SendOneImg(msg)` 发送固定站点随机图。
- 现有 `function/chatGPT/chatGPT.go` 已有旧生图能力 `AddImgPlan/CreateImg`，但目标接口为 OpenAI images，并非 DashScope 文生图接口。
- HTTP 工具在 `util/util.go`，`ChatGPTHttpPost` 目前固定使用 `chatGPTkey` 与本地代理，不适合直接承载 DashScope 专用鉴权。
- 配置结构在 `conf/config.go` 的 `BaseConfig` 中，模板文件为 `conf/conf.template.json`，当前没有 DashScope 独立 key 字段。

## Proposed Changes

### 1) 新增 DashScope 文生图能力（核心逻辑）

- 文件：`function/chatGPT/chatGPT.go`（或拆分新增 `function/chatGPT/image_dashscope.go`，保持 `chatGPT` 包内聚）。
- 变更：
  - 新增请求/响应结构体，适配接口：
    - URL：`https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation`
    - 必填参数：`model=qwen-image-2.0-pro`、`input.messages`、`parameters.size=1024*1024`、`parameters.prompt_extend=true`、`parameters.watermark=false`
    - 显式增加 `parameters.num_images_per_prompt=1`（规避已知报错 `num_images_per_prompt must be 1`）。
  - 新增入口函数（示例命名）：`AddPseudoSexyImagePlan(msgStr string, msg map[string]interface{})`。
  - 行为：
    - 解析可选参数：从原消息中去掉关键词后提取补充描述。
    - 拼接提示词：`基础提示词（中度性感、合规） + 用户补充短语（可选）`。
    - 调用 DashScope 并解析首张图片 URL。
    - 成功：发送 CQ 图片消息（`[CQ:image,file=<url>]`）或直接 URL（按现有发送兼容策略）。
    - 失败：发送简短错误文案（如“生成失败，请稍后重试”），并附 `request_id`（若响应中存在）。

### 2) 替换关键词路由到新函数

- 文件：`event/message/keywords.go`
- 变更：
  - 将 `case "来张涩图", "色图", "来张色图", "涩图", "整点二次元":` 分支中的 `img.SendOneImg(msg)` 替换为新文生图函数调用。
  - 处理入参方式：
    - 将完整消息 `msgStr` 传入新函数，便于支持“可带参数扩展”。
  - 清理 import：
    - 若 `img` 包不再被此文件使用，移除 `function/img` 引用。

### 3) 增加 DashScope 独立配置字段

- 文件：`conf/config.go`
- 变更：
  - 在 `BaseConfig` 中新增字段（示例）：
    - `DashScopeKey string \`json:"dashScopeKey"\``
  - 保持向后兼容，不影响已有 `ChatGPTKey` 使用路径。
- 文件：`conf/conf.template.json`
- 变更：
  - 增加 `dashScopeKey` 示例项，提示用户在本地 `conf.local.json` 同步配置。

### 4) 提示词与合规策略（内置）

- 实现位置：新文生图函数内常量或私有 helper。
- 策略：
  - 正向基础提示词：都市/二次元/时尚氛围 + 中度性感（修身穿搭、氛围感姿态、电影感光影）。
  - 明确禁止项（负向提示）：裸露、性行为、未成年人、强暗示、畸形、低清、文字扭曲等。
  - 用户扩展词仅追加，不允许覆盖安全约束。

### 5) 文档同步（可选但建议）

- 文件：`doc/order.md`
- 变更：
  - 将“涩图”说明从“随机图”更新为“文生图（支持追加描述）”。
  - 示例补充：`涩图 冬日街头氛围感`。

## Assumptions & Decisions

- 已确认决策：
  - 触发方式：可带参数扩展。
  - 鉴权来源：新增配置字段。
  - 风格尺度：中度性感（严格合规）。
  - 默认尺寸：`1024x1024`。
  - 失败反馈：简短错误 + `request_id`（若存在）。
- 实现约束：
  - 不引入新第三方依赖，沿用现有 HTTP/JSON 能力。
  - 使用现有异步执行器 `g.goroutineRun`，保持与当前消息处理并发模型一致。

## Verification Steps

1. 配置校验
   - 在 `conf.local.json` 补充 `dashScopeKey`，服务可正常启动且不影响旧功能。
2. 功能校验（群消息）
   - 发送 `涩图`：应返回 1 张符合“中度性感”合规图像。
   - 发送 `涩图 冬日街景 修身穿搭`：应体现补充描述要素。
3. 错误分支校验
   - 人为置空/错误 `dashScopeKey`：应返回“生成失败”简短提示，并尽量包含 `request_id`。
   - 模拟接口参数异常：确认 `num_images_per_prompt=1` 已避免已知报错。
4. 回归校验
   - `help/天气/点歌/签到` 等关键词行为不受影响。
   - `go test ./...`（若项目测试可运行）与基础编译通过。

