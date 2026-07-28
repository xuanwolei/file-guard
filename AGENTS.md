# 开发约束

## 修改与验证

- 修改配置项时，必须同步更新默认值、`conf/default.ini`、`readme.md` 与测试。
- 修改 Go 代码后必须执行 `gofmt` 和 `go test ./... -count=1`；单次测试最长 60 秒。

## 日志与通知

- 多行日志必须保持“命中驱动 + 有界上下文”：超时、文件关闭、重载时均需 flush，且始终受行数和字节上限限制。
- 钉钉 Markdown 必须按 UTF-8 字节安全截断，保留代码块闭合；通知发送保持单消费者串行语义。
- 不得提交钉钉 token、实际日志或其他敏感配置。

## 发布

- 版本标签不加 `v` 前缀。
- Linux Release 必须同时提供 `linux-amd64`、`linux-arm64` 及 `SHA256SUMS`。
- `release-dist/`、`.gocache/` 为生成目录，禁止提交。
