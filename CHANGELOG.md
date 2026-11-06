# 变更日志 (CHANGELOG)

记录项目的显著变更，分类参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased] - 待发布

### Added

### Changed

- [T01](docs/devel/task/T01.json)：仓库改为参考 egoist.dev 的无构建个人主页：宋体与 Times New Roman 混排、杏色/暖黑双主题、36rem 单栏、双列项目卡片与 SKILLS 技能区；新增 GitHub Actions 静态 Pages 部署 workflow。

### Fixed

- [C001](docs/devel/change/C001.json)：修复 SKILLS 技能轮播翻页卡顿、连续点击丢失与回绕中途吸附；翻页动画改用 Web Animations API 驱动，连续点击合并为一次连续过渡，卡片宽度改取布局宽度以消除 hover 缩放导致的偏移。

<!-- 只添加真实变更并关联真实 C/T 卡，不预登记示例卡或发布版本。
真实卡片建立后的链接语法示例：
- [T01](docs/devel/task/T01.json)：交付的能力。
- [C001](docs/devel/change/C001.json)：修复的缺陷。
填写实际发布日期和批次，归档后同步卡片路由。
-->
