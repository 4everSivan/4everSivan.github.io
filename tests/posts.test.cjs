// 回归测试: 博客文章构建管线与产物 (T02).
// 运行: node tests/posts.test.cjs
// 依赖本机 Chrome 与 python3(markdown+pygments 或 local/venv); 缺失时跳过并以 0 退出.

const { spawn, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const CHROME =
  process.env.CHROME_PATH ||
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const ROOT = path.resolve(__dirname, '..');
const PORT = Number(process.env.POSTS_TEST_PORT || process.env.CHROME_PORT || 9244);
const OUT_DIR = path.join(ROOT, 'local', 'site-test');
const POST_SLUG = '2026-11-08-machine-learning-intro';

const VENV_PYTHON = path.join(ROOT, 'local', 'venv', 'bin', 'python');
const PYTHON = fs.existsSync(VENV_PYTHON) ? VENV_PYTHON : 'python3';

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

function chromeAvailable() {
  return fs.existsSync(CHROME);
}

function pythonDepsAvailable() {
  const probe = spawnSync(PYTHON, ['-c', 'import markdown, pygments'], {
    encoding: 'utf8',
  });
  return probe.status === 0;
}

if (!chromeAvailable() || !pythonDepsAvailable()) {
  console.log(
    `SKIP: ${!chromeAvailable() ? `未找到 Chrome (${CHROME})` : 'python 缺 markdown/pygments'}.`
  );
  process.exit(0);
}

async function waitForDebugger() {
  for (let attempt = 0; attempt < 100; attempt += 1) {
    try {
      const response = await fetch(`http://127.0.0.1:${PORT}/json/version`);
      if (response.ok) {
        return;
      }
    } catch (error) {
      // Chrome 尚未就绪.
    }
    await sleep(100);
  }
  throw new Error('Chrome debug endpoint not ready');
}

async function openPageTarget() {
  const response = await fetch(`http://127.0.0.1:${PORT}/json/new?about:blank`, {
    method: 'PUT',
  });
  return response.json();
}

function connect(url) {
  const socket = new WebSocket(url);
  const pending = new Map();
  let nextId = 0;
  const ready = new Promise((resolve, reject) => {
    socket.onopen = resolve;
    socket.onerror = reject;
  });
  socket.onmessage = (event) => {
    const message = JSON.parse(event.data);
    const entry = pending.get(message.id);
    if (!entry) {
      return;
    }
    pending.delete(message.id);
    if (message.error) {
      entry.reject(new Error(JSON.stringify(message.error)));
    } else {
      entry.resolve(message.result);
    }
  };
  function send(method, params = {}) {
    const id = (nextId += 1);
    socket.send(JSON.stringify({ id, method, params }));
    return new Promise((resolve, reject) => {
      pending.set(id, { resolve, reject });
    });
  }
  return { ready, send, close: () => socket.close() };
}

async function evaluate(client, expression) {
  const result = await client.send('Runtime.evaluate', {
    expression,
    returnByValue: true,
    awaitPromise: true,
  });
  if (result.exceptionDetails) {
    throw new Error(JSON.stringify(result.exceptionDetails));
  }
  return result.result.value;
}

const results = [];
function check(name, passed, detail) {
  results.push({ name, passed, detail });
  console.log(`${passed ? 'PASS' : 'FAIL'}  ${name}${detail ? ' — ' + detail : ''}`);
}

(async () => {
  // ---------- 构建 ----------
  fs.rmSync(OUT_DIR, { recursive: true, force: true });
  const build = spawnSync(
    PYTHON,
    [path.join(ROOT, 'scripts', 'build_posts.py'), '--out', OUT_DIR],
    { encoding: 'utf8' }
  );
  check(
    '构建退出码 0 且产物齐备',
    build.status === 0 &&
      fs.existsSync(path.join(OUT_DIR, 'index.html')) &&
      fs.existsSync(path.join(OUT_DIR, 'assets', 'css', 'style.css')) &&
      fs.existsSync(path.join(OUT_DIR, 'posts', POST_SLUG, 'index.html')),
    build.status === 0 ? build.stdout.trim() : build.stderr.trim()
  );

  const sourceIndex = fs.readFileSync(path.join(ROOT, 'index.html'), 'utf8');
  const builtIndex = fs.readFileSync(path.join(OUT_DIR, 'index.html'), 'utf8');
  const postHtml = fs.readFileSync(
    path.join(OUT_DIR, 'posts', POST_SLUG, 'index.html'),
    'utf8'
  );

  check(
    '仓库源首页占位不被污染',
    sourceIndex.includes('blog-placeholder') &&
      !sourceIndex.includes('post-list--summary'),
    `placeholder=${sourceIndex.includes('blog-placeholder')}, list=${sourceIndex.includes('post-list--summary')}`
  );
  check(
    '产物首页注入 V3 列表且无占位',
    builtIndex.includes('post-list--summary') && !builtIndex.includes('blog-placeholder'),
    ''
  );

  // 缺 summary 的文章必须让构建失败 (临时目录, 不触碰仓库).
  const badPosts = fs.mkdtempSync(path.join(os.tmpdir(), 'badposts-'));
  fs.writeFileSync(
    path.join(badPosts, 'bad.md'),
    '---\ntitle: 缺摘要\ndate: 2026-11-08\n---\n\n正文\n'
  );
  const badBuild = spawnSync(
    PYTHON,
    [
      path.join(ROOT, 'scripts', 'build_posts.py'),
      '--posts-dir',
      badPosts,
      '--out',
      path.join(badPosts, 'out'),
    ],
    { encoding: 'utf8' }
  );
  fs.rmSync(badPosts, { recursive: true, force: true });
  check(
    '缺 summary 时构建报错退出',
    badBuild.status !== 0 && badBuild.stderr.includes('summary'),
    `exit=${badBuild.status}`
  );

  // ---------- 浏览器 ----------
  const profile = path.join(ROOT, 'local', 'chrome-posts-profile');
  fs.rmSync(profile, { recursive: true, force: true });
  const chrome = spawn(
    CHROME,
    [
      '--headless=new',
      '--hide-scrollbars',
      '--allow-file-access-from-files',
      `--remote-debugging-port=${PORT}`,
      `--user-data-dir=${profile}`,
      'about:blank',
    ],
    { stdio: 'ignore' }
  );

  try {
    await waitForDebugger();
    const target = await openPageTarget();
    const client = connect(target.webSocketDebuggerUrl);
    await client.ready;
    await client.send('Page.enable');
    await client.send('Runtime.enable');
    await client.send('Emulation.setEmulatedMedia', {
      features: [{ name: 'prefers-color-scheme', value: 'light' }],
    });
    await client.send('Emulation.setDeviceMetricsOverride', {
      width: 1440,
      height: 900,
      deviceScaleFactor: 1,
      mobile: false,
    });

    await client.send('Page.navigate', {
      url: 'file://' + path.join(OUT_DIR, 'index.html'),
    });
    await sleep(900);

    const home = await evaluate(
      client,
      `(() => {
        const item = document.querySelector('.blogs .post-list .post-item');
        const style = item ? getComputedStyle(item) : null;
        const summary = document.querySelector('.post-summary');
        return {
          hasList: Boolean(document.querySelector('.blogs ul.post-list.post-list--summary')),
          hasPlaceholder: Boolean(document.querySelector('.blog-placeholder')),
          itemDisplay: style ? style.display : '',
          title: document.querySelector('.post-title-link')?.textContent || '',
          href: document.querySelector('.post-title-link')?.getAttribute('href') || '',
          date: document.querySelector('.post-date')?.textContent || '',
          summary: summary ? summary.textContent : '',
          clamp: summary ? getComputedStyle(summary).webkitLineClamp : '',
        };
      })()`
    );
    check('首页 BLOGS 为 V3 列表且占位移除', home.hasList && !home.hasPlaceholder, '');
    check('列表项为摘要式网格布局', home.itemDisplay === 'grid', home.itemDisplay);
    check(
      '首项标题/链接/日期/摘要来自 frontmatter',
      home.title === '机器学习入门：核心范式、数学直觉与可运行代码' &&
        home.href === `posts/${POST_SLUG}/` &&
        home.date === '2026-11' &&
        home.summary.length > 10,
      `title=${home.title}, href=${home.href}, date=${home.date}`
    );
    check('摘要两行内截断', home.clamp === '2', `clamp=${home.clamp}`);

    await client.send('Page.navigate', {
      url: 'file://' + path.join(OUT_DIR, 'posts', POST_SLUG, 'index.html'),
    });
    await sleep(900);

    const post = await evaluate(
      client,
      `(() => {
        const read = (selector, prop) => {
          const node = document.querySelector(selector);
          return node ? getComputedStyle(node)[prop] : '';
        };
        const keyword = document.querySelector(
          '.codehilite .nt, .codehilite .k, .codehilite .kn, .codehilite .ow'
        );
        return {
          h1Count: document.querySelectorAll('h1').length,
          title: document.querySelector('h1.post-title')?.textContent || '',
          time: document.querySelector('time')?.getAttribute('datetime') || '',
          bodySize: read('.md-body', 'fontSize'),
          h2Size: read('.md-body h2', 'fontSize'),
          // h3 排印探针: 文章可以不含 ###, 注入临时节点量 CSS 契约后移除.
          h3Size: (() => {
            const probe = document.createElement('h3');
            document.querySelector('.md-body').appendChild(probe);
            const size = getComputedStyle(probe).fontSize;
            probe.remove();
            return size;
          })(),
          h1IsArticle: Boolean(document.querySelector('article h1')),
          wordmarkTag: document.querySelector('.masthead .wordmark')?.tagName || '',
          wordmarkHref: document.querySelector('.masthead .wordmark a')?.getAttribute('href') || '',
          cssHref: document.querySelector('link[rel="stylesheet"]')?.getAttribute('href') || '',
          scriptSrc: document.querySelector('script[src*="main.js"]')?.getAttribute('src') || '',
          highlightSpans: document.querySelectorAll('.codehilite span').length,
          keywordColor: keyword ? getComputedStyle(keyword).color : '',
          mathSvgs: document.querySelectorAll('.md-body .md-math svg').length,
          mathBlocks: document.querySelectorAll('.md-body .md-math-block').length,
          mathDollars: document.querySelector('.md-body').textContent.split('$').length - 1,
          mathColor: (() => {
            const node = document.querySelector('.md-body .md-math');
            return node ? getComputedStyle(node).color : '';
          })(),
          hasFootnoteRef: Boolean(document.querySelector('.footnote-ref')),
          hasBackref: Boolean(document.querySelector('.footnote-backref')),
          hasTag: /class="[^"]*tag/.test(document.documentElement.outerHTML),
          navBack: document.querySelector('.post-nav a[href="../../"]')?.textContent || '',
          theme: document.documentElement.getAttribute('data-theme'),
        };
      })()`
    );
    check(
      '文章标题为唯一 h1, 字标降为链接',
      post.h1Count === 1 &&
        post.h1IsArticle &&
        post.title === '机器学习入门：核心范式、数学直觉与可运行代码' &&
        post.wordmarkTag === 'DIV' &&
        post.wordmarkHref === '../../',
      `h1=${post.h1Count}, wordmark=${post.wordmarkTag}@${post.wordmarkHref}`
    );
    check('日期 time[datetime]', post.time === '2026-11-08', post.time);
    check(
      '正文 15px, h2 17.5px, h3 15.5px',
      post.bodySize === '15px' &&
        post.h2Size === '17.5px' &&
        post.h3Size === '15.5px',
      `body=${post.bodySize}, h2=${post.h2Size}, h3=${post.h3Size}`
    );
    check(
      '资源相对路径且 ?v= 与首页一致',
      post.cssHref.startsWith('../../assets/css/style.css?v=') &&
        post.scriptSrc.startsWith('../../assets/js/main.js?v=') &&
        post.cssHref === builtIndex.match(/href="([^"]*style.css\?v=[0-9a-z]+)"/)[1].replace('assets/', '../../assets/'),
      `css=${post.cssHref}`
    );
    check(
      '代码高亮: Pygments token 生效且关键字为强调橘',
      post.highlightSpans > 10 && post.keywordColor === 'rgb(155, 58, 36)',
      `spans=${post.highlightSpans}, keyword=${post.keywordColor}`
    );
    check(
      '数学公式: 内联 SVG 齐备且无原始 $ 残留 (C015)',
      post.mathSvgs >= 60 && post.mathBlocks === 4 && post.mathDollars === 0,
      `svgs=${post.mathSvgs}, blocks=${post.mathBlocks}, dollars=${post.mathDollars}`
    );
    check(
      '数学公式颜色跟随墨色令牌 (浅色)',
      post.mathColor === 'rgb(33, 26, 21)',
      post.mathColor
    );
    check(
      '脚注: 上标与回跳齐备',
      post.hasFootnoteRef && post.hasBackref,
      ''
    );
    check('不出现标签元素', !post.hasTag, '');
    check('页脚含返回首页', post.navBack === '返回首页', post.navBack);

    // 主题切换在文章页仍生效 (main.js 已加载; 换主题经 startViewTransition 异步提交).
    const toggled = await evaluate(
      client,
      `(async () => {
        document.querySelector('.theme-toggle').click();
        await new Promise((resolve) => setTimeout(resolve, 400));
        const bg = getComputedStyle(document.body).backgroundColor;
        const mathColor = getComputedStyle(document.querySelector('.md-body .md-math')).color;
        return { theme: document.documentElement.getAttribute('data-theme'), bg, mathColor };
      })()`
    );
    check(
      '文章页主题切换生效 (暖黑纸面)',
      toggled.theme === 'dark' && toggled.bg === 'rgb(23, 19, 15)',
      `theme=${toggled.theme}, bg=${toggled.bg}`
    );
    check(
      '数学公式颜色跟随墨色令牌 (深色)',
      toggled.mathColor === 'rgb(236, 228, 216)',
      toggled.mathColor
    );

    // 窄屏无横向滚动.
    await client.send('Emulation.setDeviceMetricsOverride', {
      width: 390,
      height: 844,
      deviceScaleFactor: 2,
      mobile: true,
    });
    await sleep(400);
    const narrow = await evaluate(
      client,
      `(() => ({
        overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
      }))()`
    );
    check('文章页 390px 无横向滚动', narrow.overflow <= 0, `diff=${narrow.overflow}`);

    client.close();
  } finally {
    chrome.kill('SIGTERM');
  }

  const failed = results.filter((item) => !item.passed);
  console.log(`\n${results.length - failed.length}/${results.length} 项通过`);
  if (failed.length) {
    process.exitCode = 1;
  }
})().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
