# 三层仓库的迁移谱系与兼容规则

本文记录 `Wei-Shaw/sub2api`（官方） -> `LuckyKuang/sub2api-plus`
（Plus） -> `luohao830/sub2api-plus`（本 fork）这条三层开发链上的数据库迁移
规则。它是本 fork 的维护记忆，不替代
[`backend/migrations/README.md`](../backend/migrations/README.md) 的基本约束。

## 核心结论

数据库迁移的兼容性不由应用版本号、Git tag 或数字前缀单独决定。运行器使用
完整迁移文件名和文件内容的 SHA256 校验和作为身份，并把它们记录在
`schema_migrations` 中。因此，某个迁移一旦进入本 fork 的发布版本或被任意
环境执行，它的文件名和内容就是不可变的数据库协议。

三层仓库的优先级必须分成两类理解：

| 问题 | 权威来源 |
| --- | --- |
| 生产库能否继续升级 | 生产库 `schema_migrations` 的 `filename + checksum` |
| 本 fork 哪些迁移不能改变 | 本 fork 已发布 tag 中的迁移文件 |
| Plus 改动的来源和意图 | Plus 的发布 tag、同步提交和迁移 SQL |
| 原始功能语义 | 官方对应 tag/commit 和 SQL |

换句话说，官方仓库是语义来源，Plus 是直接父仓库，本 fork 的已发布迁移
历史是最终兼容边界。不能因为上游代码更新了，就让上游文件覆盖本 fork
已经发布的迁移。

## 运行器契约

当前运行器的关键行为如下：

- 迁移记录表的主键是完整 `filename`，而不是数字前缀。
- 文件内容（去除首尾空白后）计算 SHA256；已记录文件 checksum 不匹配时，
  启动失败。
- 所有 `*.sql` 按完整文件名的字典序执行；数字前缀主要用于控制顺序。
- 普通迁移在事务中执行并和记录一起提交；`*_notx.sql` 为并发索引专用，
  可能留下部分执行结果，失败后必须人工复核。
- PostgreSQL advisory lock 只负责串行化多个实例，不能解决语义重复或错误
  的迁移映射。
- 新迁移在服务正常启动前运行；回滚应用镜像不会自动回滚数据库。

实现见 [`migrations_runner.go`](../backend/internal/repository/migrations_runner.go:18)。

运行器只遍历当前程序内嵌的文件。它不会要求数据库中的旧记录仍然出现在
当前文件树中，所以把旧文件改名后，旧记录可能变成“孤儿记录”，而新文件
会被当成未执行迁移再次运行。这正是禁止重命名的原因。

## 冲突决策

### 同一完整文件名，内容不同

例如两个分支都存在 `228_feature.sql`，但 SQL 不同：

- 若生产库已执行其中一版，选择另一版会触发 checksum mismatch，应用无法
  启动；这通常是保护性失败，不是安全升级。
- 若生产库尚未执行，选择一版会让另一版的 schema/data 变化丢失；新库和
  不同环境可能因此不一致。
- 不能用普通 checksum 兼容白名单解决分叉冲突。白名单只适用于已经审计的
  历史事故，不能作为日常合并手段。

处理方式是保留已经发布的文件原名和原文；如果还需要另一套语义，新增一个
严格递增的 forward-only 迁移来补足，而不是改写旧文件。若两套版本都已经
在受支持环境中执行，则必须设计一次性的等价迁移接管/兼容桥，或者明确不
支持该数据库来源，不能凭 Git 合并结果猜测。

### 数字前缀相同，完整文件名不同

例如 `228_fork.sql` 和 `228_plus.sql`。运行器会把它们当成两个迁移，按字典
序都执行。可能出现重复建表、重复回填、约束冲突或依赖顺序错误；SQL 中有
`IF NOT EXISTS` 也不能证明数据副作用安全。

本 fork 的规则是：

1. 已发布的本 fork 文件保持不动。
2. 尚未进入本 fork 发布历史的 Plus 文件，改用本 fork 当前最大前缀之后的
   第一个可用编号。
3. 保持一组相互依赖的 Plus 迁移的相对顺序，并同步修改测试、代码中的文件
   引用和发布说明。
