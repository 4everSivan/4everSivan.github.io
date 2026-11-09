// 首页渐进增强: 更新版权年份、切换主题、揭示区块与轮播交互.
// 默认样式即终态; 禁用 JS 或脚本加载失败时, 内容保持完整可见.

(function () {
  'use strict';

  window.sivanRevealReady = true;

  const yearElement = document.querySelector('[data-current-year]');
  if (yearElement) {
    yearElement.textContent = String(new Date().getFullYear());
  }

  // 主题切换: 首次按浏览器偏好, 手动选择写入 localStorage; 仅浅/深两态.
  const themeToggle = document.querySelector('.theme-toggle');
  if (themeToggle) {
    const root = document.documentElement;
    const media = window.matchMedia('(prefers-color-scheme: dark)');
    const themeMeta = document.querySelector('meta[name="theme-color"]');

    function applyTheme(theme, persist) {
      root.setAttribute('data-theme', theme);
      if (persist) {
        try {
          localStorage.setItem('theme', theme);
        } catch (error) {
          // 隐私模式无法持久化, 本次会话仍然生效.
        }
      }
      const label =
        theme === 'dark' ? '切换到浅色模式' : '切换到深色模式';
      themeToggle.setAttribute('aria-label', label);
      themeToggle.setAttribute('title', label);
      if (themeMeta) {
        themeMeta.setAttribute(
          'content',
          theme === 'dark' ? '#17130f' : '#f7e5d1'
        );
      }
    }

    function currentTheme() {
      return root.getAttribute('data-theme') === 'dark' ? 'dark' : 'light';
    }

    themeToggle.addEventListener('click', () => {
      const next = currentTheme() === 'dark' ? 'light' : 'dark';
      if (typeof document.startViewTransition !== 'function') {
        applyTheme(next, true);
        return;
      }
      // 揭示方向与速度由样式表里的纵向渐变蒙版决定, 这里只负责换主题.
      try {
        document.startViewTransition(() => applyTheme(next, true));
      } catch (error) {
        applyTheme(next, true);
      }
    });

    applyTheme(currentTheme(), false);

    media.addEventListener('change', (event) => {
      let stored = null;
      try {
        stored = localStorage.getItem('theme');
      } catch (error) {
        stored = null;
      }
      if (stored !== 'light' && stored !== 'dark') {
        applyTheme(event.matches ? 'dark' : 'light', false);
      }
    });
  }

  // 顶栏搜索: 默认放大镜, 点击展开; 快捷键按平台显示.
  const search = document.querySelector('[data-search]');
  if (search) {
    const searchToggle = search.querySelector('.search-toggle');
    const searchContent = search.querySelector('.search-content');
    const searchInput = search.querySelector('.search-input');

    function setSearch(open, focus) {
      search.classList.toggle('is-open', open);
      searchToggle.setAttribute('aria-expanded', String(open));
      searchContent.setAttribute('aria-hidden', String(!open));
      searchInput.tabIndex = open ? 0 : -1;
      if (open && focus) {
        searchInput.focus();
      }
      if (!open && document.activeElement === searchInput) {
        searchInput.blur();
      }
    }

    searchToggle.addEventListener('click', () => {
      setSearch(!search.classList.contains('is-open'), true);
    });

    search.addEventListener('keydown', (event) => {
      if (event.key === 'Escape') {
        setSearch(false, false);
      }
    });

    document.addEventListener('keydown', (event) => {
      if (
        (event.metaKey || event.ctrlKey) &&
        event.key.toLowerCase() === 'f'
      ) {
        event.preventDefault();
        const open = search.classList.contains('is-open');
        setSearch(!open, !open);
      }
    });

    document.addEventListener('pointerdown', (event) => {
      if (
        search.classList.contains('is-open') &&
        !search.contains(event.target)
      ) {
        setSearch(false, false);
      }
    });
  }

  // 引文默认折叠: 页面从 PROJECTS 开始, 点击字标后展开.
  const versePanel = document.querySelector('[data-verse-panel]');
  if (versePanel) {
    versePanel.setAttribute('aria-hidden', 'true');
  }

  // 字标回顶: 自定义缓动, 不受 reduced-motion 的瞬时跳转影响.
  // 仅首页锚点 (href="#top") 走回顶; 文章页字标链接回首页, 保持默认导航.
  const wordmark = document.querySelector('.wordmark a');
  if (wordmark) {
    wordmark.addEventListener('click', (event) => {
      if (wordmark.getAttribute('href') !== '#top') {
        return;
      }
      if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
        return;
      }
      event.preventDefault();
      if (versePanel) {
        const isOpen = versePanel.classList.toggle('is-open');
        if (isOpen) {
          versePanel.removeAttribute('aria-hidden');
        } else {
          versePanel.setAttribute('aria-hidden', 'true');
        }
      }
      const start = window.scrollY;
      if (start <= 0) {
        return;
      }
      const duration = 460;
      const startTime = performance.now();
      function step(now) {
        const progress = Math.min((now - startTime) / duration, 1);
        const eased = 1 - Math.pow(1 - progress, 3);
        const top = Math.round(start * (1 - eased));
        try {
          window.scrollTo({ top, behavior: 'instant' });
        } catch (error) {
          window.scrollTo(0, top);
        }
        if (progress < 1) {
          window.requestAnimationFrame(step);
        }
      }
      window.requestAnimationFrame(step);
      if (window.history && window.history.replaceState) {
        window.history.replaceState(null, '', '#top');
      }
    });
  }

  // 项目卡片: 指针位置驱动柔光, 离开后回到默认位置.
  document.querySelectorAll('.project-card').forEach((card) => {
    card.addEventListener(
      'pointermove',
      (event) => {
        const rect = card.getBoundingClientRect();
        card.style.setProperty('--x', `${event.clientX - rect.left}px`);
        card.style.setProperty('--y', `${event.clientY - rect.top}px`);
      },
      { passive: true }
    );
    card.addEventListener('pointerleave', () => {
      card.style.removeProperty('--x');
      card.style.removeProperty('--y');
    });
  });
})();

