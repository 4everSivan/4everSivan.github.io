// 回归测试: 博客列表点击切换文章页的过渡动画 (C016).
// 运行: node tests/page-transition.test.cjs
// 依赖本机 Chrome 与 python3(markdown+pygments 或 local/venv); 缺失时跳过并以 0 退出.

const { spawn, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const CHROME =
  process.env.CHROME_PATH ||
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const ROOT = path.resolve(__dirname, '..');
const PORT = Number(
  process.env.PAGE_TRANSITION_PORT || process.env.CHROME_PORT || 9252
);
const OUT_DIR = path.join(ROOT, 'local', 'page-transition-test');
const POST_SLUG = '2026-11-08-machine-learning-intro';
const HTTP_PORT = Number(process.env.PAGE_TRANSITION_HTTP_PORT || 8752);
const BASE_URL = `http://127.0.0.1:${HTTP_PORT}/`;

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

async function waitForServer() {
  for (let attempt = 0; attempt < 100; attempt += 1) {
    try {
      const response = await fetch(BASE_URL);
      if (response.ok) {
        return;
      }
    } catch (error) {
      // 静态服务器尚未就绪.
    }
    await sleep(100);
  }
  throw new Error('static server not ready');
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
  const homePath = path.join(OUT_DIR, 'index.html');
  const postPath = path.join(OUT_DIR, 'posts', POST_SLUG, 'index.html');
  check(
    '构建退出码 0 且产物齐备',
    build.status === 0 && fs.existsSync(homePath) && fs.existsSync(postPath),
    build.status === 0 ? build.stdout.trim() : build.stderr.trim()
  );
  if (build.status !== 0) {
    process.exit(1);
  }

  const builtHome = fs.readFileSync(homePath, 'utf8');
  const postHtml = fs.readFileSync(postPath, 'utf8');
  check(
    '列表标题与文章页导航带过渡标记',
    /class="post-title-link"[^>]*data-page-transition/.test(builtHome) &&
      postHtml.includes('<body id="top" class="post-page">') &&
      postHtml.includes('data-page-transition>返回首页</a>') &&
      postHtml.includes('data-page-transition aria-label="Sivan 首页"'),
    ''
  );

  // ---------- 浏览器 ----------
  const server = spawn(
    PYTHON,
    [
      '-m',
      'http.server',
      String(HTTP_PORT),
      '--bind',
      '127.0.0.1',
      '--directory',
      OUT_DIR,
    ],
    { stdio: 'ignore' }
  );
  const profile = path.join(ROOT, 'local', 'chrome-page-transition-profile');
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
    await waitForServer();
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

    await client.send('Page.navigate', { url: BASE_URL });
    await sleep(900);

    const home = await evaluate(
      client,
      `(() => {
        const page = document.querySelector('.page');
        const style = page ? getComputedStyle(page) : null;
        return {
          hasPost: Boolean(document.querySelector('.post')),
          transitionProperty: style ? style.transitionProperty : '',
          transitionDuration: style ? style.transitionDuration : '',
          leaving: document.documentElement.classList.contains('is-page-leaving'),
        };
      })()`
    );
    check(
      '首页内容层具备 200ms 淡出/位移过渡',
      home.hasPost === false &&
        home.transitionProperty.includes('opacity') &&
        home.transitionProperty.includes('transform') &&
        home.transitionDuration.split(', ').every((value) => value === '0.2s'),
      `property=${home.transitionProperty}, duration=${home.transitionDuration}`
    );

    // 修饰键点击不进入过渡, 也不阻止浏览器默认行为 (临时 preventDefault 避免测试打开新标签).
    const modifier = await evaluate(
      client,
      `(() => {
        const link = document.querySelector('.blogs .post-title-link');
        link.addEventListener('click', (event) => event.preventDefault(), { once: true });
        link.dispatchEvent(new MouseEvent('click', {
          bubbles: true, cancelable: true, button: 0, ctrlKey: true,
        }));
        return {
          leaving: document.documentElement.classList.contains('is-page-leaving'),
          path: location.pathname,
        };
      })()`
    );
    await sleep(350);
    const stillHome = await evaluate(client, `location.pathname`);
    check(
      'Ctrl/⌘ 点击保持默认导航, 不触发切页过渡',
      modifier.leaving === false && stillHome === modifier.path,
      `leaving=${modifier.leaving}`
    );

    // ---------- 首页 → 文章页 ----------
    const clicked = await evaluate(
      client,
      `(() => {
        document.querySelector('.blogs .post-title-link').click();
        return document.documentElement.classList.contains('is-page-leaving');
      })()`
    );
    check('点击列表标题立即进入离场态', clicked === true, '');

    await sleep(80);
    const mid = await evaluate(
      client,
      `(() => {
        const style = getComputedStyle(document.querySelector('.page'));
        return { opacity: Number(style.opacity), transform: style.transform };
      })()`
    );
    check(
      '离场中内容层正在淡出并上移',
      mid.opacity < 1 && mid.opacity >= 0 && /matrix\(1, 0, 0, 1, 0, -?\d/.test(mid.transform),
      `opacity=${mid.opacity}, transform=${mid.transform}`
    );

    await sleep(900);
    const post = await evaluate(
      client,
      `(() => {
        const node = document.querySelector('.post');
        const foot = document.querySelector('.foot');
        const style = node ? getComputedStyle(node) : null;
        return {
          path: location.pathname,
          bodyClass: document.body.className,
          hasPost: Boolean(node),
          animationName: style ? style.animationName : '',
          animationDuration: style ? style.animationDuration : '',
          footAnimation: foot ? getComputedStyle(foot).animationName : '',
          footDelay: foot ? getComputedStyle(foot).animationDelay : '',
          opacity: style ? style.opacity : '',
          transform: style ? style.transform : '',
          leaving: document.documentElement.classList.contains('is-page-leaving'),
        };
      })()`
    );
    check(
      '延迟后进入文章页且离场态已清除',
      post.path.endsWith(`/posts/${POST_SLUG}/`) &&
        post.bodyClass.includes('post-page') &&
        post.hasPost &&
        post.leaving === false,
      `path=${post.path}, body=${post.bodyClass}`
    );
    check(
      '文章正文与页脚以 page-enter 入场, 结束回到终态',
      post.animationName === 'page-enter' &&
        post.animationDuration === '0.56s' &&
        post.footAnimation === 'page-enter' &&
        post.footDelay === '0.08s' &&
        post.opacity === '1' &&
        (post.transform === 'none' ||
          post.transform === 'matrix(1, 0, 0, 1, 0, 0)'),
      `name=${post.animationName}, duration=${post.animationDuration}, foot=${post.footAnimation}@${post.footDelay}, opacity=${post.opacity}, transform=${post.transform}`
    );

    // ---------- 文章页 → 首页 ----------
    const backClicked = await evaluate(
      client,
      `(() => {
        document.querySelector('.post-nav a[href="../../"]').click();
        return document.documentElement.classList.contains('is-page-leaving');
      })()`
    );
    await sleep(900);
    const back = await evaluate(
      client,
      `({ path: location.pathname, hasPost: Boolean(document.querySelector('.post')) })`
    );
    check(
      '文章页「返回首页」同样走离场过渡',
      backClicked === true && back.path === '/' && !back.hasPost,
      `path=${back.path}`
    );

    // ---------- reduced-motion 下保留过渡 ----------
    await client.send('Emulation.setEmulatedMedia', {
      features: [
        { name: 'prefers-color-scheme', value: 'light' },
        { name: 'prefers-reduced-motion', value: 'reduce' },
      ],
    });
    await sleep(200);
    const reducedClick = await evaluate(
      client,
      `(() => {
        document.querySelector('.blogs .post-title-link').click();
        return document.documentElement.classList.contains('is-page-leaving');
      })()`
    );
    await sleep(900);
    const reducedPost = await evaluate(
      client,
      `(() => {
        const node = document.querySelector('.post');
        return {
          path: location.pathname,
          animation: node ? getComputedStyle(node).animationName : '',
        };
      })()`
    );
    check(
      'reduced-motion 下切页过渡仍保留',
      reducedClick === true &&
        reducedPost.path.endsWith(`/posts/${POST_SLUG}/`) &&
        reducedPost.animation === 'page-enter',
      `animation=${reducedPost.animation}`
    );

    client.close();
  } finally {
    chrome.kill('SIGTERM');
    server.kill('SIGTERM');
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