4. 如果新文件与本 fork 已有文件实际上是同一逻辑（相同 checksum 或经过
   审计确认等价），不要让两份 SQL 都执行；记录等价映射，并为可能缺失的
   schema 状态补充专门的幂等迁移。

这里的“改编号”只适用于尚未在本 fork 的受支持数据库中执行的外来迁移。它
不是把本 fork 已有的 `228` 改成 `230`。

### 已发布文件改名

禁止把 `228_my_change.sql` 改成 `230_my_change.sql`。旧数据库会保留
`228_my_change.sql` 的记录，新程序却会执行 `230_my_change.sql`，导致重复
执行、重复数据、约束错误或启动失败。`check_new_migrations.py` 也会把相对
基线的删除/重命名视为违规；它不能替代生产数据库盘点。

## 支持的升级来源

默认只保证：

```text
本 fork 的已发布版本 -> 本 fork 的新版本
```

如果某个用户曾经直接运行 Plus 镜像，数据库里可能已经有 Plus 文件名和
checksum。把该 Plus 迁移改成 fork 新编号后，运行器可能再次执行它；对于
简单的 `ADD COLUMN IF NOT EXISTS` 也许无害，但对回填、插入、触发器、索引
和非事务迁移不能作此假设。

因此，在声明支持 `Plus -> 本 fork` 之前必须：

- 盘点实际部署过的 Plus 版本和数据库记录；
- 保留所有可能已执行的 Plus 文件，或提供经过审计的 migration adoption
  机制；
- 在真实的 Plus 数据库快照上验证启动、schema、数据和重复执行行为；
- 在发布说明中明确支持范围和不支持的跨发行版升级路径。

官方数据库直接升级到本 fork 也遵循同样规则，不能因为官方是最初来源就
自动视为兼容。

## 每次 Plus 同步的标准流程

1. 更新 `plus/main`，使它精确镜像 Plus 的 `main`；不要在该分支加入 fork
   改动。
2. 从最新 fork `main` 创建 `sync/*` 分支，生成三层迁移清单：
   `source layer / source filename / source checksum / fork filename /
   fork checksum / status`。
3. 对每个迁移分类：`unchanged`、`fork-released`、`new-parent`、
   `equivalent`、`collision` 或 `unsupported-source`。
4. 保留 fork 已发布迁移的原名原文。对真正未发布的 Plus 新迁移使用大于
   fork 当前最大值的新前缀；不要用 `git mv` 改动旧文件。
5. 对有依赖的迁移按整体顺序审查。若原编号迁移必须早于已有 fork 迁移，
   不能只靠重新编号，应采用 expand/compatibility/contract 的前向方案，或
   暂停同步并设计桥接迁移。
6. 合并应用代码时以 Plus 为默认输入，但对 fork 的功能、协议、审计、身份
   和迁移历史逐项复核；不能使用无差别的 `-X theirs`。
7. 在同步 PR 中同时更新 `UPSTREAM.md`、迁移映射和测试说明。现有
   `check_new_migrations.py` 只检查 Git diff、命名和前缀，不能证明生产来源
   兼容性。

## 发布前验证矩阵

至少验证以下数据库状态：

| 数据库状态 | 必须确认 |
| --- | --- |
| 全新空库 | 全部最终迁移按顺序成功，schema 与代码一致 |
| 最新 fork 发布库 | 已发布文件 checksum 全部匹配，新迁移各执行一次 |
| 每个声明支持的 Plus 发布库 | Plus 文件身份、等价映射和补偿迁移均正确 |
| 含部分失败迁移的恢复库 | 普通事务迁移可重试；`_notx` 按运行手册处理 |
| 真实生产代表性快照 | 数据量、索引、约束、回填和启动时间可接受 |

升级前备份 PostgreSQL。先以 canary 实例启动新版本，确认迁移成功后再
扩展实例；advisory lock 不能替代备份和滚动发布策略。

## 版本号与迁移编号

应用版本 `vX.Y.Z+custom.NNN` 是发布身份，迁移前缀是本 fork 数据库历史的
全局执行序列，两者独立：

- 新的应用版本不会让旧迁移重新编号。
- Plus 和本 fork 可以有同名 tag，但不代表数据库迁移集合相同。
- 同一官方基线的 fork custom 序号按发布规则递增；迁移前缀继续从 fork
  历史最大值向前，不随 custom 序号重置。
