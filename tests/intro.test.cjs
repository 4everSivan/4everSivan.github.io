// 回归测试: 页面打开时顶栏与各区块从上到下依次淡入.
// 运行: node tests/intro.test.cjs
// 依赖本机 Chrome, 可用 CHROME_PATH 覆盖; 找不到 Chrome 时跳过并以 0 退出.

const { spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const CHROME =
  process.env.CHROME_PATH ||
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const ROOT = path.resolve(__dirname, '..');
const PAGE_URL = 'file://' + path.join(ROOT, 'index.html');
const PORT = Number(process.env.CHROME_PORT || 9256);

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
  const animations = document
    .getAnimations()
    .filter((animation) => animation.animationName === 'intro')
    .map((animation) => ({
      target:
        animation.effect && animation.effect.target
          ? animation.effect.target.className
          : '',
      delay:
        animation.effect && animation.effect.getTiming
          ? animation.effect.getTiming().delay
          : null,
    }))
    .sort((a, b) => a.delay - b.delay);
  const opacity = (selector) =>
    getComputedStyle(document.querySelector(selector)).opacity;
  return {
    animations,
    reduced: window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    opacities: {
      masthead: opacity('.masthead'),
      projects: opacity('.projects'),
      skills: opacity('.skills'),
      blogs: opacity('.blogs'),
      foot: opacity('.foot'),
    },
  };
})()
`;

(async () => {
  const profile = path.join(ROOT, 'local', 'chrome-intro-profile');
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
      features: [{ name: 'prefers-reduced-motion', value: 'reduce' }],
    });
    await client.send('Emulation.setDeviceMetricsOverride', {
      width: 1440,
      height: 900,
      deviceScaleFactor: 1,
      mobile: false,
    });
    await client.send('Page.navigate', { url: PAGE_URL });
    await sleep(80);

    const early = await evaluate(client, READ);
    check(
      '存在 4 段入场动画',
      early.animations.length === 4,
      early.animations.map((item) => `${item.target}:${item.delay}`).join(', ')
    );
    const delays = early.animations.map((item) => item.delay);
    check(
      '延迟从上到下递增',
      JSON.stringify(delays) === JSON.stringify([80, 160, 240, 320]),
      JSON.stringify(delays)
    );
    check(
      'reduced-motion 下仍保留入场',
      early.reduced === true && early.animations.length === 4,
      `reduced=${early.reduced}`
    );
    check(
      '顶栏不参与动画',
      early.opacities.masthead === '1' && early.opacities.foot === '0',
      JSON.stringify(early.opacities)
    );

    await sleep(1100);
    const settled = await evaluate(client, READ);
    check(
      '入场结束后全部不透明',
      Object.values(settled.opacities).every((value) => value === '1'),
      JSON.stringify(settled.opacities)
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
