// 回归测试: 首页字符画底图的存在性、居中、主题配色与交互隔离.
// 运行: node tests/page-art.test.cjs
// 依赖本机 Chrome, 可用 CHROME_PATH 覆盖; 找不到 Chrome 时跳过并以 0 退出.

const { spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const CHROME =
  process.env.CHROME_PATH ||
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const ROOT = path.resolve(__dirname, '..');
const PAGE_URL = 'file://' + path.join(ROOT, 'index.html');
const PORT = Number(process.env.CHROME_PORT || 9241);

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

function parseColor(value) {
  const match = String(value).match(/rgba?\(([^)]+)\)/);
  if (!match) {
    return null;
  }
  const [r, g, b, a = 1] = match[1].split(',').map(Number);
  const max = Math.max(r, g, b) / 255;
  const min = Math.min(r, g, b) / 255;
  const l = (max + min) / 2;
  const d = max - min;
  if (d === 0) {
    return { h: 0, s: 0, l, a };
  }
  const s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
  let h;
  if (max === r / 255) {
    h = ((g - b) / 255 / d + (g < b ? 6 : 0)) / 6;
  } else if (max === g / 255) {
    h = ((b - r) / 255 / d + 2) / 6;
  } else {
    h = ((r - g) / 255 / d + 4) / 6;
  }
  return { h: h * 360, s, l, a };
}

const results = [];
function check(name, passed, detail) {
  results.push({ name, passed, detail });
  console.log(`${passed ? 'PASS' : 'FAIL'}  ${name}${detail ? ' — ' + detail : ''}`);
}

const READ = `
(() => {
  const art = document.querySelector('.page-art');
  const left = art && art.querySelector('.page-art-left');
  const right = art && art.querySelector('.page-art-right');
  const page = document.querySelector('.page');
  if (!art || !left || !right) {
    return { exists: false };
  }
  const artStyle = getComputedStyle(art);
  const artRect = art.getBoundingClientRect();
  const pageStyle = getComputedStyle(page);
  const atCenter = document.elementFromPoint(
    window.innerWidth / 2,
    window.innerHeight / 2
  );
  function info(pre) {
    const style = getComputedStyle(pre);
    const rect = pre.getBoundingClientRect();
    const text = pre.textContent;
    return {
      color: style.color,
      fontSize: style.fontSize,
      lineHeight: style.lineHeight,
      width: Math.round(rect.width * 10) / 10,
      height: Math.round(rect.height * 10) / 10,
      centerOffsetX:
        Math.round((rect.x + rect.width / 2 - window.innerWidth / 2) * 10) / 10,
      centerOffsetY:
        Math.round((rect.y + rect.height / 2 - window.innerHeight / 2) * 10) / 10,
      lines: text.split('\\n').length,
      filled: text.replace(/\\s/g, '').length,
    };
  }
  return {
    exists: true,
    ariaHidden: art.getAttribute('aria-hidden'),
    position: artStyle.position,
    pointerEvents: artStyle.pointerEvents,
    zIndex: artStyle.zIndex,
    pageZIndex: pageStyle.zIndex,
    coversViewport:
      Math.abs(artRect.width - window.innerWidth) < 1 &&
      Math.abs(artRect.height - window.innerHeight) < 1,
    left: info(left),
    right: info(right),
    insideArt: !!(atCenter && atCenter.closest('.page-art')),
    noHorizontalScroll:
      document.documentElement.scrollWidth <= window.innerWidth + 1,
    innerWidth: window.innerWidth,
  };
})()
`;

async function read(client) {
  return evaluate(client, READ);
}

