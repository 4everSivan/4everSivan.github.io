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
- [C017](docs/devel/change/C017.json)：数学公式字体由 matplotlib 默认 DejaVu Sans 改为 STIX 衬线（Times 风格），与正文 Times New Roman + 宋体统一；字形随 matplotlib 内置分发，本地与 CI 构建一致。
- [C018](docs/devel/change/C018.json)：文章代码块左上角显示围栏语言；行号距左边框收紧到 8px 并默认隐藏，鼠标悬停、键盘聚焦或触屏轻触代码块时淡入，点按外部/移出后隐藏；横向溢出改用 10px 透明轨道 + 强调色胶囊滑块的自定义滚动条，悬停加深；右上角新增复制按钮（复制时剔除行号，成功后 1.6s 对勾反馈，Clipboard API 失败回退 execCommand）；Firefox 以 thin/scrollbar-color 降级。
- [C019](docs/devel/change/C019.json)：文章页新增内容列背景模糊带：固定层比 36rem 正文列左右各宽 40px，1px 高斯模糊 + 55% 纸面提亮，左右 72px 羽化淡出；列外字符画保持清晰，首页不出现，打印隐藏。
- [C021](docs/devel/change/C021.json)：文章标题层级调整：h2 提升到 20px/700 并在标题前显示两位强调色序号（`01 /`，CSS counter 生成，不改 Markdown 正文），h3 17px/700 墨色，h4 15.5px/700 次级墨，解决标题与正文难以分辨的问题。

### Fixed

- [C020](docs/devel/change/C020.json)：修复文章页背景模糊带在浅色主题下出现近白色提亮块；移除 `.article-blur` 的 55% 纸面提亮背景，只保留 1px 背景模糊与 72px 羽化，列外字符画仍保持清晰。
- [C001](docs/devel/change/C001.json)：修复 SKILLS 技能轮播翻页卡顿、连续点击丢失与回绕中途吸附；翻页动画改用 Web Animations API 驱动，连续点击合并为一次连续过渡，卡片宽度改取布局宽度以消除 hover 缩放导致的偏移。
- [C022](docs/devel/change/C022.json)：修复数学公式内联 SVG 自带的 `<style>` 作用于整页、把文章页顶栏与复制按钮图标的线帽改成 `butt`；同时去掉每个公式重复的 `figure_1`/`text_1` 等 id，字形定义收拢为每页一份共享 defs（整页 id 唯一，文章页 309KB→162KB），并为每个公式补 `role="img"` 与 TeX 原文替代文本。
- [C023](docs/devel/change/C023.json)：修复文章页「← 上一篇」指向更新的文章（最新一篇反而没有上一篇）的方向错误，改为指向发布日期更早的一篇；标题或摘要含 `\U`、`\1` 等反斜杠时不再导致构建崩溃或文本被改写；`--out` 指向仓库外目录时构建不再在结尾报错退出。
- [C024](docs/devel/change/C024.json)：修复键盘用户 Tab 到 SKILLS 技能卡时卡片停在视窗边缘甚至屏外、浏览器滚动轮播视窗后轨道位置与索引永久错开的问题；键盘聚焦的卡片现在滑到正中，回车即打开，视窗改为 `overflow: clip` 且脚本兜底把 `scrollLeft` 归零。
- [C025](docs/devel/change/C025.json)：修正文档与测试失真：切页过渡回归测试改为等待导航完成与入场动画 `finished`，不再因固定 900ms 采样偶发失败；部署指南补上 `matplotlib` 依赖、改正已删除的示例文章路径与「减少动态效果」说明；C021 在本日志中改归入「Changed」；样式表与脚本中与现状不符的注释同步改正。

<!-- 只添加真实变更并关联真实 C/T 卡，不预登记示例卡或发布版本。
真实卡片建立后的链接语法示例：
- [T01](docs/devel/task/T01.json)：交付的能力。
- [C001](docs/devel/change/C001.json)：修复的缺陷。
填写实际发布日期和批次，归档后同步卡片路由。
-->
