/**
 * Тёмный Монастырь — Visual Effects Engine
 * ==========================================
 * Particle systems, screen effects, ambient animations.
 * This module is loaded before app.js and exposes window.DarkFX
 */

(function () {
  'use strict';

  // ===============================================================
  // PARTICLE SYSTEM — Embers, dust, and magic particles
  // ===============================================================

  const canvas = document.getElementById('particleCanvas');
  const ctx = canvas.getContext('2d');
  let particles = [];
  let animFrameId = null;
  let mouseX = -1000;
  let mouseY = -1000;

  function resizeCanvas() {
    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;
  }

  window.addEventListener('resize', resizeCanvas);
  resizeCanvas();

  // --- Particle class ---
  class Particle {
    constructor(opts = {}) {
      this.x = opts.x ?? Math.random() * canvas.width;
      this.y = opts.y ?? canvas.height + 10;
      this.size = opts.size ?? (Math.random() * 2 + 0.5);
      this.speedX = opts.speedX ?? (Math.random() - 0.5) * 0.3;
      this.speedY = opts.speedY ?? -(Math.random() * 0.8 + 0.2);
      this.life = opts.life ?? (Math.random() * 300 + 200);
      this.maxLife = this.life;
      this.color = opts.color ?? { r: 200, g: 140 + Math.random() * 60, b: 50 + Math.random() * 30 };
      this.type = opts.type ?? 'ember';
      this.wobble = Math.random() * Math.PI * 2;
      this.wobbleSpeed = (Math.random() - 0.5) * 0.02;
      this.glow = opts.glow ?? false;
    }

    update() {
      this.life--;
      this.wobble += this.wobbleSpeed;

      // Mouse repulsion
      const dx = this.x - mouseX;
      const dy = this.y - mouseY;
      const dist = Math.sqrt(dx * dx + dy * dy);
      if (dist < 100) {
        const force = (100 - dist) / 100 * 0.3;
        this.speedX += (dx / dist) * force;
        this.speedY += (dy / dist) * force;
      }

      this.x += this.speedX + Math.sin(this.wobble) * 0.15;
      this.y += this.speedY;

      // Slight deceleration
      this.speedX *= 0.995;
      this.speedY *= 0.998;

      return this.life > 0;
    }

    draw() {
      const alpha = Math.min(1, this.life / this.maxLife) * 0.7;
      const { r, g, b } = this.color;

      if (this.glow) {
        ctx.shadowColor = `rgba(${r}, ${g}, ${b}, ${alpha * 0.5})`;
        ctx.shadowBlur = this.size * 4;
      }

      ctx.beginPath();
      ctx.arc(this.x, this.y, this.size * (0.3 + alpha * 0.7), 0, Math.PI * 2);
      ctx.fillStyle = `rgba(${r}, ${g}, ${b}, ${alpha})`;
      ctx.fill();

      if (this.glow) {
        ctx.shadowBlur = 0;
      }
    }
  }

  // --- Ambient particle spawner: редкая пыль, оседающая сверху ---
  let spawnTimer = 0;
  const AMBIENT_SPAWN_RATE = 26; // frames between spawns

  function spawnAmbientParticle() {
    particles.push(new Particle({
      x: Math.random() * canvas.width,
      y: -6,
      size: Math.random() * 1.1 + 0.3,
      speedY: Math.random() * 0.22 + 0.07,
      speedX: (Math.random() - 0.5) * 0.12,
      life: Math.random() * 500 + 300,
      color: {
        r: 135 + Math.random() * 25,
        g: 133 + Math.random() * 25,
        b: 128 + Math.random() * 22,
      },
      glow: false,
    }));
  }

  // --- Animation loop ---
  function animateParticles() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    spawnTimer++;
    if (spawnTimer >= AMBIENT_SPAWN_RATE) {
      spawnTimer = 0;
      spawnAmbientParticle();
    }

    particles = particles.filter(p => {
      const alive = p.update();
      if (alive) p.draw();
      return alive;
    });

    animFrameId = requestAnimationFrame(animateParticles);
  }

  animateParticles();

  // Track mouse for particle interaction
  document.addEventListener('mousemove', e => {
    mouseX = e.clientX;
    mouseY = e.clientY;
  });

  // ===============================================================
  // BURST EFFECTS — Triggered by game events
  // ===============================================================

  /**
   * Spawn a burst of particles at a point.
   * @param {number} x
   * @param {number} y
   * @param {object} opts - count, color {r,g,b}, speed, size, life, glow
   */
  function particleBurst(x, y, opts = {}) {
    const count = opts.count ?? 20;
    const baseColor = opts.color ?? { r: 200, g: 169, b: 110 };

    for (let i = 0; i < count; i++) {
      const angle = (Math.PI * 2 / count) * i + (Math.random() - 0.5) * 0.5;
      const speed = (opts.speed ?? 2) * (0.5 + Math.random());

      particles.push(new Particle({
        x: x + (Math.random() - 0.5) * 10,
        y: y + (Math.random() - 0.5) * 10,
        speedX: Math.cos(angle) * speed,
        speedY: Math.sin(angle) * speed,
        size: (opts.size ?? 2) * (0.5 + Math.random()),
        life: (opts.life ?? 60) + Math.random() * 30,
        color: {
          r: baseColor.r + (Math.random() - 0.5) * 30,
          g: baseColor.g + (Math.random() - 0.5) * 30,
          b: baseColor.b + (Math.random() - 0.5) * 20,
        },
        glow: opts.glow ?? true,
      }));
    }
  }

  /**
   * Spawn upward-rising particles (fire, magic, etc.)
   */
  function particleRise(x, y, opts = {}) {
    const count = opts.count ?? 15;
    const baseColor = opts.color ?? { r: 200, g: 169, b: 110 };

    for (let i = 0; i < count; i++) {
      particles.push(new Particle({
        x: x + (Math.random() - 0.5) * 40,
        y: y + (Math.random() - 0.5) * 10,
        speedX: (Math.random() - 0.5) * 0.5,
        speedY: -(Math.random() * 2 + 1),
        size: (opts.size ?? 2) * (0.3 + Math.random()),
        life: (opts.life ?? 80) + Math.random() * 40,
        color: {
          r: baseColor.r + (Math.random() - 0.5) * 20,
          g: baseColor.g + (Math.random() - 0.5) * 20,
          b: baseColor.b + (Math.random() - 0.5) * 15,
        },
        glow: opts.glow ?? true,
      }));
    }
  }

  // ===============================================================
  // SCREEN EFFECTS — Full-screen visual impacts
  // ===============================================================

  /**
   * Screen shake effect.
   * @param {number} intensity - 1 (light) to 3 (heavy)
   */
  function screenShake(intensity = 1) {
    const container = document.getElementById('gameContainer');
    if (!container) return;

    container.classList.remove('shake');
    void container.offsetWidth; // reflow to restart
    container.style.setProperty('--shake-intensity', intensity);
    container.classList.add('shake');
    container.addEventListener('animationend', () => {
      container.classList.remove('shake');
    }, { once: true });
  }

  /**
   * Blood vignette — red flash at edges.
   */
  function bloodVignette() {
    const el = document.createElement('div');
    el.className = 'blood-vignette';
    document.body.appendChild(el);
    el.addEventListener('animationend', () => el.remove(), { once: true });
  }

  /**
   * Heal glow — green flash.
   */
  function healGlow() {
    const el = document.createElement('div');
    el.className = 'heal-glow';
    document.body.appendChild(el);
    el.addEventListener('animationend', () => el.remove(), { once: true });
  }

  /**
   * Rune flash — blue/arcane flash for magic.
   */
  function runeFlash() {
    const el = document.createElement('div');
    el.className = 'rune-flash';
    document.body.appendChild(el);
    el.addEventListener('animationend', () => el.remove(), { once: true });
  }

  /**
   * Gold sparkles — from a point, for loot.
   */
  function goldSparkles(x, y, count = 8) {
    for (let i = 0; i < count; i++) {
      const el = document.createElement('div');
      el.className = 'gold-sparkle';
      el.style.left = (x + (Math.random() - 0.5) * 60) + 'px';
      el.style.top = (y + (Math.random() - 0.5) * 30) + 'px';
      el.style.animationDelay = (Math.random() * 0.3) + 's';
      document.body.appendChild(el);
      el.addEventListener('animationend', () => el.remove(), { once: true });
    }
  }

  /**
   * Quest banner — dramatic text.
   */
  function questBanner(text) {
    const el = document.createElement('div');
    el.className = 'quest-banner';
    el.textContent = text;
    document.body.appendChild(el);
    el.addEventListener('animationend', () => el.remove(), { once: true });
  }

  /**
   * Floating damage/heal number.
   */
  function floatingNumber(x, y, text, type = 'damage') {
    const el = document.createElement('div');
    el.className = 'damage-number';
    if (type === 'heal') el.classList.add('damage-number--heal');
    if (type === 'gold') el.classList.add('damage-number--gold');
    el.textContent = text;
    el.style.left = x + 'px';
    el.style.top = y + 'px';
    document.body.appendChild(el);
    el.addEventListener('animationend', () => el.remove(), { once: true });
  }

  /**
   * Level-up rays.
   */
  function levelUpRays() {
    const el = document.createElement('div');
    el.className = 'levelup-rays';
    document.body.appendChild(el);
    el.addEventListener('animationend', () => el.remove(), { once: true });
  }

  // ===============================================================
  // STAT ANIMATIONS — Flash stat elements on change
  // ===============================================================

  function flashStat(elementId, type = 'default') {
    const el = document.getElementById(elementId);
    if (!el) return;

    el.classList.remove('stat-flash', 'stat-flash--damage', 'stat-flash--heal');
    void el.offsetWidth; // reflow

    if (type === 'damage') {
      el.classList.add('stat-flash', 'stat-flash--damage');
    } else if (type === 'heal') {
      el.classList.add('stat-flash', 'stat-flash--heal');
    } else {
      el.classList.add('stat-flash');
    }

    el.addEventListener('animationend', () => {
      el.classList.remove('stat-flash', 'stat-flash--damage', 'stat-flash--heal');
    }, { once: true });
  }

  // ===============================================================
  // TYPEWRITER EFFECT — Character-by-character text reveal
  // ===============================================================

  /**
   * Type text into an element character by character.
   * @param {HTMLElement} element
   * @param {string} text
   * @param {number} speed - ms per character
   * @returns {Promise}
   */
  function typewriterText(element, text, speed = 15) {
    return new Promise(resolve => {
      element.textContent = '';
      element.classList.add('typewriter');
      let i = 0;

      function tick() {
        if (i < text.length) {
          element.textContent += text[i];
          i++;
          // Scroll parent story log
          const storyLog = document.getElementById('storyLog');
          if (storyLog) storyLog.scrollTop = storyLog.scrollHeight;
          setTimeout(tick, speed);
        } else {
          element.classList.remove('typewriter');
          resolve();
        }
      }

      tick();
    });
  }

  // ===============================================================
  // SEND ACTION EFFECT — Visual feedback on form submit
  // ===============================================================

  function sendActionEffect() {
    const form = document.getElementById('actionForm');
    const btn = document.getElementById('btnSubmit');
    if (!form || !btn) return;

    // Button pulse
    btn.style.transform = 'scale(0.85)';
    btn.style.boxShadow = '0 0 20px rgba(200, 169, 110, 0.3)';
    setTimeout(() => {
      btn.style.transform = '';
      btn.style.boxShadow = '';
    }, 200);

    // Particle burst from input
    const rect = btn.getBoundingClientRect();
    particleBurst(rect.left + rect.width / 2, rect.top + rect.height / 2, {
      count: 8,
      color: { r: 165, g: 162, b: 155 },
      speed: 1.2,
      size: 1.2,
      life: 35,
      glow: false,
    });
  }

  // ===============================================================
  // BUTTON HOVER EFFECTS — Particles on hover
  // ===============================================================

  function initButtonEffects() {
    document.querySelectorAll('.cmd-btn').forEach(btn => {
      btn.addEventListener('mouseenter', () => {
        const rect = btn.getBoundingClientRect();
        particleBurst(
          rect.left + rect.width / 2,
          rect.top + rect.height / 2,
          {
            count: 4,
            color: { r: 150, g: 148, b: 142 },
            speed: 0.6,
            size: 0.9,
            life: 26,
            glow: false,
          }
        );
      });
    });
  }

  // Initialize button effects when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initButtonEffects);
  } else {
    initButtonEffects();
  }

  // ===============================================================
  // AMBIENT AUDIO CONTEXT (visual-only, no actual sound)
  // Title flicker on hover
  // ===============================================================

  function initTitleEffect() {
    const title = document.getElementById('headerTitle');
    if (!title) return;

    title.addEventListener('mouseenter', () => {
      const rect = title.getBoundingClientRect();
      particleRise(rect.left + rect.width / 2, rect.top + rect.height, {
        count: 10,
        color: { r: 150, g: 148, b: 142 },
        size: 1.1,
        life: 50,
        glow: false,
      });
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initTitleEffect);
  } else {
    initTitleEffect();
  }

  // ===============================================================
  // LOCATION CHANGE TRANSITION
  // ===============================================================

  function locationTransition() {
    // Brief flash and particle burst centered on screen
    const cx = window.innerWidth / 2;
    const cy = window.innerHeight / 2;

    particleBurst(cx, cy, {
      count: 30,
      color: { r: 107, g: 91, b: 79 },
      speed: 3,
      size: 2,
      life: 50,
    });
  }

  // ===============================================================
  // EXPOSE PUBLIC API
  // ===============================================================

  window.DarkFX = {
    // Particle effects
    particleBurst,
    particleRise,

    // Screen effects
    screenShake,
    bloodVignette,
    healGlow,
    runeFlash,
    goldSparkles,
    questBanner,
    floatingNumber,
    levelUpRays,

    // Stat animations
    flashStat,

    // Text
    typewriterText,

    // UI effects
    sendActionEffect,
    locationTransition,
  };

})();