// 技能轮播: 中间一张完整卡, 两侧露出半张并淡出; 点击左右半卡切换.
// 动画由 Web Animations API 驱动: 卡片 hover 过渡的 transitionend 会冒泡到
// track, 若把它当作翻页结束信号, 翻页动画会在中途被提前截断.
(function () {
  'use strict';

  const carousel = document.querySelector('[data-skill-carousel]');
  if (!carousel) {
    return;
  }

  const track = carousel.querySelector('[data-skill-track]');
  const sets = track
    ? Array.from(track.querySelectorAll('.skill-set'))
    : [];
  const original = sets.find((set) => !set.hasAttribute('aria-hidden'));
  if (!track || !original) {
    return;
  }

  const cards = Array.from(original.querySelectorAll('.skill-card'));
  if (cards.length === 0) {
    return;
  }

  const allCards = Array.from(track.querySelectorAll('.skill-card'));
  const count = cards.length;
  const STEP_MS = 600;
  const MAX_MS = 900;
  const EASING = 'cubic-bezier(0.32, 0.72, 0, 1)';
  const CENTER_TOLERANCE = 3;
  // 轨道只有三份副本; 索引越界会露出空白, 两端各留一张作为缓冲.
  const MIN_INDEX = 1;
  const MAX_INDEX = count * 3 - 2;

  let index = count;
  let animation = null;

  function metrics() {
    const gap = parseFloat(getComputedStyle(original).columnGap) || 0;
    // 必须取布局宽度: hover 的 scale(1.01) 会放大 getBoundingClientRect,
    // 用它计算目标偏移会让轨道停不到正中心.
    const cardWidth = parseFloat(getComputedStyle(cards[0]).width) || 0;
    const viewWidth = carousel.clientWidth;
    return {
      center: (viewWidth - cardWidth) / 2,
      step: cardWidth + gap,
    };
  }

  function offsetOf(target) {
    const { center, step } = metrics();
    return center - target * step;
  }

  function currentOffset() {
    const matrix = new DOMMatrixReadOnly(getComputedStyle(track).transform);
    return Number.isFinite(matrix.m41) ? matrix.m41 : 0;
  }

  function setOffset(offset) {
    if (animation) {
      animation.cancel();
      animation = null;
    }
    track.style.transform = `translateX(${offset}px)`;
  }

  function normalize() {
    let shifted = false;
    if (index >= count * 2) {
      index -= count;
      shifted = true;
    } else if (index < count) {
      index += count;
      shifted = true;
    }
    if (shifted) {
      // 三份副本内容一致, 瞬时重置在视觉上不可见.
      setOffset(offsetOf(index));
    }
  }

  function go(delta) {
    const next = index + delta;
    if (next < MIN_INDEX || next > MAX_INDEX) {
      return;
    }
    index = next;

    const { center, step } = metrics();
    const from = currentOffset();
    const to = center - index * step;

    if (animation) {
      // 动画进行中再次点击: 从当前视觉位置连续过渡到新目标.
      animation.cancel();
      animation = null;
      track.style.transform = `translateX(${from}px)`;
    }

    if (Math.abs(to - from) < 0.5) {
      track.style.transform = `translateX(${to}px)`;
      normalize();
      return;
    }

    // 一次跨多格时适度延长时长, 但保持单步 600ms 的基准.
    const steps = Math.max(1, Math.round(Math.abs(to - from) / step));
    const duration = Math.min(MAX_MS, STEP_MS + (steps - 1) * 150);
    if (typeof track.animate !== 'function') {
      track.style.transform = `translateX(${to}px)`;
      normalize();
      return;
    }
    const anim = track.animate(
      [
        { transform: `translateX(${from}px)` },
        { transform: `translateX(${to}px)` },
      ],
      { duration, easing: EASING, fill: 'forwards' }
    );
    animation = anim;
    anim.onfinish = () => {
      if (animation !== anim) {
        return;
      }
      animation = null;
      // 先写入终值再取消动画, 避免取消时回到起点.
      track.style.transform = `translateX(${to}px)`;
      anim.cancel();
      normalize();
    };
  }

  // 只有真正居中的卡片按普通链接打开; 其余点击一律翻页.
  function centered(card) {
    const box = card.getBoundingClientRect();
    const view = carousel.getBoundingClientRect();
    return (
      Math.abs(box.left + box.width / 2 - (view.left + view.width / 2)) <=
      CENTER_TOLERANCE
    );
  }

  allCards.forEach((card) => {
    card.addEventListener('click', (event) => {
      if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
        return;
      }
      if (card.closest('.skill-set') === original && centered(card)) {
        return;
      }
      event.preventDefault();
      const box = card.getBoundingClientRect();
      const view = carousel.getBoundingClientRect();
      go(box.left + box.width / 2 < view.left + view.width / 2 ? -1 : 1);
    });
  });

  setOffset(offsetOf(index));

  let resizeFrame = 0;
  window.addEventListener('resize', () => {
    window.cancelAnimationFrame(resizeFrame);
    resizeFrame = window.requestAnimationFrame(() => {
      setOffset(offsetOf(index));
      normalize();
    });
  });
})();

