// 回归测试: 页脚流式贴底、底部留白与顶栏渐变遮罩.
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
  const masthead = document.querySelector('.masthead');
  const footStyle = getComputedStyle(foot);
  const mastheadStyle = getComputedStyle(masthead);
  const footRect = foot.getBoundingClientRect();
  return {
    theme: document.documentElement.getAttribute('data-theme'),
    footPosition: footStyle.position,
    footColor: footStyle.color,
    footTop: Math.round(footRect.top * 10) / 10,
    footBottom: Math.round(footRect.bottom * 10) / 10,
    footCenterX: Math.round((footRect.left + footRect.width / 2) * 10) / 10,
    mastheadPosition: mastheadStyle.position,
    mastheadBorderBottom: mastheadStyle.borderBottomWidth,
    mastheadBackground: mastheadStyle.backgroundImage,
    mastheadScrolledClass: masthead.classList.contains('is-scrolled'),
    innerWidth: window.innerWidth,
    innerHeight: window.innerHeight,
    scrollY: Math.round(window.scrollY),
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
    await client.send('Emulation.setEmulatedMedia', {
      features: [{ name: 'prefers-color-scheme', value: 'light' }],
    });
    await client.send('Page.navigate', { url: PAGE_URL });
    await sleep(1000);

    const top = await evaluate(client, READ);
    check(
      '页脚保持在文档流',
      top.footPosition === 'static',
      `position=${top.footPosition}`
    );
    check(
      '浅色页脚为灰色',
      top.footColor === 'rgb(111, 96, 84)',
      top.footColor
    );
    check(
      '顶部时页脚不在视口内',
      top.footTop >= top.innerHeight,
      `footTop=${top.footTop}, innerHeight=${top.innerHeight}`
    );
    check(
      '顶栏吸顶',
      top.mastheadPosition === 'sticky',
      `position=${top.mastheadPosition}`
    );
    check(
      '顶栏无分隔线',
      top.mastheadBorderBottom === '0px',
      `border-bottom=${top.mastheadBorderBottom}`
    );
    check(
      '顶栏为纸面色渐变遮罩',
      top.mastheadBackground.includes('linear-gradient') &&
        top.mastheadBackground.includes('rgba'),
      top.mastheadBackground.slice(0, 90)
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
      '底部时页脚贴底留白 24px',
      Math.abs(after.footBottom - (after.innerHeight - 24)) <= 1,
      `footBottom=${after.footBottom}, expected=${after.innerHeight - 24}`
    );
    check(
      '底部时页脚可见',
      after.footTop < after.innerHeight,
      `footTop=${after.footTop}`
    );
    check(
      '页脚水平居中',
      Math.abs(after.footCenterX - after.innerWidth / 2) <= 1,
      `center=${after.footCenterX}, viewport=${after.innerWidth / 2}`
    );
    check(
      '滚动后顶栏仍无分隔线',
      after.mastheadBorderBottom === '0px' && !after.mastheadScrolledClass,
      `border=${after.mastheadBorderBottom}, is-scrolled=${after.mastheadScrolledClass}`
    );

    await client.send('Emulation.setEmulatedMedia', {
      media: 'print',
      features: [{ name: 'prefers-color-scheme', value: 'light' }],
    });
    await sleep(100);
    const printBackground = await evaluate(
      client,
      "getComputedStyle(document.querySelector('.masthead')).backgroundImage"
    );
    check('打印时顶栏无背景', printBackground === 'none', printBackground);
    await client.send('Emulation.setEmulatedMedia', {
      media: '',
      features: [{ name: 'prefers-color-scheme', value: 'light' }],
    });

    await evaluate(client, "document.querySelector('.theme-toggle').click()");
    await sleep(300);
    const dark = await evaluate(client, READ);
    check(
      '深色页脚为强调橘色',
      dark.footColor === 'rgb(217, 133, 95)',
      `${dark.footColor} (theme=${dark.theme})`
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
