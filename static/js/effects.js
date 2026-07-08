/**
 * Dark Monastery — Visual Effects Engine (UNDERTALE-style)
 * =========================================================
 * Minimalist: typewriter, screen shake, stat flash only.
 * Loaded before app.js, exposes window.DarkFX
 */
(function () {
  'use strict';

  // ================================================================
  // TYPEWRITER EFFECT — character-by-character text reveal
  // ================================================================

  /**
   * Types text into an element character by character.
   * @param {HTMLElement} element
   * @param {string} text
   * @param {number} speed — ms per character (default 50 for grim pacing)
   * @returns {Promise}
   */
  function typewriterText(element, text, speed) {
    speed = speed || 50;
    return new Promise(function (resolve) {
      element.textContent = '';
      element.classList.add('typewriter');
      var i = 0;

      function tick() {
        if (i < text.length) {
          element.textContent += text[i];
          i++;
          var storyLog = document.getElementById('storyLog');
          if (storyLog) storyLog.scrollTop = storyLog.scrollHeight;

          // Longer pause on punctuation
          var pauseChars = ['.', '!', '?', '\n'];
          var extra = pauseChars.indexOf(text[i - 1]) !== -1 ? speed * 2 : 0;
          setTimeout(tick, speed + extra);
        } else {
          element.classList.remove('typewriter');
          resolve();
        }
      }

      tick();
    });
  }

  // ================================================================
  // SCREEN SHAKE — pixel screen shake
  // ================================================================

  /**
   * Screen shake effect on the game container.
   * @param {number} intensity — unused in pixel version
   */
  function screenShake(intensity) {
    var container = document.getElementById('gameContainer');
    if (!container) return;

    container.classList.remove('shake');
    void container.offsetWidth;
    container.classList.add('shake');
    container.addEventListener('animationend', function () {
      container.classList.remove('shake');
    }, { once: true });
  }

  // ================================================================
  // STAT FLASH — highlight stat change
  // ================================================================

  /**
   * Flashes a stat element on change.
   * @param {string} elementId
   * @param {string} type — 'default', 'damage', 'heal'
   */
  function flashStat(elementId, type) {
    var el = document.getElementById(elementId);
    if (!el) return;

    el.classList.remove('stat-flash', 'stat-flash--damage', 'stat-flash--heal');
    void el.offsetWidth;

    if (type === 'damage') {
      el.classList.add('stat-flash', 'stat-flash--damage');
    } else if (type === 'heal') {
      el.classList.add('stat-flash', 'stat-flash--heal');
    } else {
      el.classList.add('stat-flash');
    }

    el.addEventListener('animationend', function () {
      el.classList.remove('stat-flash', 'stat-flash--damage', 'stat-flash--heal');
    }, { once: true });
  }

  // ================================================================
  // PUBLIC API
  // ================================================================

  window.DarkFX = {
    typewriterText: typewriterText,
    screenShake: screenShake,
    flashStat: flashStat,
  };

})();
