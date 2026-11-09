#!/usr/bin/env python3
"""博客文章构建（T02，设计基线 docs/devel/design/02-博客文章.md）.

输入: posts/<slug>.md（frontmatter: title/date/summary 必填，tags 忽略）
输出: --out 指定目录（本地默认 local/site，CI 传 public）:
  - index.html            首页副本，BLOGS 区块注入 V3 摘要列表
  - posts/<slug>/index.html 文章页（head/masthead/page-art/foot 抽取自 index.html）
  - assets/               整体复制

构建不访问网络、不修改源文件、同输入同输出。
依赖: markdown(extra/footnotes/codehilite) + pygments，仅构建期使用。
"""

import argparse
import datetime
import html
import io
import re
import shutil
import sys
from pathlib import Path

try:
    import markdown  # noqa: F401  仅声明依赖，实际使用在下方
except ImportError:
    print("缺少依赖: pip install markdown pygments", file=sys.stderr)
    sys.exit(2)

ROOT = Path(__file__).resolve().parent.parent
REQUIRED_KEYS = ("title", "date", "summary")
SUMMARY_MAX_CHARS = 96
HOME_LIST_LIMIT = 5

MD_EXTENSIONS = ["extra", "footnotes", "codehilite"]
MD_EXTENSION_CONFIGS = {
    "codehilite": {"guess_lang": False, "css_class": "codehilite"}
}

# 数学公式 (C015): 行内 $…$ / 整行 $$…$$ 构建期经 mathtext 渲染为内联 SVG.
MATH_COLOR = "#211a15"  # 渲染色, 输出前改写为 currentColor 跟随主题令牌
SENTINEL_MARK = "\ue000{}\ue001"  # 私用区字符, markdown 不会转义


def die(errors):
    for item in errors:
        print(f"build_posts: {item}", file=sys.stderr)
    sys.exit(1)


def parse_frontmatter(text, source):
    """解析起始 --- 块；返回 (meta dict, 正文)。契约见设计文档 3.4。"""
    if not text.startswith("---"):
        return {}, text
    lines = text.splitlines()
    try:
        end = lines[1:].index("---") + 1
    except ValueError:
        return {}, text
    meta = {}
    for line in lines[1:end]:
        match = re.match(r"^([A-Za-z][\w-]*)\s*:\s*(.*)$", line)
        if match:
            meta[match.group(1).lower()] = match.group(2).strip().strip('"').strip("'")
    return meta, "\n".join(lines[end + 1 :]) + "\n"


def validate(meta, source, errors):
    for key in REQUIRED_KEYS:
        if not meta.get(key):
            errors.append(f"{source}: frontmatter 缺少必填键 {key}")
            return None
    try:
        datetime.date.fromisoformat(meta["date"])
    except ValueError:
        errors.append(f"{source}: date 非法（需 YYYY-MM-DD）: {meta['date']}")
        return None
    summary = meta["summary"]
    if len(summary) > SUMMARY_MAX_CHARS:
        errors.append(
            f"{source}: summary 超过 {SUMMARY_MAX_CHARS} 字（现 {len(summary)}）"
        )
        return None
    return meta["date"]


def render_math_svg(expr, display, plt):
    """单条公式 → 内联 SVG 字符串（fill=currentColor, pt→px，见设计 4.2 规则 14）."""
    fig = plt.figure()
    fig.text(0, 0, f"${expr}$", fontsize=13 if display else 11, color=MATH_COLOR)
    buf = io.BytesIO()
    fig.savefig(
        buf, format="svg", bbox_inches="tight", transparent=True, pad_inches=0.02
    )
    plt.close(fig)
    svg = buf.getvalue().decode("utf-8")
    svg = re.sub(r"<\?xml[^>]*\?>|<!DOCTYPE[^>]*>", "", svg)
    svg = re.sub(r"<!--.*?-->", "", svg, flags=re.DOTALL)
    svg = re.sub(r"<metadata>.*?</metadata>", "", svg, flags=re.DOTALL)
    # pt 尺寸换算 px (1pt = 4/3px), 保留 viewBox 供窄屏等比收缩
    def px(match):
        return f'{match.group(1)}="{float(match.group(2)) * 4 / 3:.1f}"'

    svg = re.sub(r'\b(width|height)="([\d.]+)pt"', px, svg, count=2)
    svg = svg.replace(MATH_COLOR, "currentColor")
    cls = "md-math md-math-block" if display else "md-math"
    return f'<span class="{cls}">{svg.strip()}</span>'


