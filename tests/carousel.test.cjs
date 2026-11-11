// 回归测试: SKILLS 技能轮播的翻页时序、连续点击与循环回绕.
// 运行: node tests/carousel.test.cjs
// 依赖本机 Chrome, 可用 CHROME_PATH 覆盖; 找不到 Chrome 时跳过并以 0 退出.
//
// 判定思路:
// 1. 悬停稳定后单击: 应恰好前进一格, 且全程没有非整组宽度整数倍的瞬移;
// 2. 悬停 200ms 后连续点击两次: 两次点击都不得丢失, 最终应前进两格;
// 3. 悬停 200ms 后从末张越界: 回绕必须发生在动画结束后, 不得中途吸附.
// 动画中途吸附会表现为单帧位移速度远超缓动峰值 (实测旧实现约 665px/帧).

const { spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const CHROME =
  process.env.CHROME_PATH ||
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const ROOT = path.resolve(__dirname, '..');
const PAGE_URL = 'file://' + path.join(ROOT, 'index.html');
const PORT = Number(process.env.CHROME_PORT || 9230);
const VIEWPORT_WIDTH = Number(process.env.TEST_WIDTH || 1440);
const VIEWPORT_HEIGHT = Number(process.env.TEST_HEIGHT || 900);
const MOTION = process.env.TEST_MOTION || 'no-preference';
const SETTLE_MS = 700;
const SAMPLE_MS = 1000;
// 缓动峰值速度约 2px/ms; 吸附会产生 5px/ms 以上的单帧速度.
const VELOCITY_LIMIT = 5;

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

async function mouse(client, type, x, y) {
  await client.send('Input.dispatchMouseEvent', {
    type,
    x,
    y,
    button: type === 'mouseMoved' ? 'none' : 'left',
    clickCount: type === 'mousePressed' || type === 'mouseReleased' ? 1 : 0,
  });
}

async function move(client, x, y) {
  await mouse(client, 'mouseMoved', x, y);
}

async function click(client, x, y) {
  await mouse(client, 'mousePressed', x, y);
  await mouse(client, 'mouseReleased', x, y);
}

// 按住左键水平拖动 (C027): 拖动中的 moves 必须带 buttons:1,
// 否则页面收到的 pointermove 不处于按下状态.
async function dragMouse(client, x0, y0, dx, { steps = 10, gapMs = 60 } = {}) {
  await client.send('Input.dispatchMouseEvent', {
    type: 'mousePressed',
    x: x0,
    y: y0,
    button: 'left',
    buttons: 1,
    clickCount: 1,
  });
  for (let i = 1; i <= steps; i += 1) {
    await client.send('Input.dispatchMouseEvent', {
      type: 'mouseMoved',
      x: Math.round(x0 + (dx * i) / steps),
      y: y0,
      button: 'left',
      buttons: 1,
    });
    await sleep(gapMs);
  }
  await client.send('Input.dispatchMouseEvent', {
    type: 'mouseReleased',
    x: Math.round(x0 + dx),
    y: y0,
    button: 'left',
    buttons: 0,
    clickCount: 1,
  });
}

const PAGE_HELPERS = `
(() => {
  const carousel = document.querySelector('[data-skill-carousel]');
  const track = carousel.querySelector('[data-skill-track]');
  const original = track.querySelector('.skill-set:not([aria-hidden])');
  const originalCards = Array.from(original.querySelectorAll('.skill-card'));
  const names = originalCards.map((card) => card.querySelector('.project-name').textContent);

  function sidePoint(side) {
    const view = carousel.getBoundingClientRect();
    const center = view.left + view.width / 2;
    const cards = Array.from(track.querySelectorAll('.skill-card'));
    const card = cards
      .filter((item) => {
        const box = item.getBoundingClientRect();
        if (side === 'right') {
          return box.left > center && box.left < view.right;
        }
        return box.right < center && box.right > view.left;
      })
      .sort((a, b) =>
        side === 'right'
          ? a.getBoundingClientRect().left - b.getBoundingClientRect().left
          : b.getBoundingClientRect().right - a.getBoundingClientRect().right
      )[0];
    if (!card) {
      return null;
    }
    const box = card.getBoundingClientRect();
    return {
      x: side === 'right' ? box.left + 8 : box.right - 8,
      y: box.top + box.height / 2,
    };
  }

  window.__carouselTest = {
    info() {
      const view = carousel.getBoundingClientRect();
      // 卡片宽度必须取布局宽度: hover 的 scale 会放大 getBoundingClientRect.
      const cardWidth = parseFloat(getComputedStyle(originalCards[0]).width);
      const gap = parseFloat(getComputedStyle(original).columnGap) || 0;
      const step = cardWidth + gap;
      // center 是轨道在索引 0 时把卡片放到正中的 translateX 偏移.
      const center = (view.width - cardWidth) / 2;
      const x = new DOMMatrix(getComputedStyle(track).transform).m41;
      const nearest = Math.round((center - x) / step);
      const logical = ((nearest % originalCards.length) + originalCards.length) %
        originalCards.length;
      const centered = Math.abs(x - (center - nearest * step)) <= 2;
      return {
        index: nearest,
        logical,
        name: originalCards[logical].querySelector('.project-name').textContent,
        names,
        centered,
        transform: x,
        center,
        step,
        group: step * originalCards.length,
      };
    },
    rightPoint() {
      return sidePoint('right');
    },
    leftPoint() {
      return sidePoint('left');
    },
    startSample() {
      const t0 = performance.now();
      const frames = [];
      const read = () =>
        new DOMMatrix(getComputedStyle(track).transform).m41;
      function loop(now) {
        frames.push({ t: Math.round(now - t0), x: read() });
        if (now - t0 < ${SAMPLE_MS}) {
          requestAnimationFrame(loop);
        }
      }
      requestAnimationFrame(loop);
      window.__sample = { t0, frames };
    },
  };
  return true;
})()
`;

function analyze(frames, group) {
  const jumps = [];
  for (let i = 1; i < frames.length; i += 1) {
    const dt = Math.max(frames[i].t - frames[i - 1].t, 1);
    const dx = frames[i].x - frames[i - 1].x;
    const distance = Math.abs(dx);
    if (distance < 1) {
      continue;
    }
    // 回绕后的瞬时重置恰好是整组宽度, 视觉上不可见, 不算跳变.
    const remainder = distance % group;
    if (remainder < 1 || group - remainder < 1) {
      continue;
    }
    const velocity = distance / dt;
    if (velocity > VELOCITY_LIMIT) {
      jumps.push({
        t: frames[i].t,
        dx: Math.round(dx * 10) / 10,
        dt,
        velocity: Math.round(velocity * 100) / 100,
      });
    }
  }
  return jumps;
}

const results = [];
function check(name, passed, detail) {
  results.push({ name, passed, detail });
  console.log(`${passed ? 'PASS' : 'FAIL'}  ${name}${detail ? ' — ' + detail : ''}`);
}

async function resetTo(client, targetIndex) {
  for (let attempt = 0; attempt < 8; attempt += 1) {
    const info = await evaluate(client, 'window.__carouselTest.info()');
    if (info.index === targetIndex) {
      return info;
    }
    const side = info.index < targetIndex ? 'right' : 'left';
    const point = await evaluate(
      client,
      `window.__carouselTest.${side}Point()`
    );
    if (!point) {
      throw new Error('找不到可点击的侧边卡片');
    }
    await move(client, point.x, point.y);
    await sleep(80);
    await click(client, point.x, point.y);
    await sleep(SETTLE_MS);
  }
  throw new Error(`无法复位到 index ${targetIndex}`);
}

async function pressKey(client, key, code, keyCode, text) {
  await client.send('Input.dispatchKeyEvent', {
    type: 'keyDown',
    key,
    code,
    windowsVirtualKeyCode: keyCode,
    ...(text ? { text } : {}),
  });
  await client.send('Input.dispatchKeyEvent', {
    type: 'keyUp',
    key,
    code,
    windowsVirtualKeyCode: keyCode,
  });
}

// 从最后一张项目卡出发, 用 Tab 依次聚焦中间副本的技能卡 (C024).
async function keyboardTour(client, stops) {
  await evaluate(
    client,
    "document.querySelector('.project-grid .project-card:last-of-type').focus()"
  );
  const visited = [];
  for (let i = 0; i < stops; i += 1) {
    await pressKey(client, 'Tab', 'Tab', 9);
    await sleep(SETTLE_MS);
    visited.push(
      await evaluate(
        client,
        `(() => {
          const carousel = document.querySelector('[data-skill-carousel]');
          const active = document.activeElement;
          const view = carousel.getBoundingClientRect();
          const box = active.getBoundingClientRect();
          const name = active.querySelector('.project-name');
          return {
            name: name ? name.textContent : active.tagName,
            offset: Math.round(box.left + box.width / 2 - (view.left + view.width / 2)),
            scrollLeft: carousel.scrollLeft,
            index: window.__carouselTest.info().index,
          };
        })()`
      )
    );
  }
  return visited;
}

async function scenario(client, { label, hoverMs, clicks, gapMs }) {
  // 先把指针移开, 保证 hover 过渡从本次进入卡片开始计时.
  await move(client, 700, 80);
  await sleep(SETTLE_MS);
  const point = await evaluate(client, 'window.__carouselTest.rightPoint()');
  if (!point) {
    throw new Error('找不到右侧半卡');
  }
  await move(client, point.x, point.y);
  await sleep(hoverMs);
  const before = await evaluate(client, 'window.__carouselTest.info()');
  const expectedX = before.center - before.index * before.step;
  check(
    `${label}: 起始位置精确居中`,
    before.centered,
    `transform=${before.transform}, 期望 ${expectedX}`
  );
  await evaluate(client, 'window.__carouselTest.startSample()');
  await click(client, point.x, point.y);
  for (let i = 1; i < clicks; i += 1) {
    await sleep(gapMs);
    await click(client, point.x, point.y);
  }
  await sleep(SAMPLE_MS + 100);
  const after = await evaluate(client, 'window.__carouselTest.info()');
  const frames = await evaluate(client, 'window.__sample.frames');
  return { before, after, frames };
}

(async () => {
  const profile = path.join(ROOT, 'local', 'chrome-test-profile');
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
      width: VIEWPORT_WIDTH,
      height: VIEWPORT_HEIGHT,
      deviceScaleFactor: 1,
      mobile: VIEWPORT_WIDTH < 640,
    });
    await client.send('Emulation.setEmulatedMedia', {
      features: [{ name: 'prefers-reduced-motion', value: MOTION }],
    });
    await client.send('Page.navigate', { url: PAGE_URL });
    await sleep(1000);
    await evaluate(
      client,
      "document.querySelector('[data-skill-carousel]').scrollIntoView({ block: 'center' })"
    );
    await sleep(200);
    await evaluate(client, PAGE_HELPERS);

    const geometry = await evaluate(client, 'window.__carouselTest.info()');
    check(
      '几何基线',
      Math.abs(geometry.step - 273) < 1 && Math.abs(geometry.group - 819) < 1,
      `step=${geometry.step}, group=${geometry.group}`
    );

    // 场景 1: 悬停稳定后单击, 作为基线.
    await resetTo(client, 3);
    const settled = await scenario(client, {
      label: '悬停稳定',
      hoverMs: 900,
      clicks: 1,
      gapMs: 0,
    });
    const settledExpected =
      settled.before.names[
        (settled.before.logical + 1) % settled.before.names.length
      ];
    check(
      '悬停稳定后单击前进一格',
      settled.after.name === settledExpected,
      `期望 ${settledExpected}, 实际 ${settled.after.name}`
    );
    const settledJumps = analyze(settled.frames, settled.before.group);
    check(
      '悬停稳定后单击无中途跳变',
      settledJumps.length === 0,
      settledJumps.length ? JSON.stringify(settledJumps) : '无异常帧'
    );

    // 场景 2: 悬停 200ms 后 50ms 内连点两次, 两次都必须生效.
    await resetTo(client, 3);
    const rapid = await scenario(client, {
      label: '连续点击',
      hoverMs: 200,
      clicks: 2,
      gapMs: 50,
    });
    const rapidExpected =
      rapid.before.names[
        (rapid.before.logical + 2) % rapid.before.names.length
      ];
    check(
      '连续两次点击不丢失',
      rapid.after.name === rapidExpected,
      `期望 ${rapidExpected}, 实际 ${rapid.after.name}`
    );

    // 场景 3: 从末张越界回绕, 吸附不得发生在动画中途.
    await resetTo(client, 5);
    const wrap = await scenario(client, {
      label: '回绕',
      hoverMs: 200,
      clicks: 1,
      gapMs: 0,
    });
    const wrapExpected =
      wrap.before.names[
        (wrap.before.logical + 1) % wrap.before.names.length
      ];
    check(
      '回绕后回到首张',
      wrap.after.name === wrapExpected,
      `期望 ${wrapExpected}, 实际 ${wrap.after.name}`
    );
    const wrapJumps = analyze(wrap.frames, wrap.before.group);
    check(
      '回绕无中途吸附',
      wrapJumps.length === 0,
      wrapJumps.length ? JSON.stringify(wrapJumps) : '无异常帧'
    );

    // 场景 4: 只有居中卡片按链接打开, 侧边卡片必须被拦截为翻页.
    await resetTo(client, 3);
    const nav = await evaluate(
      client,
      `(() => {
        const carousel = document.querySelector('[data-skill-carousel]');
        const track = carousel.querySelector('[data-skill-track]');
        const view = carousel.getBoundingClientRect();
        const center = view.left + view.width / 2;
        const cards = Array.from(track.querySelectorAll('.skill-card'));
        const centered = cards.find((card) => {
          const box = card.getBoundingClientRect();
          return Math.abs(box.left + box.width / 2 - center) <= 2;
        });
        const side = cards.find((card) => {
          const box = card.getBoundingClientRect();
          return box.right < center && box.right > view.left;
        });
        function probe(card) {
          let prevented = null;
          const listener = (event) => {
            prevented = event.defaultPrevented;
            event.preventDefault();
          };
          card.addEventListener('click', listener, { once: true });
          card.dispatchEvent(
            new MouseEvent('click', {
              bubbles: true,
              cancelable: true,
              view: window,
            })
          );
          card.removeEventListener('click', listener);
          return prevented;
        }
        return { centered: probe(centered), side: probe(side) };
      })()`
    );
    check(
      '居中卡片按链接打开',
      nav.centered === false,
      `defaultPrevented=${nav.centered}`
    );
    check(
      '侧边卡片拦截为翻页',
      nav.side === true,
      `defaultPrevented=${nav.side}`
    );

    // 场景 5: 键盘 Tab 进入轨道, 聚焦卡片滑到正中, 视窗不被焦点滚动 (C024).
    // 先等场景 4 的拦截翻页结束, 否则 resetTo 会读到动画中途的位置.
    await sleep(SETTLE_MS);
    await resetTo(client, 3);
    await move(client, 700, 80);
    await sleep(SETTLE_MS);
    const names = (await evaluate(client, 'window.__carouselTest.info()')).names;
    const describe = (stops) =>
      stops
        .map(
          (stop) =>
            `${stop.name}@offset=${stop.offset},scroll=${stop.scrollLeft},index=${stop.index}`
        )
        .join(', ');
    const tour = await keyboardTour(client, names.length);
    check(
      'Tab 聚焦的技能卡依次滑到正中且视窗不滚动 (C024)',
      tour.map((stop) => stop.name).join(',') === names.join(',') &&
        tour.every(
          (stop) => Math.abs(stop.offset) <= 3 && stop.scrollLeft === 0
        ),
      describe(tour)
    );

    // 聚焦卡片已居中: 回车按普通链接打开 (临时 preventDefault 避免打开新标签).
    await evaluate(
      client,
      `(() => {
        window.__enterPrevented = null;
        document.activeElement.addEventListener('click', (event) => {
          window.__enterPrevented = event.defaultPrevented;
          event.preventDefault();
        }, { once: true });
      })()`
    );
    await pressKey(client, 'Enter', 'Enter', 13, '\r');
    await sleep(100);
    const enterPrevented = await evaluate(client, 'window.__enterPrevented');
    check(
      '回车打开居中的聚焦卡片, 不被拦截为翻页 (C024)',
      enterPrevented === false,
      `defaultPrevented=${enterPrevented}`
    );

    // 不支持 overflow: clip 的浏览器: 视窗仍可被焦点滚动, 由脚本立即归零.
    await evaluate(
      client,
      `(() => {
        const style = document.createElement('style');
        style.id = 'force-overflow-hidden';
        style.textContent = '.skill-carousel { overflow: hidden !important; }';
        document.head.appendChild(style);
      })()`
    );
    const fallback = await keyboardTour(client, names.length);
    await evaluate(
      client,
      "document.getElementById('force-overflow-hidden').remove(); document.activeElement.blur()"
    );
    check(
      'overflow: hidden 降级下聚焦仍居中且 scrollLeft 归零 (C024)',
      fallback.every(
        (stop) => Math.abs(stop.offset) <= 3 && stop.scrollLeft === 0
      ),
      describe(fallback)
    );

    // 场景 6 (C027): 慢速拖拽 1.4 步, 吸附到最近的整格且不触发卡片点击.
    // 慢速 (约 0.3px/ms) 低于 0.5px/ms 的轻甩阈值, 只按位移吸附.
    await resetTo(client, 3);
    await move(client, 700, 80);
    await sleep(SETTLE_MS);
    await evaluate(
      client,
      `(() => {
        const carousel = document.querySelector('[data-skill-carousel]');
        const view = carousel.getBoundingClientRect();
        const center = view.left + view.width / 2;
        window.__cardClicks = 0;
        carousel.querySelectorAll('.skill-card').forEach((card) => {
          const box = card.getBoundingClientRect();
          if (Math.abs(box.left + box.width / 2 - center) <= 2) {
            card.addEventListener('click', () => {
              window.__cardClicks += 1;
            });
          }
        });
        return true;
      })()`
    );
    const dragPoint = await evaluate(
      client,
      `(() => {
        const view = document
          .querySelector('[data-skill-carousel]')
          .getBoundingClientRect();
        return {
          x: Math.round(view.left + view.width / 2),
          y: Math.round(view.top + view.height / 2),
        };
      })()`
    );
    const slowBefore = await evaluate(client, 'window.__carouselTest.info()');
    await dragMouse(client, dragPoint.x, dragPoint.y, -slowBefore.step * 1.4, {
      steps: 20,
      gapMs: 70,
    });
    await sleep(SETTLE_MS + 400);
    const slowAfter = await evaluate(client, 'window.__carouselTest.info()');
    const slowClicks = await evaluate(client, 'window.__cardClicks');
    check(
      '慢速拖拽 1.4 步吸附到最近整格并居中 (C027)',
      slowAfter.index === 4 && slowAfter.centered,
      `index=${slowAfter.index}, centered=${slowAfter.centered}`
    );
    check(
      '拖拽后不触发卡片点击 (C027)',
      slowClicks === 0,
      `cardClicks=${slowClicks}`
    );

    // 场景 7 (C027): 快速轻甩 0.4 步 (约 1.5px/ms): 位移四舍五入回原格,
    // 轻甩按速度多走一格; 没有轻甩判定时会弹回 index 3.
    await resetTo(client, 3);
    await move(client, 700, 80);
    await sleep(SETTLE_MS);
    await dragMouse(client, dragPoint.x, dragPoint.y, -slowBefore.step * 0.4, {
      steps: 6,
      gapMs: 12,
    });
    await sleep(SETTLE_MS + 400);
    const flickAfter = await evaluate(client, 'window.__carouselTest.info()');
    check(
      '快速轻甩 0.4 步多走一格 (C027)',
      flickAfter.index === 4 && flickAfter.centered,
      `index=${flickAfter.index}, centered=${flickAfter.centered}`
    );

    // 场景 8 (C027): 向右慢拖 2.6 步越过缓冲边界, 橡皮筋衰减后弹回并吸附首张.
    // 期望吸附到 MIN_INDEX(1), normalize 后回到 index 4, 逻辑位置为首张卡.
    await resetTo(client, 3);
    await move(client, 700, 80);
    await sleep(SETTLE_MS);
    await dragMouse(client, dragPoint.x, dragPoint.y, slowBefore.step * 2.6, {
      steps: 24,
      gapMs: 70,
    });
    await sleep(SETTLE_MS + 400);
    const rubberAfter = await evaluate(client, 'window.__carouselTest.info()');
    check(
      '右拖越界橡皮筋弹回并吸附首张 (C027)',
      rubberAfter.index === 4 &&
        rubberAfter.logical === 1 &&
        rubberAfter.centered,
      `index=${rubberAfter.index}, logical=${rubberAfter.logical}, centered=${rubberAfter.centered}`
    );

    // 场景 9 (C027): 亚阈值位移 (4px) 在右侧半卡上松手仍是点击翻页.
    await resetTo(client, 3);
    await move(client, 700, 80);
    await sleep(SETTLE_MS);
    const subPoint = await evaluate(
      client,
      'window.__carouselTest.rightPoint()'
    );
    if (!subPoint) {
      throw new Error('找不到右侧半卡');
    }
    await dragMouse(client, subPoint.x, subPoint.y, 4, {
      steps: 2,
      gapMs: 30,
    });
    await sleep(SETTLE_MS);
    const subAfter = await evaluate(client, 'window.__carouselTest.info()');
    check(
      '亚阈值 4px 位移仍是点击翻页 (C027)',
      subAfter.index === 4 && subAfter.centered,
      `index=${subAfter.index}, centered=${subAfter.centered}`
    );

    client.close();
  } finally {
    chrome.kill('SIGTERM');
  }

  const failed = results.filter((item) => !item.passed);
  console.log(
    `\n${results.length - failed.length}/${results.length} 项通过`
  );
  if (failed.length) {
    process.exitCode = 1;
  }
})().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