- Release notes 应写明 Plus tag/commit、本 fork 保留的功能，以及每个外来
  迁移的重编号或等价映射。

## 已验证的 Plus 先例

Plus 自己已经采用过同样的下游处理方式：

- 在官方 v0.1.184 导入中，将官方 `231` 迁移放到 Plus 的 `238/239/240`，
  以避开 Plus 已占用的编号。
- 在 Plus v0.2.0 的功能合并前，将功能分支的 `238-243` 改为 `245-250`，
  提交说明明确写着是为了避开 `main` 的 `238-244`。
- 这些重编号发生在对应文件进入该发行版生产历史之前；它们不是“已发布
  迁移改名也没关系”的先例。

因此，Plus 的合并历史和迁移 README 是本 fork 处理外来迁移的直接参考；
官方 SQL/commit 用来确认功能语义。本 fork 的生产记录和已发布文件仍然拥有
最高兼容优先级。

## 本次 Plus v0.2.0+custom.002 同步记录

本次同步来源为 Plus tag `v0.2.0+custom.002`，提交
`8df457f85568ab3b1c80de07ae59b2ef53183e80`。同步前 fork `main` 的基线为
`4748515e594f5580f18aac6f93fcc3dbe4f3b49b`。

| Plus 文件 | Plus SHA256 | fork 文件 | 处理 |
| --- | --- | --- | --- |
| `238_add_usage_log_native_compaction_v2.sql` | `8e0ce1864caea450f49b250a2b9c992ef19698a7d689ef772b256a7114e56474` | `245_add_usage_log_native_compaction_v2.sql` | 内容等价，保留 fork 已发布文件，不重复执行 |
| `245_client_disconnect_risk.sql` | `ef5f2dfd8e66c3ab2e7d657e5ddf511bb9ff2415802cc2544266c6eed53595b0` | `246_client_disconnect_risk.sql` | 新增迁移，顺延编号 |
| `246_client_disconnect_lifecycle_observability.sql` | `bd5173e218035dd495dfba7dbad56c58f8741cea616e5695d4520dbd440e6036` | `247_client_disconnect_lifecycle_observability.sql` | 新增迁移，顺延编号 |
| `247_usage_log_completion_metadata.sql` | `aafba61803f1cdbd9a4d657b5cf61206b2ca106a344f798a70019298722e4933` | `248_usage_log_completion_metadata.sql` | 新增迁移，顺延编号 |
| `248_content_moderation_session_blocks.sql` | `aaf08475423206072a1b1e59761d01d2f61751c64a0676db702685ec2ed33ed5` | `249_content_moderation_session_blocks.sql` | 新增迁移，顺延编号 |
| `249_content_moderation_session_blocks_unique.sql` | `2ae262d29d860e66ee403f1217bec3b9efb215ad02aeab4bba7360f04a10f0c7` | `250_content_moderation_session_blocks_unique.sql` | 新增迁移，顺延编号 |
| `250_content_moderation_input_content.sql` | `1b38db1209928a0836e38f2cd3da2b74c635321808ed6163378a1ea61596d847` | `251_content_moderation_input_content.sql` | 新增迁移，顺延编号 |

这些重编号只适用于尚未进入 fork 支持范围的 Plus 文件。若数据库曾直接运行
Plus 版本，必须先完成 Plus 文件名的执行记录盘点和 adoption/兼容桥设计，
不能把本表当作自动接管证明。

## 生产盘点查询

在执行同步或升级前，保存以下结果（不要把数据库内容提交到 Git）：

```sql
SELECT filename, checksum, applied_at
FROM schema_migrations
ORDER BY filename;
```

将查询结果与本 fork 发布 tag、Plus tag 和迁移映射清单逐项比对。发现未知
文件名、checksum 不一致、来源不明或曾经直接运行过其他发行版时，应先停止
自动发布，完成兼容性设计和代表性数据库测试。

## 禁止的捷径

- 让 Plus 文件覆盖本 fork 已发布的同名迁移。
- 把已发布的 fork 迁移 `git mv` 到更大的编号。
- 手工修改 `schema_migrations.checksum`。
- 为普通分叉冲突添加 checksum 兼容白名单。
- 因为测试只通过了全新数据库，就宣称生产升级安全。
- 用应用镜像回滚代替数据库备份或补偿迁移。
