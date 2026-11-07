// 回归测试: 顶栏搜索栏的收起/展开、快捷键与窄屏宽度.
// 运行: node tests/search.test.cjs
// 依赖本机 Chrome, 可用 CHROME_PATH 覆盖; 找不到 Chrome 时跳过并以 0 退出.

const { spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const CHROME =
  process.env.CHROME_PATH ||
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const ROOT = path.resolve(__dirname, '..');
const PAGE_URL = 'file://' + path.join(ROOT, 'index.html');
const PORT = Number(process.env.CHROME_PORT || 9259);

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
  const search = document.querySelector('[data-search]');
  const toggle = search.querySelector('.search-toggle');
  const content = search.querySelector('.search-content');
  const input = search.querySelector('.search-input');
  const inputStyle = getComputedStyle(input);
  const theme = document.querySelector('.theme-toggle');
  const github = document.querySelector('.github-link');
  const style = getComputedStyle(search);
  const rect = search.getBoundingClientRect();
  const toggleRect = toggle.getBoundingClientRect();
  const themeRect = theme.getBoundingClientRect();
  const githubRect = github.getBoundingClientRect();
  return {
    isOpen: search.classList.contains('is-open'),
    width: Math.round(rect.width * 10) / 10,
    height: Math.round(rect.height * 10) / 10,
    borderColor: style.borderTopColor,
    background: style.backgroundColor,
    expanded: toggle.getAttribute('aria-expanded'),
    contentHidden: content.getAttribute('aria-hidden'),
    tabIndex: input.tabIndex,
    focused: document.activeElement === input,
    toggleFocused: document.activeElement === toggle,
    inputFont: inputStyle.fontFamily,
    inputColor: inputStyle.color,
    searchBeforeTheme: rect.right <= themeRect.left,
    searchGap: Math.round((themeRect.left - toggleRect.right) * 10) / 10,
    barGap: Math.round((themeRect.left - rect.right) * 10) / 10,
    themeGap: Math.round((githubRect.left - themeRect.right) * 10) / 10,
    placeholder: input.getAttribute('placeholder'),
    noHorizontalScroll:
      document.documentElement.scrollWidth <= window.innerWidth + 1,
  };
})()
`;

(async () => {
  const profile = path.join(ROOT, 'local', 'chrome-search-test-profile');
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
      '默认收缩为 28px 放大镜',
      !collapsed.isOpen &&
        collapsed.width === 28 &&
        collapsed.height === 28,
      `width=${collapsed.width}, height=${collapsed.height}`
    );
    check(
      '收起态输入框不可聚焦',
      collapsed.expanded === 'false' &&
        collapsed.contentHidden === 'true' &&
        collapsed.tabIndex === -1,
      `expanded=${collapsed.expanded}, hidden=${collapsed.contentHidden}, tab=${collapsed.tabIndex}`
    );
    check(
      '位于主题按钮左侧',
      collapsed.searchBeforeTheme,
      ''
    );
    check(
      '放大镜与主题按钮间距一致',
      Math.abs(collapsed.searchGap - collapsed.themeGap) <= 1,
      `searchGap=${collapsed.searchGap}, themeGap=${collapsed.themeGap}`
    );
    check(
      '输入文字使用主题字体与墨色',
      collapsed.inputFont.includes('Times New Roman') &&
        collapsed.inputColor === 'rgb(33, 26, 21)',
      `font=${collapsed.inputFont.slice(0, 36)}, color=${collapsed.inputColor}`
    );
    check(
      '占位文字为 Search…',
      collapsed.placeholder === 'Search…',
      collapsed.placeholder
    );

    await evaluate(client, "document.querySelector('.search-toggle').click()");
    await sleep(350);
    const opened = await evaluate(client, READ);
    check(
      '点击展开 180×28',
      opened.isOpen && opened.width === 180 && opened.height === 28,
      `width=${opened.width}, height=${opened.height}`
    );
    check(
      '展开后输入框聚焦',
      opened.expanded === 'true' &&
        opened.contentHidden === 'false' &&
        opened.tabIndex === 0 &&
        opened.focused,
      `expanded=${opened.expanded}, tab=${opened.tabIndex}, focused=${opened.focused}`
    );
    check(
      '纸面填充与强调色焦点描边',
      opened.background === 'rgb(252, 241, 226)' &&
        opened.borderColor === 'rgb(155, 58, 36)',
      `bg=${opened.background}, border=${opened.borderColor}`
    );
    check(
      '展开栏与主题按钮间距一致',
      Math.abs(opened.barGap - opened.themeGap) <= 1,
      `barGap=${opened.barGap}, themeGap=${opened.themeGap}`
    );

    await evaluate(
      client,
      "document.querySelector('.search-input').dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))"
    );
    await sleep(350);
    const closed = await evaluate(client, READ);
    check(
      'Esc 收起且不残留焦点环',
      !closed.isOpen &&
        closed.width === 28 &&
        !closed.toggleFocused &&
        !closed.focused,
      `width=${closed.width}, toggleFocused=${closed.toggleFocused}`
    );

    await evaluate(
      client,
      "document.dispatchEvent(new KeyboardEvent('keydown', { key: 'f', metaKey: true, bubbles: true, cancelable: true }))"
    );
    await sleep(350);
    const shortcut = await evaluate(client, READ);
    check(
      '⌘F 展开并聚焦',
      shortcut.isOpen && shortcut.focused && shortcut.width === 180,
      `width=${shortcut.width}, focused=${shortcut.focused}`
    );

    await evaluate(
      client,
      "document.dispatchEvent(new KeyboardEvent('keydown', { key: 'f', metaKey: true, bubbles: true, cancelable: true }))"
    );
    await sleep(350);
    const shortcutClosed = await evaluate(client, READ);
    check(
      '⌘F 再次按下收起且不残留焦点环',
      !shortcutClosed.isOpen &&
        shortcutClosed.width === 28 &&
        !shortcutClosed.toggleFocused &&
        !shortcutClosed.focused,
      `width=${shortcutClosed.width}, toggleFocused=${shortcutClosed.toggleFocused}`
    );

    await evaluate(client, "document.querySelector('.search-toggle').click()");
    await sleep(350);

    await evaluate(client, "document.querySelector('.theme-toggle').click()");
    await sleep(300);
    const dark = await evaluate(client, READ);
    check(
      '深色下纸面填充与文字颜色匹配主题',
      dark.background === 'rgb(33, 27, 22)' &&
        dark.inputColor === 'rgb(236, 228, 216)',
      `bg=${dark.background}, color=${dark.inputColor}`
    );

    await client.send('Emulation.setDeviceMetricsOverride', {
      width: 390,
      height: 844,
      deviceScaleFactor: 1,
      mobile: true,
    });
    await client.send('Page.reload');
    await sleep(900);
    await evaluate(client, "document.querySelector('.search-toggle').click()");
    await sleep(350);
    const mobile390 = await evaluate(client, READ);
    check(
      '390px 展开宽 150px 且无横向滚动',
      mobile390.width === 150 && mobile390.noHorizontalScroll,
      `width=${mobile390.width}`
    );

    await client.send('Emulation.setDeviceMetricsOverride', {
      width: 320,
      height: 800,
      deviceScaleFactor: 1,
      mobile: true,
    });
    await client.send('Page.reload');
    await sleep(900);
    await evaluate(client, "document.querySelector('.search-toggle').click()");
    await sleep(350);
    const mobile320 = await evaluate(client, READ);
    check(
      '320px 展开宽 144px 且无横向滚动',
      mobile320.width === 144 && mobile320.noHorizontalScroll,
      `width=${mobile320.width}`
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
