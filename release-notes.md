Sub2API Plus v0.2.1+custom.001

## Highlights

- 同步 LuckyKuang/sub2api-plus v0.2.1+custom.001，纳入 Plus 的最新 Codex、用量记录、频道配置和网关改进。
- 保留 fork 已有的官方订阅额度重置联动、管理界面、任务调度及手动重置卡兼容逻辑。

## Changed

- `plus/main` 镜像固定对应 Plus v0.2.1+custom.001；发布、安装地址和 GHCR 镜像继续使用本 fork。
- 迁移保持不可变：已发布 fork 迁移文件不改名、不覆盖；Plus 新迁移映射为 fork 的 252-256 前缀，并在迁移谱系中记录源文件与 checksum。

## Compatibility and migration

- 支持从 fork v0.2.0+custom.003 以及此前已发布 fork 版本前向升级。启动时会按顺序执行新增的 252-256 迁移。
- 生产数据库中的 `schema_migrations` 记录和已发布迁移 checksum 不得手工修改。升级前请先备份数据库，并确认应用使用本版本的完整迁移目录。
- 现有账号、订阅分配、Codex 会话粘性和官方订阅重置联动规则保持兼容；升级后建议先以观察模式检查联动规则，再启用自动执行。

## Known issues

- 官方额度重置检测依赖 OpenAI OAuth 账号返回的数据；若重置发生在两次轮询之间，会在下一次检查时识别。
- 首次启动可能需要执行新增的用量与频道配置迁移，大型用量日志数据库的启动时间可能增加。

## Upstream baseline

Official release: v0.2.1
Official commit: 578785ee7fb35030b094b69624efe25670a36f5f
Plus baseline: v0.2.1+custom.001
Plus tag commit: 39f6e2908975636956c184bbc084e90c8b392f74
Plus main commit: 42ad960c7d1035ea5125efebffec88a5ccab16d9