// C026: 固定装饰层的纵向几何不得绑定可变视口. 移动端工具栏伸缩时 fixed + inset:0
// 的层高逐帧变化, 居中的字符画随之上下抖动. headless 下 lvh 恒等于 innerHeight,
// 无法仿真动态工具栏, 因此核对当前生效的样式规则: 声明 height 为 lvh 且不声明 bottom.
const READ_GEOMETRY = `
(() => {
  const selectors = ['.page-art', '.article-blur'];
  const rules = Object.fromEntries(selectors.map((s) => [s, []]));
  function walk(list) {
    for (const rule of list) {
      if (rule.type === CSSRule.MEDIA_RULE) {
        if (window.matchMedia(rule.conditionText).matches) {
          walk(rule.cssRules);
        }
        continue;
      }
      if (rule.type !== CSSRule.STYLE_RULE) {
        continue;
      }
      for (const part of rule.selectorText.split(',')) {
        const selector = part.trim();
        if (rules[selector]) {
          rules[selector].push({
            height: rule.style.getPropertyValue('height'),
            bottom: rule.style.getPropertyValue('bottom'),
            top: rule.style.getPropertyValue('top'),
          });
        }
      }
    }
  }
  for (const sheet of document.styleSheets) {
    walk(sheet.cssRules);
  }
  const probe = document.createElement('div');
  probe.style.cssText = 'position:fixed;top:0;width:1px;height:100lvh';
  document.body.append(probe);
  const lvh = probe.getBoundingClientRect().height;
  probe.remove();
  function measure(selector) {
    let node = document.querySelector(selector);
    const injected = !node;
    if (injected) {
      node = document.createElement('div');
      node.className = selector.slice(1);
      node.setAttribute('aria-hidden', 'true');
      document.body.append(node);
    }
    const rect = node.getBoundingClientRect();
    if (injected) {
      node.remove();
    }
    return { top: rect.top, height: rect.height };
  }
  const result = { lvh };
  for (const selector of selectors) {
    const declared = rules[selector];
    const heights = declared.map((r) => r.height).filter(Boolean);
    result[selector] = {
      ruleCount: declared.length,
      height: heights[heights.length - 1] || '',
      bottoms: declared.map((r) => r.bottom).filter(Boolean),
      tops: declared.map((r) => r.top).filter(Boolean),
      rect: measure(selector),
    };
  }
  return result;
})()
`;

function checkGeometry(label, geometry) {
  for (const selector of ['.page-art', '.article-blur']) {
    const info = geometry[selector];
    check(
      `${label} ${selector} 纵向几何不绑定可变视口`,
      info.ruleCount > 0 &&
        /lvh$/.test(info.height) &&
        info.bottoms.length === 0 &&
        info.tops.includes('0px') &&
        Math.abs(info.rect.top) < 1 &&
        Math.abs(info.rect.height - geometry.lvh) < 1,
      `height=${info.height || '(未声明)'}, bottom=${info.bottoms.join('/') || '(未声明)'}, ` +
        `rect=${Math.round(info.rect.height)}px, lvh=${Math.round(geometry.lvh)}px`
    );
  }
}

