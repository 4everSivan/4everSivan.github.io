// 回归测试: 浅色字标配色与主题扩散切换动画.
// 运行: node tests/theme.test.cjs
// 依赖本机 Chrome, 可用 CHROME_PATH 覆盖; 找不到 Chrome 时跳过并以 0 退出.

const { spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const CHROME =
  process.env.CHROME_PATH ||
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const ROOT = path.resolve(__dirname, '..');
const PAGE_URL = 'file://' + path.join(ROOT, 'index.html');
const PORT = Number(process.env.CHROME_PORT || 9242);

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

const READ_COLORS = `
(() => {
  const root = document.documentElement;
  const style = (selector) => getComputedStyle(document.querySelector(selector)).color;
  const sun = document.querySelector('.theme-toggle .icon-sun');
  const moon = document.querySelector('.theme-toggle .icon-moon');
  const sunStyle = getComputedStyle(sun);
  const moonStyle = getComputedStyle(moon);
  const sheetRules = [];
  for (const sheet of Array.from(document.styleSheets)) {
    try {
      for (const rule of Array.from(sheet.cssRules)) {
        sheetRules.push(rule);
      }
    } catch (error) {
      // 跨域样式表不可读, 忽略.
    }
  }
  return {
    theme: root.getAttribute('data-theme'),
    initial: style('.wordmark-initial'),
    rest: style('.wordmark a'),
    toggle: style('.theme-toggle'),
    sunOpacity: sunStyle.opacity,
    moonOpacity: moonStyle.opacity,
    hasRevealKeyframes: sheetRules.some(
      (rule) => rule.type === CSSRule.KEYFRAMES_RULE && rule.name === 'theme-reveal'
    ),
    hasIconInKeyframes: sheetRules.some(
      (rule) => rule.type === CSSRule.KEYFRAMES_RULE && rule.name === 'icon-in'
    ),
    hasIconOutKeyframes: sheetRules.some(
      (rule) => rule.type === CSSRule.KEYFRAMES_RULE && rule.name === 'icon-out'
    ),
    hasViewTransitionRule: sheetRules.some(
      (rule) =>
        rule.selectorText &&
        rule.selectorText.includes('::view-transition-new(root)')
    ),
    hasIconViewTransitionRule: sheetRules.some(
      (rule) =>
        rule.selectorText &&
        rule.selectorText.includes('::view-transition-new(theme-toggle)')
    ),
  };
})()
`;

const PROBE_TRANSITION = `
(() => {
  const button = document.querySelector('.theme-toggle');
  const rect = button.getBoundingClientRect();
  window.__vt = null;
  document.startViewTransition = (callback) => {
    const root = document.documentElement;
    const style = getComputedStyle(root);
    const record = {
      before: root.getAttribute('data-theme'),
      x: parseFloat(style.getPropertyValue('--theme-x')),
      y: parseFloat(style.getPropertyValue('--theme-y')),
      radius: parseFloat(style.getPropertyValue('--theme-r')),
      buttonX: rect.left + rect.width / 2,
      buttonY: rect.top + rect.height / 2,
      innerWidth: window.innerWidth,
      innerHeight: window.innerHeight,
    };
    callback();
    record.after = root.getAttribute('data-theme');
    window.__vt = record;
    return {
      finished: Promise.resolve(),
      ready: Promise.resolve(),
      updateCallbackDone: Promise.resolve(),
    };
  };
  button.click();
  return window.__vt;
})()
`;

(async () => {
  const profile = path.join(ROOT, 'local', 'chrome-theme-profile');
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

    const light = await evaluate(client, READ_COLORS);
    check('浅色主题已生效', light.theme === 'light', `data-theme=${light.theme}`);
    check(
      '浅色 S 为墨色',
      light.initial === 'rgb(33, 26, 21)',
      light.initial
    );
    check(
      '浅色 ivan. 与主题按钮同灰',
      light.rest === light.toggle && light.rest === 'rgb(111, 96, 84)',
      `ivan.=${light.rest}, toggle=${light.toggle}`
    );
    check('存在扩散关键帧', light.hasRevealKeyframes, '');
    check('存在 view-transition 规则', light.hasViewTransitionRule, '');
    check(
      '浅色显示月亮图标',
      light.sunOpacity === '0' && light.moonOpacity === '1',
      `sun=${light.sunOpacity}, moon=${light.moonOpacity}`
    );
    check(
      '存在图标切换关键帧',
      light.hasIconInKeyframes && light.hasIconOutKeyframes,
      ''
    );
    check('存在图标 view-transition 规则', light.hasIconViewTransitionRule, '');

    const vt = await evaluate(client, PROBE_TRANSITION);
    check(
      '点击调用 View Transition',
      vt && vt.before === 'light' && vt.after === 'dark',
      vt ? `${vt.before} -> ${vt.after}` : '未调用'
    );
    check(
      '扩散圆心在按钮中心',
      vt &&
        Math.abs(vt.x - vt.buttonX) <= 0.5 &&
        Math.abs(vt.y - vt.buttonY) <= 0.5,
      vt ? `圆心=(${vt.x}, ${vt.y}), 按钮=(${vt.buttonX}, ${vt.buttonY})` : ''
    );
    const needed = vt
      ? Math.hypot(
          Math.max(vt.x, vt.innerWidth - vt.x),
          Math.max(vt.y, vt.innerHeight - vt.y)
        )
      : 0;
    check(
      '扩散半径覆盖视口最远角',
      vt && vt.radius >= needed - 1,
      vt ? `r=${vt.radius}, 需要=${Math.round(needed)}` : ''
    );

    // 等图标 420ms 过渡结束再读取最终状态.
    await sleep(500);
    const dark = await evaluate(client, READ_COLORS);
    check('深色主题已生效', dark.theme === 'dark', `data-theme=${dark.theme}`);
    check(
      '深色 S 为强调橘色',
      dark.initial === 'rgb(217, 133, 95)',
      dark.initial
    );
    check(
      '深色 ivan. 为墨色',
      dark.rest === 'rgb(236, 228, 216)',
      dark.rest
    );
    check(
      '深色显示太阳图标',
      dark.sunOpacity === '1' && dark.moonOpacity === '0',
      `sun=${dark.sunOpacity}, moon=${dark.moonOpacity}`
    );

    // 不支持 View Transitions 时必须仍能切换.
    const fallback = await evaluate(
      client,
      `(() => {
        document.startViewTransition = undefined;
        document.querySelector('.theme-toggle').click();
        return document.documentElement.getAttribute('data-theme');
      })()`
    );
    check('无 View Transition 时降级切换', fallback === 'light', `data-theme=${fallback}`);

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
