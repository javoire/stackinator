## [1.0.1](https://github.com/javoire/stackinator/compare/v1.0.0...v1.0.1) (2025-11-28)

### Bug Fixes

* combine semantic-release and goreleaser into single workflow ([47dd4ac](https://github.com/javoire/stackinator/commit/47dd4acd108c7c9d3d86eb1a1efe3ff28b11589d))

## 1.0.0 (2025-11-28)

### Features

* add help examples and development scripts ([34c354d](https://github.com/javoire/stackinator/commit/34c354d5ccd66595b5b626ae180aaea32c1c8228))
* add Homebrew tap setup with GoReleaser ([afa6ee1](https://github.com/javoire/stackinator/commit/afa6ee16b65647f34b73dc399b2534407ce9761a))
* add loading spinners for slow operations ([965c9f4](https://github.com/javoire/stackinator/commit/965c9f43d93c9a7da97e1041509c523262c2b2eb))
* add prune command with --all flag and skip pushing local-only branches ([77f42a6](https://github.com/javoire/stackinator/commit/77f42a61a4e9b23530589f6af85eb57c81a62a2a))
* add reparent command and improve sync ([009c47d](https://github.com/javoire/stackinator/commit/009c47d16040d21826d893a80097572848ae67bb))
* add stack rename command ([0e6623a](https://github.com/javoire/stackinator/commit/0e6623a6f4cbda002fe81fbdc79e9600cdbc42fd))
* add stack worktree command ([ea7e088](https://github.com/javoire/stackinator/commit/ea7e0885d8d40c1bebc4f54b3889415f5a68dff1))
* auto-track stackparent and support base branch arg ([36d6863](https://github.com/javoire/stackinator/commit/36d6863853248a29e342bca3b6560f8f2b63d2fa))
* display stack status tree after sync completion ([fe7b215](https://github.com/javoire/stackinator/commit/fe7b215625134295ea6fb5bcff4bcdbc2f3cd2bb))
* enhance status and new commands with better feedback ([7b49661](https://github.com/javoire/stackinator/commit/7b49661699aa5e6939845862b0fef5aadda829ea))
* enhance sync detection, autostash, and merged branch handling ([99781e9](https://github.com/javoire/stackinator/commit/99781e981f7d82d301f3e13a1af2816fbb047782))
* handle squash merges with rebase --onto ([2973f86](https://github.com/javoire/stackinator/commit/2973f861e19b425679d991a2323e412e04b98f54))
* improve stack display with proper tree structure and PR URLs ([ea22c43](https://github.com/javoire/stackinator/commit/ea22c4396790e4a4b3b134c14e138ebc82716503))
* initial implementation of stackinator CLI ([f9f747a](https://github.com/javoire/stackinator/commit/f9f747a1eedc19c01d9a4915855657f456c21405))
* prompt to add branch to stack on sync if not tracked ([8a5a2b7](https://github.com/javoire/stackinator/commit/8a5a2b71bf4a5fe246679a60c459513b5762e1b2))
* show PR URL in status and display stack after new ([03921b7](https://github.com/javoire/stackinator/commit/03921b74817eac89786cbb6445ee93b29f1e87b9)), closes [#1](https://github.com/javoire/stackinator/issues/1)

### Bug Fixes

* skip redundant PR base update when already correct ([958d521](https://github.com/javoire/stackinator/commit/958d5214ba174c27eab306a4e86dcbd9796cc52c))

### Performance Improvements

* cache PR data to reduce gh API calls ([ff0b068](https://github.com/javoire/stackinator/commit/ff0b068ca0e9ae33b5c6f737c2ac6c17570ad66e))
* optimize stack status performance and add progress spinners ([b167d86](https://github.com/javoire/stackinator/commit/b167d863898b4e07f2c0f086317ddde614556e40))
* optimize status command and fix stack tree filtering ([304f428](https://github.com/javoire/stackinator/commit/304f4289d4deb54e54779bafa24ff1b32d5905f6))
* parallelize network operations for faster sync, status, and prune ([59669c4](https://github.com/javoire/stackinator/commit/59669c411fcf0fdb211e88ce8bc4bdbdfa95090f))

## 1.0.0 (2025-11-28)

### Features

* add help examples and development scripts ([34c354d](https://github.com/javoire/stackinator/commit/34c354d5ccd66595b5b626ae180aaea32c1c8228))
* add Homebrew tap setup with GoReleaser ([afa6ee1](https://github.com/javoire/stackinator/commit/afa6ee16b65647f34b73dc399b2534407ce9761a))
* add loading spinners for slow operations ([965c9f4](https://github.com/javoire/stackinator/commit/965c9f43d93c9a7da97e1041509c523262c2b2eb))
* add prune command with --all flag and skip pushing local-only branches ([77f42a6](https://github.com/javoire/stackinator/commit/77f42a61a4e9b23530589f6af85eb57c81a62a2a))
* add reparent command and improve sync ([009c47d](https://github.com/javoire/stackinator/commit/009c47d16040d21826d893a80097572848ae67bb))
* add stack rename command ([0e6623a](https://github.com/javoire/stackinator/commit/0e6623a6f4cbda002fe81fbdc79e9600cdbc42fd))
* add stack worktree command ([ea7e088](https://github.com/javoire/stackinator/commit/ea7e0885d8d40c1bebc4f54b3889415f5a68dff1))
* auto-track stackparent and support base branch arg ([36d6863](https://github.com/javoire/stackinator/commit/36d6863853248a29e342bca3b6560f8f2b63d2fa))
* display stack status tree after sync completion ([fe7b215](https://github.com/javoire/stackinator/commit/fe7b215625134295ea6fb5bcff4bcdbc2f3cd2bb))
* enhance status and new commands with better feedback ([7b49661](https://github.com/javoire/stackinator/commit/7b49661699aa5e6939845862b0fef5aadda829ea))
* enhance sync detection, autostash, and merged branch handling ([99781e9](https://github.com/javoire/stackinator/commit/99781e981f7d82d301f3e13a1af2816fbb047782))
* handle squash merges with rebase --onto ([2973f86](https://github.com/javoire/stackinator/commit/2973f861e19b425679d991a2323e412e04b98f54))
* improve stack display with proper tree structure and PR URLs ([ea22c43](https://github.com/javoire/stackinator/commit/ea22c4396790e4a4b3b134c14e138ebc82716503))
* initial implementation of stackinator CLI ([f9f747a](https://github.com/javoire/stackinator/commit/f9f747a1eedc19c01d9a4915855657f456c21405))
* prompt to add branch to stack on sync if not tracked ([8a5a2b7](https://github.com/javoire/stackinator/commit/8a5a2b71bf4a5fe246679a60c459513b5762e1b2))
* show PR URL in status and display stack after new ([03921b7](https://github.com/javoire/stackinator/commit/03921b74817eac89786cbb6445ee93b29f1e87b9)), closes [#1](https://github.com/javoire/stackinator/issues/1)

### Performance Improvements

* cache PR data to reduce gh API calls ([ff0b068](https://github.com/javoire/stackinator/commit/ff0b068ca0e9ae33b5c6f737c2ac6c17570ad66e))
* optimize stack status performance and add progress spinners ([b167d86](https://github.com/javoire/stackinator/commit/b167d863898b4e07f2c0f086317ddde614556e40))
* optimize status command and fix stack tree filtering ([304f428](https://github.com/javoire/stackinator/commit/304f4289d4deb54e54779bafa24ff1b32d5905f6))
* parallelize network operations for faster sync, status, and prune ([59669c4](https://github.com/javoire/stackinator/commit/59669c411fcf0fdb211e88ce8bc4bdbdfa95090f))

## 1.0.0 (2025-11-28)

### Features

* add help examples and development scripts ([34c354d](https://github.com/javoire/stackinator/commit/34c354d5ccd66595b5b626ae180aaea32c1c8228))
* add Homebrew tap setup with GoReleaser ([afa6ee1](https://github.com/javoire/stackinator/commit/afa6ee16b65647f34b73dc399b2534407ce9761a))
* add loading spinners for slow operations ([965c9f4](https://github.com/javoire/stackinator/commit/965c9f43d93c9a7da97e1041509c523262c2b2eb))
* add prune command with --all flag and skip pushing local-only branches ([77f42a6](https://github.com/javoire/stackinator/commit/77f42a61a4e9b23530589f6af85eb57c81a62a2a))
* add reparent command and improve sync ([009c47d](https://github.com/javoire/stackinator/commit/009c47d16040d21826d893a80097572848ae67bb))
* add stack rename command ([0e6623a](https://github.com/javoire/stackinator/commit/0e6623a6f4cbda002fe81fbdc79e9600cdbc42fd))
* add stack worktree command ([ea7e088](https://github.com/javoire/stackinator/commit/ea7e0885d8d40c1bebc4f54b3889415f5a68dff1))
* auto-track stackparent and support base branch arg ([36d6863](https://github.com/javoire/stackinator/commit/36d6863853248a29e342bca3b6560f8f2b63d2fa))
* display stack status tree after sync completion ([fe7b215](https://github.com/javoire/stackinator/commit/fe7b215625134295ea6fb5bcff4bcdbc2f3cd2bb))
* enhance status and new commands with better feedback ([7b49661](https://github.com/javoire/stackinator/commit/7b49661699aa5e6939845862b0fef5aadda829ea))
* enhance sync detection, autostash, and merged branch handling ([99781e9](https://github.com/javoire/stackinator/commit/99781e981f7d82d301f3e13a1af2816fbb047782))
* handle squash merges with rebase --onto ([2973f86](https://github.com/javoire/stackinator/commit/2973f861e19b425679d991a2323e412e04b98f54))
* improve stack display with proper tree structure and PR URLs ([ea22c43](https://github.com/javoire/stackinator/commit/ea22c4396790e4a4b3b134c14e138ebc82716503))
* initial implementation of stackinator CLI ([f9f747a](https://github.com/javoire/stackinator/commit/f9f747a1eedc19c01d9a4915855657f456c21405))
* show PR URL in status and display stack after new ([03921b7](https://github.com/javoire/stackinator/commit/03921b74817eac89786cbb6445ee93b29f1e87b9)), closes [#1](https://github.com/javoire/stackinator/issues/1)

### Performance Improvements

* cache PR data to reduce gh API calls ([ff0b068](https://github.com/javoire/stackinator/commit/ff0b068ca0e9ae33b5c6f737c2ac6c17570ad66e))
* optimize stack status performance and add progress spinners ([b167d86](https://github.com/javoire/stackinator/commit/b167d863898b4e07f2c0f086317ddde614556e40))
* optimize status command and fix stack tree filtering ([304f428](https://github.com/javoire/stackinator/commit/304f4289d4deb54e54779bafa24ff1b32d5905f6))
* parallelize network operations for faster sync, status, and prune ([59669c4](https://github.com/javoire/stackinator/commit/59669c411fcf0fdb211e88ce8bc4bdbdfa95090f))

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
