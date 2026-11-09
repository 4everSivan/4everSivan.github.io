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
    import markdown
    from markdown.extensions.codehilite import CodeHiliteExtension
    from pygments.formatters import HtmlFormatter
except ImportError:
    print("缺少依赖: pip install markdown pygments", file=sys.stderr)
    sys.exit(2)

ROOT = Path(__file__).resolve().parent.parent
REQUIRED_KEYS = ("title", "date", "summary")
SUMMARY_MAX_CHARS = 96
HOME_LIST_LIMIT = 5

MD_EXTENSIONS = ["extra", "footnotes"]


class InlineLineNumbers(CodeHiliteExtension):
    """行号内联在每行行首.

    codehilite 的 linenums 配置只解析布尔值, 无法从外部传 'inline',
    因此在子类构造完成后直接写入 Pygments 支持的 inline 模式.
    """

    def __init__(self, **kwargs):
        kwargs.pop("linenums", None)
        super().__init__(**kwargs)
        self.config["linenums"] = ["inline", self.config["linenums"][1]]


class LanguageFormatter(HtmlFormatter):
    """在代码块外层写入语言标签, 供左上角静态展示."""

    def __init__(self, **options):
        self.label = options.pop("lang_str", "")
        prefix = "language-"
        if self.label.startswith(prefix):
            self.label = self.label[len(prefix) :]
        super().__init__(**options)

    def _wrap_div(self, inner):
        label = html.escape(self.label)
        yield 0, (
            f'<div class="{html.escape(self.cssclass)}" '
            f'data-language="{label}">'
            f'<span class="code-lang">{label}</span>'
        )
        yield from inner
        yield 0, "</div>\n"


def markdown_extensions():
    return [
        *MD_EXTENSIONS,
        InlineLineNumbers(
            guess_lang=False,
            css_class="codehilite",
            pygments_formatter=LanguageFormatter,
        ),
    ]


# 数学公式 (C015): 行内 $…$ / 整行 $$…$$ 构建期经 mathtext 渲染为内联 SVG.
MATH_COLOR = "#211a15"  # 渲染色, 输出前改写为 currentColor 跟随主题令牌
SENTINEL_MARK = "\ue000{}\ue001"  # 私用区字符, markdown 不会转义

# mathtext 字形定义: <path id="字体-码位" d="…" transform="…"/>, 同 id 同形 (C022)
MATH_GLYPH = re.compile(r'<path id="([^"]+)" d="[^"]*"[^>]*/>')


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


def render_math_svg(expr, display, plt, glyphs):
    """单条公式 → 内联 SVG 字符串（fill=currentColor, pt→px，见设计 4.2 规则 14）.

    字形 <path> 定义按 id 收进 glyphs, 由文章页统一输出一次 (C022).
    """
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
    # 内联 SVG 里的 <style> 在 HTML 中作用于整页, 会把页面图标的
    # stroke-linecap 改成 butt; 公式只用填充, 直接移除.
    svg = re.sub(r"<style[^>]*>.*?</style>", "", svg, flags=re.DOTALL)
    # 透明背景矩形与 figure_1/text_1 等分组 id 每个公式都一样, 保留会整页重复.
    svg = re.sub(r'<g id="patch_\d+">.*?</g>', "", svg, flags=re.DOTALL)
    svg = re.sub(r'<g id="[a-z]+_\d+">', "<g>", svg)

    def hoist(match):
        known = glyphs.setdefault(match.group(1), match.group(0))
        if known != match.group(0):
            raise ValueError(f"字形 {match.group(1)} 定义不一致")
        return ""

    svg = MATH_GLYPH.sub(hoist, svg)
    svg = re.sub(r"<defs>\s*</defs>", "", svg)
    svg = re.sub(r">\s+<", "><", svg)
    # pt 尺寸换算 px (1pt = 4/3px), 保留 viewBox 供窄屏等比收缩
    def px(match):
        return f'{match.group(1)}="{float(match.group(2)) * 4 / 3:.1f}"'

    svg = re.sub(r'\b(width|height)="([\d.]+)pt"', px, svg, count=2)
    svg = svg.replace(MATH_COLOR, "currentColor")
    cls = "md-math md-math-block" if display else "md-math"
    # SVG 本身没有可读文本, 以 TeX 原文作为辅助技术的替代文本.
    label = html.escape(expr)
    return f'<span class="{cls}" role="img" aria-label="{label}">{svg.strip()}</span>'


