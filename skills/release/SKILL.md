---
name: release
description: 为 Sub2API Plus 执行受保护的 PR-first 版本发布。使用 release-cli 和 push-cli 完成版本准备、精确合并提交验证、不可变 vX.Y.Z+custom.NNN 标签、GHCR 镜像与 GitHub Release 发布，以及发布后的 UPSTREAM.md 最终化。
---

# Sub2API Plus 发布

仅在用户明确要求发布版本时使用本技能。当前技能固定适用于
`luohao830/sub2api-plus`；如果当前仓库不是该仓库，先停止并使用目标项目
自己的发布说明。

## 固定约定

- 分发仓库：`https://github.com/luohao830/sub2api-plus`
- 默认分支：`main`，禁止直接推送或直接提交到 `main`
- Plus 来源：`https://github.com/LuckyKuang/sub2api-plus`
- 官方来源：`https://github.com/Wei-Shaw/sub2api`
- Git/GitHub Release 标签：`vX.Y.Z+custom.NNN`
- 应用内版本：`X.Y.Z+custom.NNN`
- OCI 镜像标签：`vX.Y.Z-custom.NNN`（OCI 不接受 `+`）
- 镜像：`ghcr.io/luohao830/sub2api-plus`
- 发布工作流：`.github/workflows/release.yml`，工作流名为 `Release`
- 版本真相源：`backend/cmd/server/VERSION`
- 上游映射真相源：`UPSTREAM.md`
- 发布流程真相源：`docs/RELEASING.md`

同一官方基线上的自定义迭代递增 `NNN`；合并新的官方基线后从
`001` 重新开始。不能因为上游已经使用某个 custom 序号就复用或覆盖已有
标签。

## 发布前提

开始前从仓库根目录确认远端与状态：

```bash
git remote -v
git status --short --branch
git fetch --tags --prune origin
git fetch --prune plus main
```

`origin` 必须指向个人 fork，`plus` 只用于读取 Plus 上游。发布候选必须从
最新 `origin/main` 建立工作分支，并通过 `push-cli submit-pr` 提交；不得
stash、rebase、force-push、删除远端对象或绕过分支保护。

仓库外部治理必须已经满足 `docs/RELEASING.md` 的要求，包括：

- `main` 的 pull-request 规则、严格 required checks 和 merge-commit 模式；
- GitHub Auto-merge；
- `release` Actions Environment（仅允许 `v*+custom.*` 标签，禁止管理员绕过）；
- 禁止更新/删除自定义标签的 Tag ruleset。

`release-cli` 在 promotion 和 publication 前会再次检查这些条件；任何治理
漂移都应停止并报告，不能使用 `--admin` 或手工 API 绕过。

## 准备发布 PR

1. 确认官方版本、官方提交和 Plus 标签，并在 `UPSTREAM.md` 添加目标
   `planned` 映射。新官方基线使用 `custom.001`，同一基线使用下一个三位
   序号。
2. 更新 `backend/cmd/server/VERSION` 和根目录 `Dockerfile`、
   `backend/Dockerfile` 中的 `ARG VERSION`。
3. 运行文档同步工具，并检查 README、发布元数据和迁移策略：

   ```bash
   python3 tools/update_release_docs.py
   python3 tools/update_release_docs.py --check
   python3 tools/check_readme_sync.py
   python3 tools/check_release.py --tag vX.Y.Z+custom.NNN \
     --notes-file release-notes.md --require-status planned
   python3 tools/check_new_migrations.py \
     --target-release vX.Y.Z+custom.NNN
   git diff --check
   ```

   现有 SQL migration 不可修改或复用；冲突时创建新的递增迁移文件，并在
   发布说明中说明映射。
4. 编写 `release-notes.md`。首行必须是 annotated tag subject，必需章节为：

   ```markdown
   Sub2API Plus vX.Y.Z+custom.NNN

   ## Highlights
   ...

   ## Compatibility and migration
   ...

   ## Known issues
   ...

   ## Upstream baseline
   Official release: vX.Y.Z
   Official commit: <40-character commit>
   ```

   可按需要增加 `Changed` 和 `Fixed`。说明 Plus 基线、fork 定制内容、回滚
   目标、迁移影响和已知限制；发布说明应放在 Release，不要塞进 README。
