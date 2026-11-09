# 变更日志 (CHANGELOG)

记录项目的显著变更，分类参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased] - 待发布

### Added

- [C002](docs/devel/change/C002.json)：首页新增左右两侧各半露的字符画底图（左鸢尾花、右带翼人像），浅色模式棕色、深色模式与字标 S 相同的强调橘色；底图固定在视口内，不拦截交互，打印隐藏。
- [T02](docs/devel/task/T02.json)：博客文章体系：`posts/*.md`（frontmatter：title/date/summary）为唯一真源，`scripts/build_posts.py` 构建期生成文章页（Pygments 双色语法高亮 + 脚注，无标签）并把 V3 摘要列表（标题 + 两行摘要 + Times 日期，最多 5 篇）注入首页 BLOGS；产物不入库，Actions 部署时现构建，运行时零依赖。
- [C015](docs/devel/change/C015.json)：博客正文支持数学公式：行内 `$…$` 与整行 `$$…$$` 在构建期经 matplotlib mathtext 预渲染为内联 SVG（fill 改写为 `currentColor`，浅深主题随墨色令牌自动适配），行内公式与正文基线对齐、整行公式居中且窄屏等比收缩；代码块内的 `$` 不参与解析，公式渲染失败即构建报错并报出原文；新增首篇正式文章《机器学习入门：核心范式、数学直觉与可运行代码》（含 3 张原理配图），替换原 demo 文章；构建依赖增加 matplotlib（仅含公式文章触发）。

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
- [C011](docs/devel/change/C011.json)：引文默认折叠，页面从 PROJECTS 开始；点击字标切换展开/收起，展开时滑回顶部，收起时回到 PROJECTS。
- [C012](docs/devel/change/C012.json)：页面打开时 PROJECTS、SKILLS、BLOGS、页脚从上到下依次淡入上移，间隔 80ms；顶栏不参与；不随 reduced-motion 关闭。
- [C013](docs/devel/change/C013.json)：顶栏主题按钮左侧新增搜索：默认 28px 放大镜，点击展开 180×28 纸面填充搜索栏，输入文字使用主题衬线字体与墨色；快捷键可切换展开/收起，收起后释放焦点；收起态图标与主题按钮间距与主题/GitHub 一致；窄屏收窄不溢出。
- [C014](docs/devel/change/C014.json)：主题切换改为自上而下渐变揭示：新主题以纵向渐变蒙版从视口顶部向下覆盖整页，羽化带约 14vh、600ms ease-in-out；不再以主题按钮为圆心圆形扩散，也不再向页面注入圆心与半径变量。
- [C016](docs/devel/change/C016.json)：新增博客切页过渡：点击首页列表标题（及文章页上一篇/返回首页/字标）时，内容层 200ms 淡出上移后导航；文章页标题与正文、页脚加载时 560ms 自下方淡入归位；顶栏与字符画保持静止，禁用 JS 时退回普通跳转。

### Fixed

- [C001](docs/devel/change/C001.json)：修复 SKILLS 技能轮播翻页卡顿、连续点击丢失与回绕中途吸附；翻页动画改用 Web Animations API 驱动，连续点击合并为一次连续过渡，卡片宽度改取布局宽度以消除 hover 缩放导致的偏移。

<!-- 只添加真实变更并关联真实 C/T 卡，不预登记示例卡或发布版本。
真实卡片建立后的链接语法示例：
- [T01](docs/devel/task/T01.json)：交付的能力。
- [C001](docs/devel/change/C001.json)：修复的缺陷。
填写实际发布日期和批次，归档后同步卡片路由。
-->
