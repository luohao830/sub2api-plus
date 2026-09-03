Sub2API Plus v0.1.183+custom.005

## Highlights

- Added an administrator-configurable OpenAI subscription quota reset monitor.
- Added account and user-subscription selectors to the existing admin monitor UI.
- Cascaded selected reset windows so monthly includes weekly, weekly includes daily, and daily includes five-hour quota resets.

## Changed

- OpenAI OAuth account reset-card changes are treated as manual resets by default; an explicit option enables card-triggered subscription resets.
- Fork-owned tooling, installation links, and container image references now target `luohao830/sub2api-plus` while retaining Plus attribution and module identity.

## Fixed

- Persisted monitor execution settings and closed database query rows explicitly.
- Made GitHub CLI JSON handling deterministic when a terminal forces ANSI formatting.

## Compatibility and migration

- Migration 238 adds persistence for subscription quota reset monitor rules and execution history.
- Existing installations can apply the forward-only migration during startup; no account credentials are changed.
- This release is based on `LuckyKuang/sub2api-plus` `v0.1.183+custom.004` (tag commit `6c1e6d69398398022a832f869cdb70e69ba47c4d`).

## Known issues

- The monitor requires valid OpenAI OAuth accounts and depends on the provider's quota-reset response; it should be observed in dry-run mode before automatic execution is enabled.
- Manual reset-card actions can only be distinguished when the account reset-card state is observed between checks; a reset occurring between polls may be classified as an official reset.

## Upstream baseline

Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
Plus baseline: v0.1.183+custom.004
Plus tag commit: 6c1e6d69398398022a832f869cdb70e69ba47c4d
