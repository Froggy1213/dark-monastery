/**
 * Dark Monastery — UI Module (UNDERTALE-style)
 * ==============================================
 * HUD updates, stats menu, story log, thinking indicator.
 * Depends on: window.GameState, window.DarkFX
 */
(function () {
  'use strict';

  const FX = window.DarkFX;
  const GS = window.GameState;

  const storyLog = document.getElementById('storyLog');
  const storyInner = document.getElementById('storyInner');
  const thinkingIndicator = document.getElementById('thinkingIndicator');
  const actionInput = document.getElementById('actionInput');

  // ===============================================================
  // HUD HELPERS
  // ===============================================================

  /**
   * Converts condition string to HP percentage for the bar.
   */
  function conditionToHP(condition) {
    if (!condition) return { pct: 0.8, text: '??/20' };
    var lower = condition.toLowerCase();

    // Try to parse "Healthy (15/20)" format
    var match = condition.match(/\((\d+)\s*\/\s*(\d+)\)/);
    if (match) {
      var cur = parseInt(match[1], 10);
      var max = parseInt(match[2], 10);
      return { pct: Math.min(1, cur / max), text: cur + '/' + max };
    }

    // Keyword mapping (English)
    var map = [
      { keys: ['healthy', 'unharmed', 'fine', 'fit'], pct: 1.0, text: '20/20' },
      { keys: ['bruised', 'scratched', 'light', 'minor', 'slight'], pct: 0.8, text: '16/20' },
      { keys: ['wounded', 'hurt', 'injured', 'bloody'], pct: 0.55, text: '11/20' },
      { keys: ['severe', 'serious', 'badly', 'heavy', 'critical'], pct: 0.3, text: '6/20' },
      { keys: ['dying', 'mortal', 'death', 'agon', 'fatal', 'dead'], pct: 0.12, text: '2/20' },
    ];

    for (var i = 0; i < map.length; i++) {
      for (var j = 0; j < map[i].keys.length; j++) {
        if (lower.indexOf(map[i].keys[j]) !== -1) {
          return { pct: map[i].pct, text: map[i].text };
        }
      }
    }

    return { pct: 0.8, text: '??/20' };
  }

  function updateHUD(state) {
    // HP bar
    var hp = conditionToHP(state.condition);
    var hpFill = document.getElementById('hpBarFill');
    var hpText = document.getElementById('hpText');
    if (hpFill) hpFill.style.width = (hp.pct * 100) + '%';
    if (hpText) hpText.textContent = hp.text;

    // Location
    var locEl = document.getElementById('hudLocation');
    if (locEl && state.location) {
      locEl.textContent = state.location;
    }

    // Gold
    var goldEl = document.getElementById('hudGold');
    if (goldEl) {
      goldEl.textContent = '☠ ' + (state.gold ?? 0);
    }
  }

  // ===============================================================
  // STATE UPDATE — diff-based, HUD + stats menu + sidebar
  // ===============================================================

  function setStatText(id, value) {
    var el = document.getElementById(id);
    if (el) el.textContent = value;
  }

  function setBothText(modalId, sidebarId, value) {
    setStatText(modalId, value);
    setStatText(sidebarId, value);
  }

  function setHTML(id, html) {
    var el = document.getElementById(id);
    if (el) el.innerHTML = html;
  }

  function setBothHTML(modalId, sidebarId, html) {
    setHTML(modalId, html);
    setHTML(sidebarId, html);
  }

  function updateState(state, turnCount) {
    var prev = GS.previousState;

    // Update HUD
    updateHUD(state);

    // --- Location ---
    setBothText('statLocation', 'sbLocation', state.location || '—');

    // --- Condition ---
    var condEl = document.getElementById('statCondition');
    if (condEl && condEl.textContent !== (state.condition || '—')) {
      setBothText('statCondition', 'sbCondition', state.condition || '—');
      FX.flashStat('statCondition');
      FX.flashStat('sbCondition');
    }

    // --- Sanity ---
    setBothText('statSanity', 'sbSanity', state.sanity || '—');

    // --- Gold ---
    var goldEl = document.getElementById('statGold');
    var prevGold = prev ? (prev.gold ?? 0) : 0;
    var newGold = state.gold ?? 0;
    if (goldEl) {
      if (prevGold !== newGold) {
        setBothText('statGold', 'sbGold', newGold);
        if (newGold > prevGold) {
          FX.flashStat('statGold');
          FX.flashStat('sbGold');
        } else {
          FX.flashStat('statGold', 'damage');
          FX.flashStat('sbGold', 'damage');
        }
      } else {
        setBothText('statGold', 'sbGold', newGold);
      }
    }

    // --- Equipped ---
    var eqEl = document.getElementById('statEquipped');
    if (eqEl && eqEl.textContent !== (state.equipped || '—')) {
      setBothText('statEquipped', 'sbEquipped', state.equipped || '—');
      FX.flashStat('statEquipped');
      FX.flashStat('sbEquipped');
    }

    // --- Inventory ---
    var newInv = (state.inventory || []).map(function (i) { return esc(i); });
    var invHTML = newInv.map(function (i) { return '<li>' + i + '</li>'; }).join('');
    setBothHTML('statInventory', 'sbInventory', invHTML);

    // --- Skills ---
    var newSkills = (state.skills || []).map(function (s) { return esc(s); });
    var skillsHTML = newSkills.map(function (s) { return '<li>' + s + '</li>'; }).join('');
    setBothHTML('statSkills', 'sbSkills', skillsHTML);

    // --- Quests ---
    var newQuests = (state.active_quests || []).map(function (q) { return esc(q); });
    var questHTML = newQuests.map(function (q) { return '<li>' + q + '</li>'; }).join('');
    setBothHTML('statQuests', 'sbQuests', questHTML);

    // Store for next diff
    GS.previousState = JSON.parse(JSON.stringify(state));
  }

  // ===============================================================
  // STORY LOG
  // ===============================================================

  function addStoryEntry(text, isResponse, isAction, isError) {
    var placeholder = storyInner.querySelector('.story-placeholder');
    if (placeholder) placeholder.remove();

    var entry = document.createElement('div');
    entry.className = 'story-entry';

    if (isAction) {
      var actionEl = document.createElement('div');
      actionEl.className = 'story-action';
      actionEl.textContent = text;
      entry.appendChild(actionEl);
    } else if (isError) {
      var msgEl = document.createElement('div');
      msgEl.className = 'story-message story-message--error';
      msgEl.textContent = text;
      entry.appendChild(msgEl);
    } else {
      var msgEl = document.createElement('div');
      msgEl.className = isResponse ? 'story-message' : 'story-message story-message--system';

      if (isResponse && text.length > 25) {
        entry.appendChild(msgEl);
        storyInner.appendChild(entry);
        FX.typewriterText(msgEl, text, 50);
        scrollToBottom();
        enableInput();
        return;
      } else {
        msgEl.textContent = text;
      }

      entry.appendChild(msgEl);
    }

    storyInner.appendChild(entry);
    scrollToBottom();

    if (isResponse || isError) {
      enableInput();
    }
  }

  function scrollToBottom() {
    setTimeout(function () {
      storyLog.scrollTop = storyLog.scrollHeight;
    }, 50);
  }

  function enableInput() {
    if (actionInput) {
      actionInput.disabled = false;
      actionInput.focus();
    }
  }

  // ===============================================================
  // THINKING INDICATOR
  // ===============================================================

  function updateThinking(show) {
    if (show) {
      thinkingIndicator.classList.add('thinking-indicator--visible');
      if (actionInput) actionInput.disabled = true;
    } else {
      thinkingIndicator.classList.remove('thinking-indicator--visible');
    }
  }

  // ===============================================================
  // STATS MENU (dossier)
  // ===============================================================

  function openStatsMenu() {
    var overlay = document.getElementById('statsOverlay');
    if (overlay) overlay.hidden = false;
  }

  function closeStatsMenu() {
    var overlay = document.getElementById('statsOverlay');
    if (overlay) overlay.hidden = true;
  }

  // ===============================================================
  // HELPERS
  // ===============================================================

  function esc(str) {
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  // ===============================================================
  // PUBLIC API
  // ===============================================================

  window.GameUI = {
    updateState: updateState,
    addStoryEntry: addStoryEntry,
    updateThinking: updateThinking,
    scrollToBottom: scrollToBottom,
    enableInput: enableInput,
    openStatsMenu: openStatsMenu,
    closeStatsMenu: closeStatsMenu,
  };

})();
