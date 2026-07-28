# Hyhacct Responses 协议迁移设计

## 目标

将 Commander 的标题、场景和原型图视觉分析从 `/v1/chat/completions` 切换为 `/v1/responses`，以使用新的聊天凭据；图生图和文生图接口保持现状。

## 范围

- 仅影响 `api_key_chat` 关联的 `ChatCompletion` 与 `ChatWithVision` 调用。
- 保留 `/v1/images/generations`、`/v1/images/edits`、`api_key` 和 `model_images` 的既有行为。
- 生产配置启用 `chat_protocol: responses`；旧环境可继续使用 `chat_completions`。

## 设计

1. 在 Hyhacct 配置增加 `chat_protocol`，有效值为 `chat_completions` 与 `responses`；未配置时保持旧协议，避免破坏现有环境。
2. 将现有 `openai.ChatCompletionRequest` 转换为内部聊天输入：
   - system 文本转换为 Responses 的 `input` 中 system 消息；
   - 用户文本转换为 `input_text`；
   - 参考图 data URL 转换为 `input_image`。
3. 当协议为 `responses` 时向 `/responses` 发送请求，并从 `output` 数组的 `message/content` 中拼接 `output_text`；无可用文本时返回明确错误。
4. 当协议为 `chat_completions` 时保持原有 `/chat/completions` 请求和 `choices[0].message.content` 解析。

## 测试

- 新增失败测试，验证 Responses 文本请求路径与输入格式。
- 新增失败测试，验证视觉请求包含 `input_image` data URL。
- 新增失败测试，验证 Responses `output_text` 被正确解析。
- 保留旧协议测试，验证未配置或明确旧协议时仍请求 `/chat/completions`。
- 使用新聊天凭据对生产上游发起最小视觉探针；只有通过后更新生产配置和重启 Commander 容器。

## 部署与回滚

- 构建包含适配器的 Linux Server 二进制并按现有 Commander 发布路径部署。
- 更新线上 `config.yaml` 的 `api_key_chat` 和 `chat_protocol: responses`，不记录任何密钥到仓库或文档。
- 若视觉探针失败，将 `chat_protocol` 改回 `chat_completions` 并重启服务即可回滚；图生图不受影响。