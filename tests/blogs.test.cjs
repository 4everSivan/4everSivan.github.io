// 回归测试: BLOGS 空状态区块的结构、样式与位置.
// 运行: node tests/blogs.test.cjs
// 依赖本机 Chrome, 可用 CHROME_PATH 覆盖; 找不到 Chrome 时跳过并以 0 退出.

const { spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const CHROME =
  process.env.CHROME_PATH ||
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const ROOT = path.resolve(__dirname, '..');
const PAGE_URL = 'file://' + path.join(ROOT, 'index.html');
const PORT = Number(process.env.CHROME_PORT || 9248);

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

if (!fs.existsSync(CHROME)) {
  console.log(`SKIP: 未找到 Chrome (${CHROME}), 可用 CHROME_PATH 指定.`);
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

const READ = `
(() => {
  const blogs = document.querySelector('.blogs');
  const skills = document.querySelector('.skills');
  const footer = document.querySelector('.foot');
  if (!blogs) {
    return { exists: false };
  }
  const title = blogs.querySelector('.section-title');
  const placeholder = blogs.querySelector('.blog-placeholder');
  const icon = blogs.querySelector('.blog-placeholder-icon');
  const svg = icon && icon.querySelector('svg');
  const text = blogs.querySelector('.blog-placeholder-text');
  const placeholderStyle = placeholder ? getComputedStyle(placeholder) : null;
  const iconStyle = icon ? getComputedStyle(icon) : null;
  const textStyle = text ? getComputedStyle(text) : null;
  const blogsStyle = getComputedStyle(blogs);
  return {
    exists: true,
    id: blogs.id,
    title: title ? title.textContent : null,
    titleTag: title ? title.tagName : null,
    text: text ? text.textContent : null,
    textColor: textStyle ? textStyle.color : null,
    placeholderPaddingTop: placeholderStyle ? placeholderStyle.paddingTop : null,
    iconWidth: iconStyle ? iconStyle.width : null,
    iconHeight: iconStyle ? iconStyle.height : null,
    iconBorderStyle: iconStyle ? iconStyle.borderStyle : null,
    iconBorderWidth: iconStyle ? iconStyle.borderWidth : null,
    iconBackground: iconStyle ? iconStyle.backgroundColor : null,
    iconColor: iconStyle ? iconStyle.color : null,
    iconPaths: svg ? svg.querySelectorAll('path').length : 0,
    links: blogs.querySelectorAll('a').length,
    marginTop: blogsStyle.marginTop,
    afterSkills:
      !!(skills.compareDocumentPosition(blogs) & Node.DOCUMENT_POSITION_FOLLOWING),
    beforeFooter:
      !!(blogs.compareDocumentPosition(footer) & Node.DOCUMENT_POSITION_FOLLOWING),
    noHorizontalScroll:
      document.documentElement.scrollWidth <= window.innerWidth + 1,
    innerWidth: window.innerWidth,
  };
})()
`;

(async () => {
  const profile = path.join(ROOT, 'local', 'chrome-blogs-test-profile');
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
    await client.send('Page.navigate', { url: PAGE_URL });
    await sleep(1000);

    const light = await evaluate(client, READ);
    check('BLOGS 区块存在', light.exists && light.id === 'blogs', `id=${light.id}`);
    check(
      '位于 SKILLS 与页脚之间',
      light.afterSkills && light.beforeFooter,
      `afterSkills=${light.afterSkills}, beforeFooter=${light.beforeFooter}`
    );
    check(
      '区块标题为 BLOGS',
      light.title === 'BLOGS' && light.titleTag === 'H2',
      `${light.titleTag} ${light.title}`
    );
    check(
      '占位文案为 COMING SOON',
      light.text === 'COMING SOON',
      light.text
    );
    check(
      '羽毛笔无边框 22px',
      light.iconWidth === '22px' &&
        light.iconHeight === '22px' &&
        light.iconBorderStyle === 'none' &&
        light.iconBackground === 'rgba(0, 0, 0, 0)',
      `${light.iconWidth} / ${light.iconBorderStyle} / ${light.iconBackground}`
    );
    check('占位整体下移', light.placeholderPaddingTop === '24px', light.placeholderPaddingTop);
    check('图标为羽毛笔线稿', light.iconPaths === 3, `${light.iconPaths} 条路径`);
    check('区块内无链接', light.links === 0, `${light.links} 个链接`);
    check('区块上间距 64px', light.marginTop === '64px', light.marginTop);
    check('无横向滚动', light.noHorizontalScroll, `width=${light.innerWidth}`);
    check(
      '浅色文案为次级灰',
      light.textColor === 'rgb(111, 96, 84)',
      light.textColor
    );
    check(
      '浅色图标为次级墨',
      light.iconColor === 'rgb(93, 80, 69)',
      light.iconColor
    );

    await evaluate(client, "document.querySelector('.theme-toggle').click()");
    await sleep(250);
    const dark = await evaluate(client, READ);
    check(
      '深色文案为次级灰',
      dark.textColor === 'rgb(140, 127, 112)',
      dark.textColor
    );
    check(
      '深色图标为次级墨',
      dark.iconColor === 'rgb(181, 168, 152)',
      dark.iconColor
    );

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
