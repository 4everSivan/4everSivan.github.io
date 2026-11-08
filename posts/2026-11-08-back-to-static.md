---
title: 回到静态：把个人主页迁到 GitHub Pages
date: 2026-11-08
summary: 抛弃 Hugo 与整套主题体系，回到一个 index.html：36rem 单栏、零运行时依赖，部署只剩一次 git push。
---

这个仓库曾经是一个 Hugo 文档站：主题、短代码、内容流水线一应俱全。改版之后只剩一个 `index.html`、一份样式和一段渐进增强脚本[^why]。

## 为什么不用静态站点生成器

个人主页的复杂度撑不起一条构建管线：

- 页面只有一个，日常改动是字号、间距与动效；
- 没有外部依赖，冷启动即终态；
- **可维护性**优先于功能密度。

> 留白即分隔，卡片自身承担边界。
> —— 01-博客首页 设计基线

## 部署的关键一步

Actions 里先跑治理校验，再构建文章、发布产物：

```yaml
- name: Verify governance and structure
  run: ./scripts/adflow-verify --json

- name: Build site
  run: python3 scripts/build_posts.py --out public
```

本地预览不需要装任何全局包：

```bash
python3 -m venv local/venv
local/venv/bin/pip install markdown pygments
local/venv/bin/python scripts/build_posts.py --out local/site
open local/site/index.html
```

### 回归测试

| 环节 | 命令 | 说明 |
|---|---|---|
| 门禁 | `scripts/adflow-verify` | Exit 0 才可发布 |
| 回归 | `node tests/*.test.cjs` | 12 个 CDP 用例文件 |
| 构建 | `scripts/build_posts.py` | md 源生成文章页 |

---

至此，*写作*与*发布*重新变成同一件事：写一篇 [Markdown](https://daringfireball.net/projects/markdown/)，推一次 main。

[^why]: 准确说，是「运行时零依赖」：构建文章页仍然用到 `markdown` 与 `pygments`，但它们只活在构建期，浏览器永远看不到。