5. 在非 `main` 发布分支提交版本变更。普通中间推送使用快速路径：

   ```bash
   python3 skills/push-cli/scripts/push_cli.py push --remote origin
   ```

   最终候选只提交一次完整验证：

   ```bash
   python3 skills/push-cli/scripts/push_cli.py submit-pr --remote origin
   ```

   `submit-pr` 的默认 `full` profile 会记录精确 base/head、执行平台容器矩阵、
   推送 typed `sub2api/local-validation` 状态并创建 PR。任何 head 或 base
   变化都必须重新提交 proof。

## Promotion、标签与发布

PR 的 required checks 和 local-validation 全部成功后，使用明确 PR 号执行：

```bash
python3 skills/release-cli/scripts/release_cli.py promote-pr \
  --tag vX.Y.Z+custom.NNN --pr <release-pr> --notes-file release-notes.md
```

promotion 会使用受保护的 `gh pr merge --auto --merge`，并等待实际合并提交
上的 `CI` 与 `Security Scan`。只有这两个 workflow 在实际 `origin/main` 合并
SHA 上成功后，才允许创建标签。

切换到实际合并提交后运行聚焦元数据门禁：

```bash
python3 skills/release-cli/scripts/release_cli.py validate \
  --tag vX.Y.Z+custom.NNN --pr <release-pr> --notes-file release-notes.md
python3 skills/release-cli/scripts/release_cli.py tag \
  --tag vX.Y.Z+custom.NNN --pr <release-pr> --notes-file release-notes.md
git show --no-patch vX.Y.Z+custom.NNN
```

`tag` 只创建本地 annotated tag，不推送。确认目标提交、notes、版本映射和
远端无同名标签后，才执行唯一的不可逆发布动作：

```bash
python3 skills/release-cli/scripts/release_cli.py publish \
  --tag vX.Y.Z+custom.NNN --remote origin
```

不要使用 `git push --tags`、手工创建 Release、覆盖标签或重新递增版本掩盖
发布失败。

## 监控与验证

发布后分别执行，保持每一步可恢复：

```bash
python3 skills/release-cli/scripts/release_cli.py monitor \
  --tag vX.Y.Z+custom.NNN --remote origin
python3 skills/release-cli/scripts/release_cli.py verify \
  --tag vX.Y.Z+custom.NNN --remote origin
```

Release workflow 会校验标签 provenance、精确 main CI/Security Scan、构建
GoReleaser 制品、AMD64/ARM64 多架构 GHCR 镜像和不可变定价资产。验证必须确认
正式 GitHub Release、`model-pricing.json`、`model-pricing-manifest.json`，以及
公开镜像至少包含 `linux/amd64`：

```bash
docker buildx imagetools inspect \
  ghcr.io/luohao830/sub2api-plus:vX.Y.Z-custom.NNN
```

不要手工修改 GHCR 可见性或替换已发布定价资产；公开性以匿名 manifest/pull
验证为准。

## 发布后最终化

验证成功后，使用确定性最终化分支把 `UPSTREAM.md` 的目标映射从 `planned`
改为 `published`：

```bash
python3 skills/release-cli/scripts/release_cli.py finalize \
  --tag vX.Y.Z+custom.NNN --remote origin
```

该命令通过 `release-finalization` profile 创建后续 PR；等待其 required checks
后，不带 notes 再次 promotion：

```bash
python3 skills/release-cli/scripts/release_cli.py promote-pr \
  --tag vX.Y.Z+custom.NNN --pr <finalization-pr> --remote origin
```

最终化提交只负责发布状态和必要的生成文档，不应成为发布标签目标。标签应
继续指向已经验证的发布合并提交。

## 失败恢复与报告

- 标签已推送但终端中断：只运行 `monitor`，不要重复 `publish`。
- PR proof、base 或 head 变化：重新运行 `submit-pr`，不要重写 proof。
- Environment、Tag ruleset 或 required checks 缺失：停止并报告治理阻塞。
- 已发布版本不能 retag、删除或覆盖；修复使用下一个 custom 迭代并在
  `UPSTREAM.md` 记录状态。
- 最终报告包含旧/新版本、官方与 Plus 基线、发布提交 SHA、tag、Release URL、
  GHCR 镜像标签与 digest、main/Release workflow 结果以及迁移说明。

完整状态转换和恢复规则以仓库内的
`docs/RELEASING.md` 与 `skills/release-cli/references/release-cli.md` 为准。
