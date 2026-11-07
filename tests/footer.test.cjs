// 回归测试: 页脚固定在视口底部与底部留白.
// 运行: node tests/footer.test.cjs
// 依赖本机 Chrome, 可用 CHROME_PATH 覆盖; 找不到 Chrome 时跳过并以 0 退出.

const { spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const CHROME =
  process.env.CHROME_PATH ||
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const ROOT = path.resolve(__dirname, '..');
const PAGE_URL = 'file://' + path.join(ROOT, 'index.html');
const PORT = Number(process.env.CHROME_PORT || 9249);

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
  const foot = document.querySelector('.foot');
  const blogs = document.querySelector('.blogs');
  const style = getComputedStyle(foot);
  const rect = foot.getBoundingClientRect();
  const blogsRect = blogs.getBoundingClientRect();
  return {
    position: style.position,
    bottom: style.bottom,
    paddingBottom: style.paddingBottom,
    footTop: Math.round(rect.top * 10) / 10,
    footBottom: Math.round(rect.bottom * 10) / 10,
    footCenterX: Math.round((rect.left + rect.width / 2) * 10) / 10,
    blogsBottom: Math.round(blogsRect.bottom * 10) / 10,
    innerWidth: window.innerWidth,
    innerHeight: window.innerHeight,
    scrollY: Math.round(window.scrollY),
    scrollHeight: document.documentElement.scrollHeight,
    noHorizontalScroll:
      document.documentElement.scrollWidth <= window.innerWidth + 1,
  };
})()
`;

(async () => {
  const profile = path.join(ROOT, 'local', 'chrome-footer-test-profile');
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
    await client.send('Emulation.setDeviceMetricsOverride', {
      width: 1440,
      height: 900,
      deviceScaleFactor: 1,
      mobile: false,
    });
    await client.send('Page.navigate', { url: PAGE_URL });
    await sleep(1000);

    const top = await evaluate(client, READ);
    check(
      '页脚固定在视口底部',
      top.position === 'fixed' && top.bottom === '0px',
      `position=${top.position}, bottom=${top.bottom}`
    );
    check(
      '顶部时页脚贴视口底边',
      Math.abs(top.footBottom - top.innerHeight) <= 1,
      `footBottom=${top.footBottom}, innerHeight=${top.innerHeight}`
    );
    check('页脚底部留白 24px', top.paddingBottom === '24px', top.paddingBottom);
    check(
      '页脚水平居中',
      Math.abs(top.footCenterX - top.innerWidth / 2) <= 1,
      `center=${top.footCenterX}, viewport=${top.innerWidth / 2}`
    );
    check('无横向滚动', top.noHorizontalScroll, `width=${top.innerWidth}`);

    const bottom = await evaluate(
      client,
      `(() => {
        window.scrollTo(0, document.documentElement.scrollHeight);
        return new Promise((resolve) => {
          requestAnimationFrame(() => {
            requestAnimationFrame(() => resolve(true));
          });
        });
      })()`
    );
    check('滚动到底部', bottom === true, '');

    const after = await evaluate(client, READ);
    check(
      '滚动后页脚仍贴视口底边',
      Math.abs(after.footBottom - after.innerHeight) <= 1,
      `footBottom=${after.footBottom}, innerHeight=${after.innerHeight}`
    );
    check(
      'BLOGS 内容不被页脚遮挡',
      after.blogsBottom <= after.footTop,
      `blogsBottom=${after.blogsBottom}, footTop=${after.footTop}`
    );

    await client.send('Emulation.setEmulatedMedia', { media: 'print' });
    await sleep(100);
    const printPosition = await evaluate(
      client,
      "getComputedStyle(document.querySelector('.foot')).position"
    );
    check('打印时页脚回到文档流', printPosition === 'static', printPosition);
    await client.send('Emulation.setEmulatedMedia', { media: '' });

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