// 博客切页: 内容层先淡出, 再进入文章页; 文章页加载时自下方淡入归位.
// 过渡链接由构建期写入 data-page-transition; 禁用 JS 时保持普通整页导航.
(function () {
  'use strict';

  const links = Array.from(
    document.querySelectorAll('a[data-page-transition]')
  );
  if (links.length === 0) {
    return;
  }

  // 与样式表里的 200ms 离场过渡对齐, 再留 20ms 收尾缓冲.
  const LEAVE_MS = 220;
  let leaving = false;

  function prefetch(link) {
    if (link.dataset.prefetched === 'true') {
      return;
    }
    link.dataset.prefetched = 'true';
    const hint = document.createElement('link');
    hint.rel = 'prefetch';
    hint.href = link.href;
    document.head.appendChild(hint);
  }

  links.forEach((link) => {
    link.addEventListener('pointerenter', () => prefetch(link), {
      passive: true,
    });
    link.addEventListener('focus', () => prefetch(link));
  });

  function leave(url) {
    if (leaving) {
      return;
    }
    leaving = true;
    document.documentElement.classList.add('is-page-leaving');
    window.setTimeout(() => {
      window.location.assign(url);
    }, LEAVE_MS);
  }

  links.forEach((link) => {
    link.addEventListener('click', (event) => {
      if (event.defaultPrevented) {
        return;
      }
      if (
        event.button !== 0 ||
        event.metaKey ||
        event.ctrlKey ||
        event.shiftKey ||
        event.altKey
      ) {
        return;
      }
      if (link.target && link.target !== '_self') {
        return;
      }
      if (link.hasAttribute('download')) {
        return;
      }
      const url = new URL(link.href, window.location.href);
      if (url.origin !== window.location.origin) {
        return;
      }
      event.preventDefault();
      leave(url.href);
    });
  });

  // 返回时页面可能从 bfcache 恢复: 清掉离场态, 避免内容保持透明.
  window.addEventListener('pageshow', () => {
    leaving = false;
    document.documentElement.classList.remove('is-page-leaving');
  });
})();
