# 变更日志 (CHANGELOG)

记录项目的显著变更，分类参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased] - 待发布

### Added

- [C002](docs/devel/change/C002.json)：首页新增左右两侧各半露的字符画底图（左鸢尾花、右带翼人像），浅色模式棕色、深色模式与字标 S 相同的强调橘色；底图固定在视口内，不拦截交互，打印隐藏。

### Changed

- [T01](docs/devel/task/T01.json)：仓库改为参考 egoist.dev 的无构建个人主页：宋体与 Times New Roman 混排、杏色/暖黑双主题、36rem 单栏、双列项目卡片与 SKILLS 技能区；新增 GitHub Actions 静态 Pages 部署 workflow。
- [C003](docs/devel/change/C003.json)：浅色模式字标 S 改为墨色、`ivan.` 改为次级灰；主题切换新增以按钮为圆心的 600ms 圆形扩散，不支持 View Transitions 时瞬时切换。
- [C004](docs/devel/change/C004.json)：隐藏浏览器右侧滚动栏，保留滚轮、触控板与键盘滚动。
- [C005](docs/devel/change/C005.json)：首页 SKILLS 下新增 BLOGS 区块；暂无文章时显示 22px 羽毛笔线稿 + `COMING SOON`，无边框、无链接。
- [C006](docs/devel/change/C006.json)：页脚回到文档流，滚动到底部时贴底，页面底部留白 24px；顶栏改用纸面色到透明的渐变遮罩并移除分隔线。
- [C007](docs/devel/change/C007.json)：主题切换时太阳/月亮图标改为 420ms 旋转 + 淡入淡出，独立于整页颜色扩散。
- [C008](docs/devel/change/C008.json)：页脚 © 2026 Sivan 浅色为次级灰、深色为强调橘色。
- [C009](docs/devel/change/C009.json)：项目卡与技能卡改为图标与标题同行、描述横跨下方；窄屏单列、宽屏双列。
- [C010](docs/devel/change/C010.json)：点击顶栏字标改为 460ms 缓动滚回顶部，reduced-motion 下也不再瞬时跳转。
- [C011](docs/devel/change/C011.json)：引文默认折叠，页面从 PROJECTS 开始；点击字标展开引文并滑回顶部。

### Fixed

- [C001](docs/devel/change/C001.json)：修复 SKILLS 技能轮播翻页卡顿、连续点击丢失与回绕中途吸附；翻页动画改用 Web Animations API 驱动，连续点击合并为一次连续过渡，卡片宽度改取布局宽度以消除 hover 缩放导致的偏移。

<!-- 只添加真实变更并关联真实 C/T 卡，不预登记示例卡或发布版本。
真实卡片建立后的链接语法示例：
- [T01](docs/devel/task/T01.json)：交付的能力。
- [C001](docs/devel/change/C001.json)：修复的缺陷。
填写实际发布日期和批次，归档后同步卡片路由。
-->
