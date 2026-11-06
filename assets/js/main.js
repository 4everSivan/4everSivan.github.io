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
      applyTheme(currentTheme() === 'dark' ? 'light' : 'dark', true);
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

  const revealElements = Array.from(document.querySelectorAll('.reveal'));
  const prefersReducedMotion = window.matchMedia(
    '(prefers-reduced-motion: reduce)'
  );

  function showAll() {
    revealElements.forEach((element) => {
      element.classList.add('is-visible');
    });
  }

  if (prefersReducedMotion.matches || !('IntersectionObserver' in window)) {
    showAll();
  } else {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (!entry.isIntersecting) {
            return;
          }
          entry.target.classList.add('is-visible');
          observer.unobserve(entry.target);
        });
      },
      { rootMargin: '0px 0px -8% 0px', threshold: 0.05 }
    );

    revealElements.forEach((element) => observer.observe(element));
  }

  // 顶栏: 滚动超过阈值后渐显底部分界线.
  const masthead = document.querySelector('.masthead');
  if (masthead) {
    let frame = 0;
    function syncMasthead() {
      masthead.classList.toggle('is-scrolled', window.scrollY > 24);
      frame = 0;
    }
    window.addEventListener(
      'scroll',
      () => {
        if (frame === 0) {
          frame = window.requestAnimationFrame(syncMasthead);
        }
      },
      { passive: true }
    );
    syncMasthead();
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

  const count = cards.length;
  let index = count;
  let animating = false;

  function metrics() {
    const gap = parseFloat(getComputedStyle(original).columnGap) || 0;
    const cardWidth = cards[0].getBoundingClientRect().width;
    const viewWidth = carousel.getBoundingClientRect().width;
    return {
      center: (viewWidth - cardWidth) / 2,
      step: cardWidth + gap,
    };
  }

  function position(animate) {
    const { center, step } = metrics();
    if (!animate) {
      track.classList.add('is-resetting');
    }
    track.style.transform = `translateX(${center - index * step}px)`;
    if (!animate) {
      void track.offsetWidth;
      track.classList.remove('is-resetting');
    }
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
      position(false);
    }
  }

  function go(delta) {
    if (animating) {
      return;
    }
    animating = true;
    index += delta;
    position(true);

    function finish() {
      track.removeEventListener('transitionend', finish);
      normalize();
      animating = false;
    }

    track.addEventListener('transitionend', finish);
    window.setTimeout(() => {
      if (animating) {
        finish();
      }
    }, 900);
  }

  // 点击左右半卡切换; 中间的卡片保持为可点击链接.
  function centered(card) {
    const box = card.getBoundingClientRect();
    const view = carousel.getBoundingClientRect();
    const cardCenter = box.left + box.width / 2;
    const viewCenter = view.left + view.width / 2;
    return Math.abs(cardCenter - viewCenter) < box.width / 2;
  }

  sets.forEach((set) => {
    set.querySelectorAll('.skill-card').forEach((card) => {
      card.addEventListener('click', (event) => {
        if (animating) {
          event.preventDefault();
          return;
        }
        if (centered(card)) {
          return;
        }
        event.preventDefault();
        const box = card.getBoundingClientRect();
        const view = carousel.getBoundingClientRect();
        const cardCenter = box.left + box.width / 2;
        const viewCenter = view.left + view.width / 2;
        go(cardCenter < viewCenter ? -1 : 1);
      });
    });
  });

  position(false);

  let resizeFrame = 0;
  window.addEventListener('resize', () => {
    window.cancelAnimationFrame(resizeFrame);
    resizeFrame = window.requestAnimationFrame(() => {
      position(false);
    });
  });
})();
