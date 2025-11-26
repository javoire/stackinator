## 1.0.0 (2025-11-26)

### Features

* add help examples and development scripts ([25a6304](https://github.com/javoire/stackinator/commit/25a630403ae733c36e12cdafb3fee0d533df199a))
* add Homebrew tap setup with GoReleaser ([a3c8d29](https://github.com/javoire/stackinator/commit/a3c8d29d25bf321f7608d8a2012ee6d8bd8298fc))
* add loading spinners for slow operations ([2faeaf0](https://github.com/javoire/stackinator/commit/2faeaf0f764decded67689827e924cb331b4b4ae))
* add prune command with --all flag and skip pushing local-only branches ([ab03859](https://github.com/javoire/stackinator/commit/ab038598ff1562bcfe6de08c069232fb9117b51b))
* add reparent command and improve sync ([5d8a514](https://github.com/javoire/stackinator/commit/5d8a5146e46a0f0a8b41d0945017f7201628b8b0))
* add stack rename command ([2ff4ed1](https://github.com/javoire/stackinator/commit/2ff4ed11c1b3499f311ef55984ae806b76e68462))
* add stack worktree command ([2409143](https://github.com/javoire/stackinator/commit/24091437d558ad78bf098ab0c28eabf4e09918c7))
* auto-track stackparent and support base branch arg ([ec53a4b](https://github.com/javoire/stackinator/commit/ec53a4b78463e3cb45523d8eefb7adf806dde3ae))
* display stack status tree after sync completion ([afb9154](https://github.com/javoire/stackinator/commit/afb9154d9ac3c99335336e580d1d7c058cc42c1c))
* enhance status and new commands with better feedback ([c9300d1](https://github.com/javoire/stackinator/commit/c9300d14a64a52570a0b4a55cbbba59a61e7311e))
* enhance sync detection, autostash, and merged branch handling ([599d004](https://github.com/javoire/stackinator/commit/599d00436534ecd167be15761bd89c690e08aa16))
* handle squash merges with rebase --onto ([f3588c5](https://github.com/javoire/stackinator/commit/f3588c50d2ea1c2e2f84731d0ceb715cdd28ed74))
* improve stack display with proper tree structure and PR URLs ([1db69c3](https://github.com/javoire/stackinator/commit/1db69c3d25df07cfbbb9f1ea48f62c49be1fe52f))
* initial implementation of stackinator CLI ([b2a8142](https://github.com/javoire/stackinator/commit/b2a81427db768795fe3d56eecff914187b1f56c2))
* show PR URL in status and display stack after new ([1c6675b](https://github.com/javoire/stackinator/commit/1c6675b7a06d0f8cce0daa96b83c83b7dac39876)), closes [#1](https://github.com/javoire/stackinator/issues/1)

### Performance Improvements

* cache PR data to reduce gh API calls ([db339d7](https://github.com/javoire/stackinator/commit/db339d7c8a74b1956cbfa9b3fc691c3d69a805c9))
* optimize stack status performance and add progress spinners ([1e94f29](https://github.com/javoire/stackinator/commit/1e94f295ae9eed44df1debd163720180739d828e))
* optimize status command and fix stack tree filtering ([63a05d0](https://github.com/javoire/stackinator/commit/63a05d0772d1cc2977740c05abeffc19eedc3667))
* parallelize network operations for faster sync, status, and prune ([2ac41b1](https://github.com/javoire/stackinator/commit/2ac41b1b1552e3dac51c886090d7885ac5525e38))
