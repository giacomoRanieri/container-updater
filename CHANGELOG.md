# Changelog

## 1.0.0 (2026-07-24)


### Features

* add registry credential management and workspace sync ([1cdc3ca](https://github.com/giacomoRanieri/container-updater/commit/1cdc3ca1bac6caecb3a718d473f420483bccdd7b))
* add registry credential management and workspace sync ([38681f2](https://github.com/giacomoRanieri/container-updater/commit/38681f28b6eb298b73dc18b1d9040102bea84854))
* add repository versioning, release-please, commitlint, and governance ([0481870](https://github.com/giacomoRanieri/container-updater/commit/0481870ed7e733757c37bb743eb2e1516c17d7a6))
* **api:** implement POST /api/scan endpoint and wire Scan Now frontend button ([881e2fa](https://github.com/giacomoRanieri/container-updater/commit/881e2faf7393803a07604ca2587c865480e00242))
* **ci:** add PR preview Docker images, automated PR cleanup workflow, environment gate, and spec updates ([83f5246](https://github.com/giacomoRanieri/container-updater/commit/83f5246c61b13534754a9324c25381485ecc8222))
* complete implementation of container-updater backend, frontend, docker, and ci-cd ([2fd53a8](https://github.com/giacomoRanieri/container-updater/commit/2fd53a8b0b93519a09f403ec3d26eaf87a6786a7))
* **k8s:** monitor async rollout status & stream live progress events (fixes [#28](https://github.com/giacomoRanieri/container-updater/issues/28), fixes [#29](https://github.com/giacomoRanieri/container-updater/issues/29)) ([99107ba](https://github.com/giacomoRanieri/container-updater/commit/99107ba7abf1f32a873f9dc445f5d24c91ea9ddc))
* **k8s:** monitor async rollout status, stream live progress & fix multi-doc YAML (fixes [#28](https://github.com/giacomoRanieri/container-updater/issues/28), fixes [#29](https://github.com/giacomoRanieri/container-updater/issues/29), fixes [#31](https://github.com/giacomoRanieri/container-updater/issues/31)) ([cba993f](https://github.com/giacomoRanieri/container-updater/commit/cba993fc5aab337b14575a76e73d5b3fbbb734f9))
* **monitor:** add SemVer tag tracking for version-pinned updates (fixes [#19](https://github.com/giacomoRanieri/container-updater/issues/19)) ([2988d66](https://github.com/giacomoRanieri/container-updater/commit/2988d66e52a54910d6f4b0db113a4c9cf5d8b7a6))


### Bug Fixes

* **api:** resolve SQLite scan error on stats calculation MAX(last_checked_at) (fixes [#15](https://github.com/giacomoRanieri/container-updater/issues/15)) ([2ecb7ee](https://github.com/giacomoRanieri/container-updater/commit/2ecb7eedc12ccc8bc1276c34c0f8f3d4830355f6))
* **auth:** dynamically resolve frontend redirect URL in OIDC callbacks ([eda2385](https://github.com/giacomoRanieri/container-updater/commit/eda23854496edb402170aa02948f05213248d418))
* **backend:** adjust go.mod go version to 1.22 for alpine builder compatibility ([9681626](https://github.com/giacomoRanieri/container-updater/commit/9681626737d870a3fc183be978ac06bbadd0e6c4))
* **backend:** fix unused imports and crypto/ssh dependencies caught via local build ([9ad2907](https://github.com/giacomoRanieri/container-updater/commit/9ad2907657c2edf4502a9dbca7e1bc3c74e3b798))
* **backend:** tidy modules and add go mod tidy to Dockerfile.backend ([ad33fe0](https://github.com/giacomoRanieri/container-updater/commit/ad33fe08279c0f51711fbfcde9e5c861555c80a4))
* **backend:** upgrade builder image and go.mod to Go 1.23 for chi dependency ([055179a](https://github.com/giacomoRanieri/container-updater/commit/055179ad85c3920a9fcc80885c068270cbf10389))
* **backend:** use golang:alpine builder and pin x/sys v0.22.0 ([e99f414](https://github.com/giacomoRanieri/container-updater/commit/e99f4149b1bf98ae943e0c6d6eb312b127375f56))
* **ci:** add manual workflow_dispatch trigger to pr-cleanup workflow ([ce295e1](https://github.com/giacomoRanieri/container-updater/commit/ce295e14e6fae91516391f679b4e23e13483e7cb))
* **ci:** add manual workflow_dispatch trigger to pr-cleanup workflow ([5570562](https://github.com/giacomoRanieri/container-updater/commit/5570562d86eae4b66fc0561813030190551ba64d))
* **ci:** add workflow_dispatch trigger to ci-cd workflow ([aadbd6a](https://github.com/giacomoRanieri/container-updater/commit/aadbd6a48bdd25ca5ce394e8d43ea3493432d46d))
* **ci:** add workflow_dispatch trigger to ci-cd workflow ([b7cf5d8](https://github.com/giacomoRanieri/container-updater/commit/b7cf5d82ce4cd6f556e275c02eaad0240e37f061))
* **ci:** allow commit headers up to 150 characters in commitlint config ([46de803](https://github.com/giacomoRanieri/container-updater/commit/46de803d1a02243d90f09192f47fa25f198c0eea))
* **ci:** fix GHCR package deletion REST endpoints for user-owned containers ([df41ffe](https://github.com/giacomoRanieri/container-updater/commit/df41ffee7bc344f6a997a4fdcc12bd2f4806a606))
* **ci:** fix GHCR package deletion REST endpoints for user-owned containers ([7aff01d](https://github.com/giacomoRanieri/container-updater/commit/7aff01d06268df15b601c005b480e86cb7bd1018))
* **ci:** grant write permissions for packages to GITHUB_TOKEN in ci-cd workflow ([3f8d71d](https://github.com/giacomoRanieri/container-updater/commit/3f8d71dfa26c6b6f7bdf682d7f2babcc25a7c5fe))
* **ci:** lowercase docker repository name for ghcr compatibility ([dde49c2](https://github.com/giacomoRanieri/container-updater/commit/dde49c2f70d8e44d43165e0577ab3903a7ca7bd5))
* **ci:** revert direct commit on develop ([ddad6dc](https://github.com/giacomoRanieri/container-updater/commit/ddad6dc7374d84a71ee785d4fffcdbfb192533e5))
* **db:** prevent SQLite timestamp scan errors and allow dynamic CORS origins ([25d11a9](https://github.com/giacomoRanieri/container-updater/commit/25d11a95a0a4be5453e2d7f6a14ed1de64706c50))
* **frontend:** pin pnpm to 9.15.4 in Dockerfile.frontend to avoid pnpm 10 build script policy exit code ([c2d4724](https://github.com/giacomoRanieri/container-updater/commit/c2d47249893ba2ea36865424ce72f8d23affc34f))
* **frontend:** update base docker image to node:22-alpine for pnpm compatibility ([e135f9e](https://github.com/giacomoRanieri/container-updater/commit/e135f9ef1771ec2a8d9b4f839bacaba151e4b7c9))
* **frontend:** use npm run build in Dockerfile.frontend to avoid pnpm workspace lookup error ([5da078e](https://github.com/giacomoRanieri/container-updater/commit/5da078e4c75ec4bbd631994bced4c5b46de0b989))
* **k8s:** preserve all resources in multi-document YAML files during manifest edit (fixes [#31](https://github.com/giacomoRanieri/container-updater/issues/31)) ([2911c18](https://github.com/giacomoRanieri/container-updater/commit/2911c183c00289eedcedee0e2246aaee56b1b0a1))
* **k8s:** preserve original YAML indentation and AST comments during manifest edit (FR-026) ([dc9a078](https://github.com/giacomoRanieri/container-updater/commit/dc9a078325f6744faf667815b6450c55a2f0c333))
* **k8s:** remove leftover merge markers and fix syntax error in update.go ([1b5b151](https://github.com/giacomoRanieri/container-updater/commit/1b5b1511b3b2367c5b1e2075abeef8f3947274e8))
* **k8s:** resolve deployment name parsing error for hyphenated names (fixes [#26](https://github.com/giacomoRanieri/container-updater/issues/26)) ([fa6bba7](https://github.com/giacomoRanieri/container-updater/commit/fa6bba754c1dfb6aa0ae4c2712ba32be2f352088))
* **k8s:** resolve deployment name parsing error for hyphenated names (fixes [#26](https://github.com/giacomoRanieri/container-updater/issues/26)) ([0a999e3](https://github.com/giacomoRanieri/container-updater/commit/0a999e38fd68174f67328dfe2bdf5a983e40316b))
* **monitor:** implement pure HTTP OCI Registry v2 client for daemonless registry inspects (fixes [#17](https://github.com/giacomoRanieri/container-updater/issues/17)) ([be22bec](https://github.com/giacomoRanieri/container-updater/commit/be22bec0301ffce139a0c6220e42642c6f06ef0f))
* **monitor:** run registry update checks for Kubernetes workloads when Docker daemon client is disabled (fixes [#16](https://github.com/giacomoRanieri/container-updater/issues/16)) ([733aaee](https://github.com/giacomoRanieri/container-updater/commit/733aaee18d3fb5e047a630de13f4ad96f5e86edf))
* **registry:** paginate ListTags via RFC 5988 Link headers (fixes [#21](https://github.com/giacomoRanieri/container-updater/issues/21)) ([e90f277](https://github.com/giacomoRanieri/container-updater/commit/e90f277323131c2df52ea5c11d16b5308187b322))
* **registry:** use RFC 7235 quoted-string parser in parseHeaderParams (fixes [#20](https://github.com/giacomoRanieri/container-updater/issues/20)) ([f25d958](https://github.com/giacomoRanieri/container-updater/commit/f25d9582f430b976f4d1cb508f91f93385dcc4b2))
* resolve runtime backend URL injection and OIDC issuer mismatch (fixes [#12](https://github.com/giacomoRanieri/container-updater/issues/12), fixes [#13](https://github.com/giacomoRanieri/container-updater/issues/13)) ([adbaf87](https://github.com/giacomoRanieri/container-updater/commit/adbaf8760c2b98c3b025ae24a54479c62601fb53))
* resolve runtime backend URL, OIDC issuer, stats scan, K8s check & HTTP OCI client (fixes [#12](https://github.com/giacomoRanieri/container-updater/issues/12), fixes [#13](https://github.com/giacomoRanieri/container-updater/issues/13), fixes [#15](https://github.com/giacomoRanieri/container-updater/issues/15), fixes [#16](https://github.com/giacomoRanieri/container-updater/issues/16), fixes [#17](https://github.com/giacomoRanieri/container-updater/issues/17)) ([c1cf796](https://github.com/giacomoRanieri/container-updater/commit/c1cf7969f3563a74d12f84a501f701610efb9a9d))
* **ui:** resolve workload card badge overlap, title truncation and redundant names ([c2f8542](https://github.com/giacomoRanieri/container-updater/commit/c2f85428249614d0fb4a4ae223efae5f0732daaf))
* **ui:** resolve workload card overlap, redundant names, POST /api/scan & SemVer tag tracking (fixes [#19](https://github.com/giacomoRanieri/container-updater/issues/19)) ([f3e0525](https://github.com/giacomoRanieri/container-updater/commit/f3e05255575c76147976b74fa48e39b6640ccf5d))
