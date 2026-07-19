# Changelog

## 1.0.0 (2026-07-19)


### Features

* add repository versioning, release-please, commitlint, and governance ([0481870](https://github.com/giacomoRanieri/container-updater/commit/0481870ed7e733757c37bb743eb2e1516c17d7a6))
* **ci:** add PR preview Docker images, automated PR cleanup workflow, environment gate, and spec updates ([83f5246](https://github.com/giacomoRanieri/container-updater/commit/83f5246c61b13534754a9324c25381485ecc8222))
* complete implementation of container-updater backend, frontend, docker, and ci-cd ([2fd53a8](https://github.com/giacomoRanieri/container-updater/commit/2fd53a8b0b93519a09f403ec3d26eaf87a6786a7))


### Bug Fixes

* **backend:** adjust go.mod go version to 1.22 for alpine builder compatibility ([9681626](https://github.com/giacomoRanieri/container-updater/commit/9681626737d870a3fc183be978ac06bbadd0e6c4))
* **backend:** fix unused imports and crypto/ssh dependencies caught via local build ([9ad2907](https://github.com/giacomoRanieri/container-updater/commit/9ad2907657c2edf4502a9dbca7e1bc3c74e3b798))
* **backend:** tidy modules and add go mod tidy to Dockerfile.backend ([ad33fe0](https://github.com/giacomoRanieri/container-updater/commit/ad33fe08279c0f51711fbfcde9e5c861555c80a4))
* **backend:** upgrade builder image and go.mod to Go 1.23 for chi dependency ([055179a](https://github.com/giacomoRanieri/container-updater/commit/055179ad85c3920a9fcc80885c068270cbf10389))
* **backend:** use golang:alpine builder and pin x/sys v0.22.0 ([e99f414](https://github.com/giacomoRanieri/container-updater/commit/e99f4149b1bf98ae943e0c6d6eb312b127375f56))
* **ci:** allow commit headers up to 150 characters in commitlint config ([46de803](https://github.com/giacomoRanieri/container-updater/commit/46de803d1a02243d90f09192f47fa25f198c0eea))
* **ci:** grant write permissions for packages to GITHUB_TOKEN in ci-cd workflow ([3f8d71d](https://github.com/giacomoRanieri/container-updater/commit/3f8d71dfa26c6b6f7bdf682d7f2babcc25a7c5fe))
* **ci:** lowercase docker repository name for ghcr compatibility ([dde49c2](https://github.com/giacomoRanieri/container-updater/commit/dde49c2f70d8e44d43165e0577ab3903a7ca7bd5))
* **frontend:** pin pnpm to 9.15.4 in Dockerfile.frontend to avoid pnpm 10 build script policy exit code ([c2d4724](https://github.com/giacomoRanieri/container-updater/commit/c2d47249893ba2ea36865424ce72f8d23affc34f))
* **frontend:** update base docker image to node:22-alpine for pnpm compatibility ([e135f9e](https://github.com/giacomoRanieri/container-updater/commit/e135f9ef1771ec2a8d9b4f839bacaba151e4b7c9))
* **frontend:** use npm run build in Dockerfile.frontend to avoid pnpm workspace lookup error ([5da078e](https://github.com/giacomoRanieri/container-updater/commit/5da078e4c75ec4bbd631994bced4c5b46de0b989))
