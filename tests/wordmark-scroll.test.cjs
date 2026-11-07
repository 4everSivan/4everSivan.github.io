// 回归测试: 点击字标平滑回顶, 即使在 reduced-motion 下也不是瞬时跳转.
// 运行: node tests/wordmark-scroll.test.cjs
// 依赖本机 Chrome, 可用 CHROME_PATH 覆盖; 找不到 Chrome 时跳过并以 0 退出.

const { spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const CHROME =
  process.env.CHROME_PATH ||
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const ROOT = path.resolve(__dirname, '..');
const PAGE_URL = 'file://' + path.join(ROOT, 'index.html');
const PORT = Number(process.env.CHROME_PORT || 9253);

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

(async () => {
  const profile = path.join(ROOT, 'local', 'chrome-wordmark-profile');
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
    await client.send('Emulation.setEmulatedMedia', {
      features: [{ name: 'prefers-reduced-motion', value: 'reduce' }],
    });
    await client.send('Page.navigate', { url: PAGE_URL });
    await sleep(900);
    // 展开引文, 让页面可滚动.
    await evaluate(client, "document.querySelector('.wordmark a').click()");
    await sleep(700);

    await evaluate(
      client,
      'window.scrollTo(0, 600); window.__marker = 1; true'
    );
    await sleep(120);
    const start = await evaluate(client, 'window.scrollY');
    check('起始滚动位置', start > 0, `scrollY=${start}`);

    await evaluate(client, "document.querySelector('.wordmark a').click()");

    const times = [80, 160, 260, 420, 620];
    const samples = [];
    let last = 0;
    for (const time of times) {
      await sleep(time - last);
      last = time;
      samples.push({ time, y: await evaluate(client, 'window.scrollY') });
    }

    const early = samples[0];
    check(
      '点击后不是瞬时跳转',
      early.y > 0 && early.y < start,
      `80ms scrollY=${early.y}, 起始=${start}`
    );
    check(
      '滚动位置持续下降',
      samples.every((sample, index) =>
        index === 0 ? true : sample.y <= samples[index - 1].y
      ),
      samples.map((sample) => `${sample.time}ms:${sample.y}`).join(' ')
    );
    check('最终回到顶部', samples[samples.length - 1].y === 0, `scrollY=${samples[samples.length - 1].y}`);
    check(
      '哈希更新为 #top',
      (await evaluate(client, 'window.location.hash')) === '#top',
      await evaluate(client, 'window.location.hash')
    );
    check(
      '未发生页面重载',
      (await evaluate(client, 'window.__marker')) === 1,
      ''
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