def extract_math(body, source):
    """提取公式渲染为 SVG, 正文以哨兵占位; 返回 (正文, {哨兵: svg html})."""
    protected = []

    def stash_code(match):
        protected.append(match.group(0))
        return SENTINEL_MARK.format(f"K{len(protected) - 1}")

    # 代码块内的 $ 不是公式: 围栏块与行内代码先整体摘出
    body = re.sub(r"```.*?```", stash_code, body, flags=re.DOTALL)
    body = re.sub(r"`[^`\n]*`", stash_code, body)

    pending = []

    def stash_math(expr, display):
        expr = expr.strip()
        token = SENTINEL_MARK.format(f"M{len(pending)}")
        pending.append((token, expr, display))
        return token

    body = re.sub(
        r"\$\$(.+?)\$\$",
        lambda m: stash_math(m.group(1), True),
        body,
        flags=re.DOTALL,
    )

    def inline(m):
        expr = m.group(1)
        return stash_math(expr, False) if expr.strip() else m.group(0)

    body = re.sub(r"\$([^$\n]+?)\$", inline, body)

    # 还原代码段, 公式哨兵留待 HTML 阶段替换
    body = re.sub(
        SENTINEL_MARK.format(r"K(\d+)"), lambda m: protected[int(m.group(1))], body
    )
    if not pending:
        return body, {}

    try:
        import matplotlib

        matplotlib.use("Agg")
        # 默认 dejavusans 与正文 Times/宋体冲突; STIX 为 Times 风格衬线,
        # 随 matplotlib 内置分发, CI 与本地产物一致.
        matplotlib.rcParams["mathtext.fontset"] = "stix"
        import matplotlib.pyplot as plt
    except ImportError:
        die([f"{source}: 文章含数学公式, 构建需安装 matplotlib"])

    rendered, errors = {}, []
    for token, expr, display in pending:
        try:
            rendered[token] = render_math_svg(expr, display, plt)
        except Exception as error:  # noqa: BLE001
            errors.append(f"{source}: 公式渲染失败: {expr[:60]}… ({error})")
    if errors:
        die(errors)
    return body, rendered


def render_markdown(body, source=""):
    """md → HTML；# / ## 都落为 h2，### 落为 h3，更深收拢到 h4。"""
    body, math_html = extract_math(body, source)
    rendered = markdown.markdown(
        body, extensions=MD_EXTENSIONS, extension_configs=MD_EXTENSION_CONFIGS
    )
    for token, svg in math_html.items():
        rendered = rendered.replace(token, svg)
    rendered = re.sub(r"<h1\b", "<h2", rendered)
    rendered = re.sub(r"</h1>", "</h2>", rendered)
    rendered = re.sub(r"<h[56]\b", "<h4", rendered)
    rendered = re.sub(r"</h[56]>", "</h4>", rendered)
    return rendered


def slice_between(text, start_pat, end_pat):
    match = re.search(start_pat + r".*?" + end_pat, text, re.DOTALL)
    return match.group(0) if match else ""


def extract_layout(index_html):
    """从首页抽取文章页需要的片段与缓存版本串。"""
    head = slice_between(index_html, r"<head\b", r"</head>")
    page_art = slice_between(index_html, r'<div class="page-art"', r"</div>")
    masthead = slice_between(index_html, r'<header class="masthead">', r"</header>")
    foot = slice_between(index_html, r'<footer class="foot">', r"</footer>")
    script = slice_between(
        index_html, r'<script src="assets/js/main\.js', r"</script>"
    )
    version = ""
    match = re.search(r"assets/css/style\.css\?v=([0-9A-Za-z]+)", index_html)
    if match:
        version = match.group(1)
    missing = [
        name
        for name, part in (
            ("head", head),
            ("page-art", page_art),
            ("masthead", masthead),
            ("foot", foot),
            ("main.js 引用", script),
        )
        if not part
    ]
    return head, page_art, masthead, foot, script, version, missing


def relativize(fragment):
    """站内资源与首页锚点改写为文章页相对路径。"""
    fragment = fragment.replace('href="assets/', 'href="../../assets/')
    fragment = fragment.replace('src="assets/', 'src="../../assets/')
    return fragment


