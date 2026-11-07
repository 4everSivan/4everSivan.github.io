// 回归测试: 引文默认折叠、页面从 PROJECTS 开始、点击字标展开.
// 运行: node tests/verse-panel.test.cjs
// 依赖本机 Chrome, 可用 CHROME_PATH 覆盖; 找不到 Chrome 时跳过并以 0 退出.

const { spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const CHROME =
  process.env.CHROME_PATH ||
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const ROOT = path.resolve(__dirname, '..');
const PAGE_URL = 'file://' + path.join(ROOT, 'index.html');
const PORT = Number(process.env.CHROME_PORT || 9254);

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
  const panel = document.querySelector('[data-verse-panel]');
  const verse = document.querySelector('.verse');
  const projects = document.querySelector('.projects');
  const panelRect = panel.getBoundingClientRect();
  const verseRect = verse.getBoundingClientRect();
  const projectsRect = projects.getBoundingClientRect();
  return {
    panelHeight: Math.round(panelRect.height * 10) / 10,
    verseHeight: Math.round(verseRect.height * 10) / 10,
    verseTop: Math.round(verseRect.top * 10) / 10,
    projectsTop: Math.round(projectsRect.top * 10) / 10,
    ariaHidden: panel.getAttribute('aria-hidden'),
    isOpen: panel.classList.contains('is-open'),
    gridRows: getComputedStyle(panel).gridTemplateRows,
    scrollY: Math.round(window.scrollY),
    innerHeight: window.innerHeight,
  };
})()
`;

(async () => {
  const profile = path.join(ROOT, 'local', 'chrome-verse-profile');
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
    await sleep(900);

    const collapsed = await evaluate(client, READ);
    check(
      '引文默认折叠',
      collapsed.panelHeight <= 1,
      `panelHeight=${collapsed.panelHeight}, verseLayoutHeight=${collapsed.verseHeight}`
    );
    check(
      '引文对辅助技术隐藏',
      collapsed.ariaHidden === 'true',
      `aria-hidden=${collapsed.ariaHidden}`
    );
    check(
      '默认从 PROJECTS 开始',
      collapsed.projectsTop >= 0 && collapsed.projectsTop < collapsed.innerHeight,
      `projectsTop=${collapsed.projectsTop}, innerHeight=${collapsed.innerHeight}`
    );
    check(
      '顶部无法继续上翻',
      collapsed.scrollY === 0 &&
        (await evaluate(client, 'window.scrollTo(0, -200); window.scrollY')) === 0,
      `scrollY=${collapsed.scrollY}`
    );

    await evaluate(client, "document.querySelector('.wordmark a').click()");
    await sleep(750);
    const opened = await evaluate(client, READ);
    check(
      '点击字标展开引文',
      opened.isOpen && opened.ariaHidden === null && opened.panelHeight > 100,
      `isOpen=${opened.isOpen}, aria-hidden=${opened.ariaHidden}, panelHeight=${opened.panelHeight}`
    );
    check(
      '引文在视口内可见',
      opened.verseTop >= 0 && opened.verseTop < opened.innerHeight,
      `verseTop=${opened.verseTop}, innerHeight=${opened.innerHeight}`
    );

    await evaluate(client, 'window.scrollTo(0, 600)');
    await sleep(200);
    await evaluate(client, "document.querySelector('.wordmark a').click()");
    await sleep(750);
    const afterScroll = await evaluate(client, READ);
    check(
      '滚动后点击字标回到顶部',
      afterScroll.scrollY === 0,
      `scrollY=${afterScroll.scrollY}`
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