def math_defs_html(glyphs):
    """全文公式共用的字形定义: 零尺寸隐藏 SVG, 供各公式 <use> 引用."""
    if not glyphs:
        return ""
    return (
        '<svg class="md-math-defs" aria-hidden="true" focusable="false"><defs>'
        + "".join(glyphs.values())
        + "</defs></svg>\n"
    )


def extract_math(body, source):
    """提取公式渲染为 SVG, 正文以哨兵占位.

    返回 (正文, {哨兵: svg html}, 字形定义 html).
    """
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
        return body, {}, ""

    try:
        import matplotlib

        matplotlib.use("Agg")
        # 默认 dejavusans 与正文 Times/宋体冲突; STIX 为 Times 风格衬线,
        # 随 matplotlib 内置分发, CI 与本地产物一致.
        matplotlib.rcParams["mathtext.fontset"] = "stix"
        import matplotlib.pyplot as plt
    except ImportError:
        die([f"{source}: 文章含数学公式, 构建需安装 matplotlib"])

    rendered, glyphs, errors = {}, {}, []
    for token, expr, display in pending:
        try:
            rendered[token] = render_math_svg(expr, display, plt, glyphs)
        except Exception as error:  # noqa: BLE001
            errors.append(f"{source}: 公式渲染失败: {expr[:60]}… ({error})")
    if errors:
        die(errors)
    return body, rendered, math_defs_html(glyphs)


def render_markdown(body, source=""):
    """md → (正文 HTML, 公式字形定义 HTML)。

    # / ## 都落为 h2，### 落为 h3，更深收拢到 h4。
    """
    body, math_html, math_defs = extract_math(body, source)
    rendered = markdown.markdown(body, extensions=markdown_extensions())
    # 行号只是排版辅助, 不进入辅助技术阅读顺序.
    rendered = rendered.replace(
        '<span class="linenos">', '<span class="linenos" aria-hidden="true">'
    )
    for token, svg in math_html.items():
        rendered = rendered.replace(token, svg)
    rendered = re.sub(r"<h1\b", "<h2", rendered)
    rendered = re.sub(r"</h1>", "</h2>", rendered)
    rendered = re.sub(r"<h[56]\b", "<h4", rendered)
    rendered = re.sub(r"</h[56]>", "</h4>", rendered)
    return rendered, math_defs


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

    # 替换值用函数传入: 字符串形式会把标题/摘要里的 \U、\1 等当作转义解析.
    head_part = relativize(head)
    head_part = re.sub(
        r"<title>.*?</title>", lambda _: f"<title>{title} · Sivan</title>", head_part
    )
    head_part = re.sub(
        r'<meta name="description" content="[^"]*"',
        lambda _: f'<meta name="description" content="{summary}"',
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
        f'{post["math_defs"]}'
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
        + '\n\n  <div class="article-blur" aria-hidden="true"></div>\n\n  '
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
            body_html, math_defs = render_markdown(body, path.name)
            posts.append(
                {
                    "slug": path.stem,
                    "meta": meta,
                    "html": body_html,
                    "math_defs": math_defs,
                }
            )
    if errors:
        die(errors)

    # date 倒序，同日按 slug 字典序（glob 已按名排序，稳定）。
    posts.sort(key=lambda post: (post["meta"]["date"], post["slug"]), reverse=True)

    out_dir.mkdir(parents=True, exist_ok=True)
    shutil.copytree(ROOT / "assets", out_dir / "assets", dirs_exist_ok=True)

    # 列表为日期倒序, 「上一篇」是时间更早的一篇, 即列表中的下一项.
    for index, post in enumerate(posts):
        prev_post = posts[index + 1] if index + 1 < len(posts) else None
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

    # --out 可指向仓库外 (如临时目录), 此时直接显示绝对路径.
    shown = out_dir.relative_to(ROOT) if out_dir.is_relative_to(ROOT) else out_dir
    print(
        f"build_posts: {len(posts)} 篇文章 → {shown} "
        f"(首页列表 {min(len(posts), HOME_LIST_LIMIT)} 条)"
    )


if __name__ == "__main__":
    main()
