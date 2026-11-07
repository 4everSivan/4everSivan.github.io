// 回归测试: 滚动栏隐藏但页面仍可滚动.
// 运行: node tests/scroll.test.cjs
// 依赖本机 Chrome, 可用 CHROME_PATH 覆盖; 找不到 Chrome 时跳过并以 0 退出.

const { spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const CHROME =
  process.env.CHROME_PATH ||
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const ROOT = path.resolve(__dirname, '..');
const PAGE_URL = 'file://' + path.join(ROOT, 'index.html');
const PORT = Number(process.env.CHROME_PORT || 9246);

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
  const doc = document.documentElement;
  const rules = [];
  for (const sheet of Array.from(document.styleSheets)) {
    try {
      for (const rule of Array.from(sheet.cssRules)) {
        rules.push(rule);
      }
    } catch (error) {
      // 跨域样式表不可读, 忽略.
    }
  }
  const webkitRule = rules.find(
    (rule) =>
      rule.selectorText === 'html::-webkit-scrollbar' &&
      rule.style &&
      rule.style.display === 'none'
  );
  return {
    scrollbarWidth: getComputedStyle(doc).scrollbarWidth,
    hasWebkitRule: !!webkitRule,
    scrollHeight: doc.scrollHeight,
    clientHeight: doc.clientHeight,
    scrollWidth: doc.scrollWidth,
    clientWidth: doc.clientWidth,
    scrollY: window.scrollY,
  };
})()
`;

(async () => {
  const profile = path.join(ROOT, 'local', 'chrome-scroll-test-profile');
  fs.rmSync(profile, { recursive: true, force: true });
  const chrome = spawn(
    CHROME,
    [
      '--headless=new',
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
    await client.send('Emulation.setScrollbarsHidden', { hidden: false });
    await client.send('Page.navigate', { url: PAGE_URL });
    await sleep(1000);

    const before = await evaluate(client, READ);
    check(
      '滚动栏已隐藏',
      (before.scrollbarWidth === 'none' || before.scrollbarWidth === '') &&
        before.hasWebkitRule,
      `scrollbar-width=${before.scrollbarWidth}, webkit规则=${before.hasWebkitRule}`
    );
    check(
      '页面仍可纵向滚动',
      before.scrollHeight > before.clientHeight,
      `scrollHeight=${before.scrollHeight}, clientHeight=${before.clientHeight}`
    );
    check(
      '无横向滚动',
      before.scrollWidth <= before.clientWidth,
      `scrollWidth=${before.scrollWidth}, clientWidth=${before.clientWidth}`
    );

    const scrolled = await evaluate(
      client,
      `(() => {
        window.scrollTo(0, 200);
        return new Promise((resolve) => {
          requestAnimationFrame(() => {
            requestAnimationFrame(() => resolve(window.scrollY));
          });
        });
      })()`
    );
    check('滚轮/脚本滚动生效', scrolled > 0, `scrollY=${scrolled}`);

    await evaluate(client, 'window.scrollTo(0, 0)');
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