(async () => {
  const profile = path.join(ROOT, 'local', 'chrome-page-art-profile');
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

    const light = await read(client);
    check('底图存在且覆盖视口', light.exists && light.coversViewport, '');
    check('装饰层不可交互', light.pointerEvents === 'none', `pointer-events=${light.pointerEvents}`);
    check('装饰层对辅助技术隐藏', light.ariaHidden === 'true', `aria-hidden=${light.ariaHidden}`);
    check('固定定位', light.position === 'fixed', `position=${light.position}`);
    check(
      '位于内容之后',
      Number(light.zIndex) < Number(light.pageZIndex),
      `art z-index=${light.zIndex}, page z-index=${light.pageZIndex}`
    );
    check(
      '右侧字符画右移半幅、垂直居中',
      Math.abs(light.right.centerOffsetX - light.innerWidth / 2) <= 2 &&
        Math.abs(light.right.centerOffsetY) <= 2,
      `offset=(${light.right.centerOffsetX}, ${light.right.centerOffsetY})`
    );
    check(
      '左侧字符画半露并右移一个字符',
      light.left.centerOffsetX + light.innerWidth / 2 > 2 &&
        light.left.centerOffsetX + light.innerWidth / 2 < 20 &&
        Math.abs(light.left.centerOffsetY) <= 2,
      `offset=(${light.left.centerOffsetX}, ${light.left.centerOffsetY})`
    );
    check(
      '左右字符画高度与字号一致',
      Math.abs(light.left.height - light.right.height) <= 1 &&
        light.left.fontSize === light.right.fontSize &&
        light.left.lineHeight === light.right.lineHeight,
      `left=${light.left.width}x${light.left.height}, right=${light.right.width}x${light.right.height}`
    );
    check(
      '字符画内容完整',
      light.left.lines === 44 &&
        light.right.lines === 44 &&
        light.left.filled > 100 &&
        light.right.filled > 100,
      `left=${light.left.lines} 行/${light.left.filled} 字符, right=${light.right.lines} 行/${light.right.filled} 字符`
    );
    check('无横向滚动', light.noHorizontalScroll, `width=${light.innerWidth}`);
    check('中心点未被底图拦截', light.insideArt === false, '');
    checkGeometry('1440px', await evaluate(client, READ_GEOMETRY));

    const lightColor = parseColor(light.right.color);
    check(
      '浅色模式为棕色',
      lightColor &&
        lightColor.h >= 15 &&
        lightColor.h <= 45 &&
        lightColor.l < 0.5 &&
        lightColor.s > 0.25 &&
        lightColor.a > 0.2 &&
        lightColor.a < 0.45 &&
        light.left.color === light.right.color,
      `${light.right.color}`
    );

    // 切换深色主题, 验证颜色变为橙色.
    await evaluate(client, "document.querySelector('.theme-toggle').click()");
    await sleep(200);
    const dark = await read(client);
    const darkColor = parseColor(dark.right.color);
    check(
      '深色模式为字标同款橘色',
      darkColor &&
        darkColor.h >= 10 &&
        darkColor.h <= 45 &&
        darkColor.l > 0.5 &&
        darkColor.s > 0.4 &&
        darkColor.a >= 0.9 &&
        dark.left.color === dark.right.color,
      `${dark.right.color}`
    );
    check(
      '切换主题后位置不变',
      Math.abs(dark.right.centerOffsetX - dark.innerWidth / 2) <= 2 &&
        dark.left.centerOffsetX + dark.innerWidth / 2 > 2 &&
        dark.left.centerOffsetX + dark.innerWidth / 2 < 20 &&
        Math.abs(dark.right.centerOffsetY) <= 2 &&
        Math.abs(dark.left.centerOffsetY) <= 2,
      `left=(${dark.left.centerOffsetX}, ${dark.left.centerOffsetY}), right=(${dark.right.centerOffsetX}, ${dark.right.centerOffsetY})`
    );

    // 打印时隐藏底图.
    await client.send('Emulation.setEmulatedMedia', { media: 'print' });
    await sleep(100);
    const printDisplay = await evaluate(
      client,
      "getComputedStyle(document.querySelector('.page-art')).display"
    );
    check('打印隐藏底图', printDisplay === 'none', `display=${printDisplay}`);
    await client.send('Emulation.setEmulatedMedia', { media: '' });

    // 窄屏: 底图不得撑出横向滚动, 仍保持居中.
    await client.send('Emulation.setDeviceMetricsOverride', {
      width: 390,
      height: 844,
      deviceScaleFactor: 1,
      mobile: true,
    });
    await client.send('Page.reload');
    await sleep(1000);
    const mobile = await read(client);
    check(
      '窄屏左右半露且无横向滚动',
      mobile.noHorizontalScroll &&
        Math.abs(mobile.right.centerOffsetX - mobile.innerWidth / 2) <= 2 &&
        mobile.left.centerOffsetX + mobile.innerWidth / 2 > 2 &&
        mobile.left.centerOffsetX + mobile.innerWidth / 2 < 20 &&
        Math.abs(mobile.right.centerOffsetY) <= 2 &&
        Math.abs(mobile.left.centerOffsetY) <= 2,
      `left=(${mobile.left.centerOffsetX}, ${mobile.left.centerOffsetY}), right=(${mobile.right.centerOffsetX}, ${mobile.right.centerOffsetY})`
    );
    checkGeometry('390px', await evaluate(client, READ_GEOMETRY));

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
