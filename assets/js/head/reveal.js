// 首页滚动入场与数字滚动动效. 渐进增强: 无 JS 时内容完整可见,
// prefers-reduced-motion 或 IntersectionObserver 不可用时直接呈现终态.
document.documentElement.classList.add('js');

(function () {
  'use strict';

  var reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');

  function formatNumber(value, suffix) {
    return new Intl.NumberFormat('zh-CN').format(value) + (suffix || '');
  }

  function animateCount(el) {
    var target = parseInt(el.getAttribute('data-count-to'), 10);
    if (isNaN(target)) {
      return;
    }
    var suffix = el.getAttribute('data-count-suffix') || '';
    if (reducedMotion.matches || !window.requestAnimationFrame) {
      el.textContent = formatNumber(target, suffix);
      return;
    }
    var duration = 800;
    var start = null;
    function step(timestamp) {
      if (start === null) {
        start = timestamp;
      }
      var progress = Math.min((timestamp - start) / duration, 1);
      var eased = 1 - Math.pow(1 - progress, 3);
      el.textContent = formatNumber(Math.round(target * eased), suffix);
      if (progress < 1) {
        window.requestAnimationFrame(step);
      }
    }
    window.requestAnimationFrame(step);
  }

  function init() {
    var home = document.querySelector('.hextra-home');
    if (!home) {
      return;
    }

    var targets = home.querySelectorAll('h2, .hextra-cards, .category-wall, .recent-docs, .hextra-feature-grid');
    targets.forEach(function (el) {
      el.setAttribute('data-reveal', '');
      var items = el.children;
      for (var i = 0; i < items.length; i++) {
        if (items[i].matches('a, li')) {
          items[i].classList.add('reveal-item');
          items[i].style.setProperty('--i', i);
        }
      }
    });

    var counters = home.querySelectorAll('[data-count-to]');

    // 降级路径: 全部直接可见, 数字写终值
    if (reducedMotion.matches || !('IntersectionObserver' in window)) {
      targets.forEach(function (el) {
        el.classList.add('revealed');
      });
      counters.forEach(function (el) {
        animateCount(el);
      });
      return;
    }

    // 动效路径: 数字先归零, 进入视口后再滚动计数, 避免首屏外数字提前完成
    counters.forEach(function (el) {
      el.textContent = formatNumber(0, el.getAttribute('data-count-suffix'));
    });

    var observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (!entry.isIntersecting) {
          return;
        }
        var el = entry.target;
        el.classList.add('revealed');
        el.querySelectorAll('[data-count-to]').forEach(animateCount);
        observer.unobserve(el);
      });
    }, { threshold: 0.15 });

    targets.forEach(function (el) {
      observer.observe(el);
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
