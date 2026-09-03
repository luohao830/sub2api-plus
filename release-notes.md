Sub2API Plus v0.2.0+custom.002

## Highlights

- Imported the Plus `v0.2.0+custom.001` feature set and the follow-up Plus main
  metadata commit into the fork.
- Preserved the fork's OpenAI subscription quota reset monitor, including
  account and user-subscription targeting, dry-run observation, and reset-card
  handling.
- Added the upstream v0.2.0 Codex routing, reasoning-effort, native-compaction,
  billing, and group-policy improvements.

## Changed

- Kept the fork's distribution identity, installation links, and container image
  names while retaining Plus attribution and module compatibility.
- Kept the fork migration history immutable. The upstream native-compaction
  migration is represented as `245_add_usage_log_native_compaction_v2.sql` after
  the existing fork migration `238_subscription_quota_reset_monitors.sql`.
- Added the Plus v0.2.0 release-finalization metadata to the fork's `plus/main`
  mirror before this release was prepared.

## Fixed

- Retained the subscription reset monitor's explicit manual-reset detection so a
  reset-card use is not mistaken for an official quota reset unless configured.
- Preserved the upstream Codex identity, routing, and usage-accounting fixes
  while carrying forward the fork's monitor API, worker, UI, and tests.

## Compatibility and migration

- This release applies forward-only migrations `238` through `245`. Migration
  `238` creates the fork-owned subscription quota reset monitor tables; `239`
  through `245` add the upstream v0.2.0 usage, group-policy, pricing, and
  native-compaction fields.
- Existing account credentials and subscription assignments are unchanged.
  Review monitor rules in dry-run mode before enabling automatic execution.
- The release is based on the fork `v0.1.183+custom.005` plus the synchronized
  Plus v0.2.0 baseline and the fork's subscription quota reset monitor changes.

## Known issues

- Official quota-reset detection depends on the OpenAI OAuth account response;
  a reset that occurs between polling intervals can only be classified on the
  next observed snapshot.
- The first startup after upgrade may spend additional time applying the new
  forward-only migrations on large usage-log tables.

## Upstream baseline

Official release: v0.2.0
Official commit: aa236488351eb71e120fc2b6fb32e36b0374c918
Plus baseline: v0.2.0+custom.001
Plus tag commit: 2b921d7bf09c0484678862b854b52a4a0fb08dda
Plus main synchronization commit: 1bd44a7ef1c208eb5ff246da5269bbcc82dab6cd
