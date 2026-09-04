Sub2API Plus v0.2.0+custom.003

## Highlights

- 发布同一官方 v0.2.0 基线上的 fork 修订版，包含 Plus v0.2.0+custom.002
  的最新客户端断开风险、内容审核会话阻断、用量完成元数据和网关修复。
- 保留 fork 的 OpenAI 官方订阅额度重置联动功能及其管理界面、任务调度和
  手动重置卡处理逻辑。

## Changed

- `plus/main` 固定镜像继续对应 Plus v0.2.0+custom.002，fork 的发布、安装
  地址和 GHCR 镜像身份保持不变。
- 保持已发布迁移不可变：Plus 的等价迁移不重复执行，新 Plus 迁移使用 fork
  的 246-251 前缀，并在迁移谱系文档中记录 checksum 映射。

## Compatibility and migration

- 本版本从 fork v0.2.0+custom.002 前向升级，按顺序执行新增的 246-251 迁移。
- 已发布 fork 迁移文件和生产数据库中的 `schema_migrations` 记录不得改名、
  覆盖或手工修改 checksum。直接从 Plus 数据库升级前，必须先盘点迁移文件名
  和 checksum，并完成兼容/adoption 设计。
- 现有账号、订阅分配和官方订阅重置联动规则保持兼容；升级后建议先以观察模式
  检查联动规则，再启用自动执行。

## Known issues

- 官方额度重置检测依赖 OpenAI OAuth 账号返回的数据；重置发生在两次轮询之间
  时，会在下一次观测时被识别。
- 首次启动可能需要为新增的用量、内容审核和客户端断开风险表执行迁移，大型
  用量日志数据库的启动时间可能增加。

## Upstream baseline

Official release: v0.2.0
Official commit: aa236488351eb71e120fc2b6fb32e36b0374c918
Plus baseline: v0.2.0+custom.002
Plus tag commit: 8df457f85568ab3b1c80de07ae59b2ef53183e80