def build_post_page(post, layout, prev_post):
    head, page_art, masthead, foot, script, version, _ = layout
    title = html.escape(post["meta"]["title"])
    summary = html.escape(post["meta"]["summary"])

    head_part = relativize(head)
    head_part = re.sub(
        r"<title>.*?</title>", f"<title>{title} · Sivan</title>", head_part
    )
    head_part = re.sub(
        r'<meta name="description" content="[^"]*"',
        f'<meta name="description" content="{summary}"',
        head_part,
    )

    # 字标让位 h1 给文章标题，降为普通链接并指回首页。
    masthead_part = masthead.replace(
        '<h1 class="wordmark">', '<div class="wordmark">'
    ).replace("</h1>", "</div>", 1)
    masthead_part = masthead_part.replace(
        'href="#top"', 'href="../../" data-page-transition'
    )

    prev_link = ""
    if prev_post:
        prev_link = (
            f'<a href="../{prev_post["slug"]}/" data-page-transition>'
            f'← 上一篇：{html.escape(prev_post["meta"]["title"])}</a>'
        )
    nav = (
        f'<nav class="post-nav" aria-label="文章导航">'
        f'{prev_link}<a href="../../" data-page-transition>返回首页</a></nav>'
    )

    article = (
        f'<article class="post">\n'
        f'<header class="post-header">\n'
        f'<h1 class="post-title">{title}</h1>\n'
        f'<p class="post-meta"><time datetime="{post["meta"]["date"]}">'
        f'{post["meta"]["date"]}</time></p>\n'
        f"</header>\n"
        f'<div class="md-body">\n{post["html"]}\n</div>\n'
        f"{nav}\n"
        f"</article>"
    )

    return (
        "<!doctype html>\n"
        '<html lang="zh-CN">\n'
        + head_part
        + '\n<body id="top" class="post-page">\n'
        + '  <a class="skip-link" href="#main">跳到正文</a>\n\n  '
        + page_art
        + "\n\n  "
        + masthead_part
        + '\n\n  <div class="page">\n    <main id="main">\n      '
        + article
        + "\n    </main>\n\n    "
        + foot
        + "\n  </div>\n\n  "
        + relativize(script)
        + "\n</body>\n</html>\n"
    )


def build_home_list(posts):
    items = []
    for post in posts[:HOME_LIST_LIMIT]:
        meta = post["meta"]
        month = meta["date"][:7]
        items.append(
            f'        <li class="post-item">\n'
            f'          <a class="post-title-link" '
            f'href="posts/{post["slug"]}/" data-page-transition>'
            f"{html.escape(meta['title'])}</a>\n"
            f'          <span class="post-date">{month}</span>\n'
            f'          <p class="post-summary">{html.escape(meta["summary"])}</p>\n'
            f"        </li>"
        )
    return (
        '        <ul class="post-list post-list--summary">\n'
        + "\n".join(items)
        + "\n        </ul>"
    )


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--posts-dir", default="posts")
    parser.add_argument("--out", default="local/site")
    args = parser.parse_args()

    posts_dir = ROOT / args.posts_dir
    out_dir = ROOT / args.out
    errors = []

    index_path = ROOT / "index.html"
    index_html = index_path.read_text(encoding="utf-8")
    layout = extract_layout(index_html)
    if layout[6]:
        die([f"index.html 片段缺失: {', '.join(layout[6])}"])

    posts = []
    if posts_dir.is_dir():
        for path in sorted(posts_dir.glob("*.md")):
            meta, body = parse_frontmatter(
                path.read_text(encoding="utf-8"), path.name
            )
            date = validate(meta, path.name, errors)
            if date is None:
                continue
            posts.append(
                {
                    "slug": path.stem,
                    "meta": meta,
                    "html": render_markdown(body, path.name),
                }
            )
    if errors:
        die(errors)

    # date 倒序，同日按 slug 字典序（glob 已按名排序，稳定）。
    posts.sort(key=lambda post: (post["meta"]["date"], post["slug"]), reverse=True)

    out_dir.mkdir(parents=True, exist_ok=True)
    shutil.copytree(ROOT / "assets", out_dir / "assets", dirs_exist_ok=True)

    for index, post in enumerate(posts):
        prev_post = posts[index - 1] if index > 0 else None
        page = build_post_page(post, layout, prev_post)
        target = out_dir / "posts" / post["slug"] / "index.html"
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(page, encoding="utf-8")

    home_html = index_html
    if posts:
        placeholder = re.search(
            r'[ \t]*<div class="blog-placeholder">.*?</div>', index_html, re.DOTALL
        )
        if not placeholder:
            die(["index.html: 未找到 blog-placeholder 区块，无法注入列表"])
        home_html = index_html.replace(
            placeholder.group(0), build_home_list(posts), 1
        )
    (out_dir / "index.html").write_text(home_html, encoding="utf-8")

    print(
        f"build_posts: {len(posts)} 篇文章 → {out_dir.relative_to(ROOT)} "
        f"(首页列表 {min(len(posts), HOME_LIST_LIMIT)} 条)"
    )


if __name__ == "__main__":
    main()
