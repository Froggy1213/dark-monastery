/**
 * Dark Monastery — Application Entry Point (UNDERTALE-style)
 * ============================================================
 * Session management, event wiring, initialization.
 * Depends on: window.DarkFX, window.GameState, window.GameWS, window.GameUI
 */
(function () {
  'use strict';

  const FX = window.DarkFX;
  const GS = window.GameState;
  const WS = window.GameWS;
  const UI = window.GameUI;

  // DOM references
  const actionForm = document.getElementById('actionForm');
  const actionInput = document.getElementById('actionInput');
  const savesModal = document.getElementById('savesModal');
  const savesList = document.getElementById('savesList');
  const storyInner = document.getElementById('storyInner');

  // ===============================================================
  // SESSION MANAGEMENT
  // ===============================================================

  async function newGame() {
    try {
      const resp = await fetch('/api/game/new', { method: 'POST' });
      const data = await resp.json();

      if (data.error) {
        UI.addStoryEntry('⚠ ' + data.error, false, false, true);
        return;
      }

      GS.sessionId = data.session_id;
      GS.previousState = null;
      storyInner.innerHTML = '';

      UI.updateState(data.state, data.turn_count);
      UI.addStoryEntry(data.state.message, true);
      GS.currentTurnCount = data.turn_count;

      UI.closeStatsMenu();
      WS.connect();
    } catch (err) {
      UI.addStoryEntry('⚠ Error: ' + err.message, false, false, true);
    }
  }

  async function saveGame() {
    if (!GS.sessionId) return;

    try {
      const resp = await fetch('/api/game/save', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: GS.sessionId }),
      });
      const data = await resp.json();

      if (data.status === 'ok') {
        UI.addStoryEntry('💾 Game saved.', false, false, false);
      }
    } catch (err) {
      UI.addStoryEntry('⚠ Save error: ' + err.message, false, false, true);
    }
  }

  async function loadGame(saveId) {
    try {
      const resp = await fetch('/api/game/load', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: saveId }),
      });
      const data = await resp.json();

      if (data.error) {
        UI.addStoryEntry('⚠ ' + data.error, false, false, true);
        return;
      }

      GS.sessionId = data.session_id;
      GS.previousState = null;
      storyInner.innerHTML = '';

      UI.updateState(data.state, data.turn_count);
      UI.addStoryEntry('📂 Game loaded.', false, false, false);
      UI.addStoryEntry(data.state.message, true);
      UI.renderChoices(data.state.choices || []);
      GS.currentTurnCount = data.turn_count;

      UI.closeStatsMenu();
      WS.connect();
      closeSavesModal();
    } catch (err) {
      UI.addStoryEntry('⚠ Load error: ' + err.message, false, false, true);
    }
  }

  async function showSaves() {
    try {
      const resp = await fetch('/api/game/saves');
      const data = await resp.json();
      const saves = data.saves || [];

      if (saves.length === 0) {
        savesList.innerHTML = '<p style="color:var(--text-dark);text-align:center;padding:1rem;">No scrolls found</p>';
      } else {
        savesList.innerHTML = saves.map(function (s) {
          return '<div class="save-item" data-id="' + esc(s.session_id) + '">' +
            '<span class="save-item__info">' + esc(s.location) + ' — turn ' + s.turn_count + '</span>' +
            '<span class="save-item__meta">' + esc(s.session_id) + '</span>' +
            '</div>';
        }).join('');

        savesList.querySelectorAll('.save-item').forEach(function (el) {
          el.addEventListener('click', function () { loadGame(el.dataset.id); });
        });
      }

      savesModal.hidden = false;
    } catch (err) {
      UI.addStoryEntry('⚠ Error: ' + err.message, false, false, true);
    }
  }

  function closeSavesModal() {
    savesModal.hidden = true;
  }

  // ===============================================================
  // CUSTOM CONFIRMATION DIALOG
  // ===============================================================

  function showConfirm(title, text) {
    return new Promise(function (resolve) {
      const overlay = document.createElement('div');
      overlay.className = 'confirm-overlay';

      overlay.innerHTML =
        '<div class="confirm-dialog">' +
        '<div class="confirm-dialog__title">' + esc(title) + '</div>' +
        '<div class="confirm-dialog__text">' + esc(text) + '</div>' +
        '<div class="confirm-dialog__buttons">' +
        '<button class="cmd-btn cmd-btn--danger" id="confirmYes">Yes</button>' +
        '<button class="cmd-btn" id="confirmNo">Cancel</button>' +
        '</div>' +
        '</div>';

      document.body.appendChild(overlay);

      overlay.querySelector('#confirmYes').addEventListener('click', function () {
        overlay.remove();
        resolve(true);
      });

      overlay.querySelector('#confirmNo').addEventListener('click', function () {
        overlay.remove();
        resolve(false);
      });

      overlay.addEventListener('click', function (e) {
        if (e.target === overlay) {
          overlay.remove();
          resolve(false);
        }
      });
    });
  }

  // ===============================================================
  // EVENT LISTENERS
  // ===============================================================

  actionForm.addEventListener('submit', function (e) {
    e.preventDefault();
    if (GS.isThinking) return;
    WS.sendAction(actionInput.value);
  });

  document.getElementById('btnNewGame').addEventListener('click', async function () {
    const confirmed = await showConfirm(
      'New Game',
      'Begin a new adventure? Current progress will be lost.'
    );
    if (confirmed) {
      FX.screenShake(2);
      setTimeout(function () { newGame(); }, 300);
    }
  });

  document.getElementById('btnSave').addEventListener('click', saveGame);
  document.getElementById('btnLoad').addEventListener('click', showSaves);
  document.getElementById('btnCloseModal').addEventListener('click', closeSavesModal);

  savesModal.addEventListener('click', function (e) {
    if (e.target === savesModal) closeSavesModal();
  });

  // Stats menu
  document.getElementById('btnMenu').addEventListener('click', function () {
    UI.openStatsMenu();
  });

  document.getElementById('btnCloseStats').addEventListener('click', function () {
    UI.closeStatsMenu();
  });

  // Close stats overlay by clicking outside
  var statsOverlay = document.getElementById('statsOverlay');
  if (statsOverlay) {
    statsOverlay.addEventListener('click', function (e) {
      if (e.target === statsOverlay) UI.closeStatsMenu();
    });
  }

  // Keyboard shortcuts
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') {
      var statsOverlay = document.getElementById('statsOverlay');
      if (statsOverlay && !statsOverlay.hidden) {
        UI.closeStatsMenu();
        return;
      }
      if (!savesModal.hidden) {
        closeSavesModal();
        return;
      }
    }
  });

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
  // INITIALIZATION
  // ===============================================================

  async function init() {
    await newGame();
  }

  init();

})();
