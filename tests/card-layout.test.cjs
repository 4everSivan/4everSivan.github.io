// 回归测试: 卡片图标 + 标题同行、描述横跨下方.
// 运行: node tests/card-layout.test.cjs
// 依赖本机 Chrome, 可用 CHROME_PATH 覆盖; 找不到 Chrome 时跳过并以 0 退出.

const { spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const CHROME =
  process.env.CHROME_PATH ||
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const ROOT = path.resolve(__dirname, '..');
const PAGE_URL = 'file://' + path.join(ROOT, 'index.html');
const PORT = Number(process.env.CHROME_PORT || 9252);

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

function readCard(selector) {
  return `(() => {
    const card = document.querySelector(${JSON.stringify(selector)});
    if (!card) {
      return null;
    }
    const style = getComputedStyle(card);
    const icon = card.querySelector('.project-icon');
    const name = card.querySelector('.project-name');
    const desc = card.querySelector('.project-desc');
    const body = card.querySelector('.project-body');
    const cardRect = card.getBoundingClientRect();
    const iconRect = icon.getBoundingClientRect();
    const nameRect = name.getBoundingClientRect();
    const descRect = desc.getBoundingClientRect();
    return {
      display: style.display,
      areas: style.gridTemplateAreas,
      bodyDisplay: getComputedStyle(body).display,
      iconCenterY: iconRect.top + iconRect.height / 2,
      nameCenterY: nameRect.top + nameRect.height / 2,
      nameLeft: nameRect.left,
      iconRight: iconRect.right,
      descTop: descRect.top,
      iconBottom: iconRect.bottom,
      descWidth: Math.round(descRect.width * 10) / 10,
      contentWidth:
        Math.round(
          (cardRect.width -
            parseFloat(style.paddingLeft) -
            parseFloat(style.paddingRight) -
            parseFloat(style.borderLeftWidth) -
            parseFloat(style.borderRightWidth)) * 10
        ) / 10,
      noHorizontalScroll:
        document.documentElement.scrollWidth <= window.innerWidth + 1,
    };
  })()`;
}

function assertCard(label, info) {
  check(
    `${label}: 卡片为两行网格`,
    info &&
      info.display === 'grid' &&
      info.areas.includes('icon name') &&
      info.areas.includes('desc desc'),
    info ? `display=${info.display}, areas=${info.areas}` : '未找到卡片'
  );
  check(
    `${label}: 图标与标题同行`,
    info &&
      Math.abs(info.iconCenterY - info.nameCenterY) <= 2 &&
      info.nameLeft >= info.iconRight,
    info
      ? `iconCenterY=${info.iconCenterY.toFixed(1)}, nameCenterY=${info.nameCenterY.toFixed(1)}, nameLeft=${info.nameLeft.toFixed(1)}, iconRight=${info.iconRight.toFixed(1)}`
      : ''
  );
  check(
    `${label}: 描述在下方并横跨卡片`,
    info &&
      info.descTop >= info.iconBottom &&
      Math.abs(info.descWidth - info.contentWidth) <= 1,
    info
      ? `descTop=${info.descTop.toFixed(1)}, iconBottom=${info.iconBottom.toFixed(1)}, descWidth=${info.descWidth}/${info.contentWidth}`
      : ''
  );
  check(
    `${label}: 无横向滚动`,
    info && info.noHorizontalScroll,
    ''
  );
}

(async () => {
  const profile = path.join(ROOT, 'local', 'chrome-card-layout-profile');
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
    await sleep(900);

    const project = await evaluate(client, readCard('.project-grid .project-card'));
    assertCard('项目卡 1440', project);
    check(
      '项目卡: 文本容器为 contents',
      project && project.bodyDisplay === 'contents',
      project ? project.bodyDisplay : ''
    );

    const skill = await evaluate(
      client,
      readCard('.skill-set:not([aria-hidden]) .skill-card')
    );
    assertCard('技能卡 1440', skill);

    await client.send('Emulation.setDeviceMetricsOverride', {
      width: 390,
      height: 844,
      deviceScaleFactor: 1,
      mobile: true,
    });
    await client.send('Page.reload');
    await sleep(900);

    const mobileProject = await evaluate(
      client,
      readCard('.project-grid .project-card')
    );
    assertCard('项目卡 390', mobileProject);

    const mobileSkill = await evaluate(
      client,
      readCard('.skill-set:not([aria-hidden]) .skill-card')
    );
    assertCard('技能卡 390', mobileSkill);

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
