/**
 * Тёмный Монастырь — Application Entry Point
 * ============================================
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

      // Grand entrance effect
      FX.particleBurst(window.innerWidth / 2, window.innerHeight / 2, {
        count: 40,
        color: { r: 200, g: 169, b: 110 },
        speed: 3,
        size: 2,
        life: 80,
        glow: true,
      });

      WS.connect();
    } catch (err) {
      UI.addStoryEntry('⚠ Ошибка: ' + err.message, false, false, true);
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
        UI.addStoryEntry('💾 Игра сохранена в свиток.', false, false, false);
        FX.runeFlash();
        FX.questBanner('💾 Сохранено');
      }
    } catch (err) {
      UI.addStoryEntry('⚠ Ошибка сохранения: ' + err.message, false, false, true);
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
      UI.addStoryEntry('📂 Свиток загружен.', false, false, false);
      UI.addStoryEntry(data.state.message, true);
      GS.currentTurnCount = data.turn_count;

      FX.runeFlash();
      FX.locationTransition();

      WS.connect();
      closeSavesModal();
    } catch (err) {
      UI.addStoryEntry('⚠ Ошибка загрузки: ' + err.message, false, false, true);
    }
  }

  async function showSaves() {
    try {
      const resp = await fetch('/api/game/saves');
      const data = await resp.json();
      const saves = data.saves || [];

      if (saves.length === 0) {
        savesList.innerHTML = '<p style="color:var(--stone);text-align:center;padding:1rem;">Свитков сохранений не найдено</p>';
      } else {
        savesList.innerHTML = saves.map(s => `
          <div class="save-item" data-id="${esc(s.session_id)}">
            <span class="save-item__info">${esc(s.location)} — ход ${s.turn_count}</span>
            <span class="save-item__meta">${esc(s.session_id)}</span>
          </div>
        `).join('');

        savesList.querySelectorAll('.save-item').forEach(el => {
          el.addEventListener('click', () => loadGame(el.dataset.id));
        });
      }

      savesModal.hidden = false;
    } catch (err) {
      UI.addStoryEntry('⚠ Ошибка: ' + err.message, false, false, true);
    }
  }

  function closeSavesModal() {
    savesModal.hidden = true;
  }

  // ===============================================================
  // CUSTOM CONFIRMATION DIALOG
  // ===============================================================

  function showConfirm(title, text) {
    return new Promise(resolve => {
      const overlay = document.createElement('div');
      overlay.className = 'confirm-overlay';

      overlay.innerHTML = `
        <div class="confirm-dialog">
          <div class="confirm-dialog__title">${esc(title)}</div>
          <div class="confirm-dialog__text">${esc(text)}</div>
          <div class="confirm-dialog__buttons">
            <button class="cmd-btn cmd-btn--danger" id="confirmYes">Да</button>
            <button class="cmd-btn" id="confirmNo">Отмена</button>
          </div>
        </div>
      `;

      document.body.appendChild(overlay);

      overlay.querySelector('#confirmYes').addEventListener('click', () => {
        overlay.remove();
        resolve(true);
      });

      overlay.querySelector('#confirmNo').addEventListener('click', () => {
        overlay.remove();
        resolve(false);
      });

      overlay.addEventListener('click', e => {
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

  actionForm.addEventListener('submit', e => {
    e.preventDefault();
    if (GS.isThinking) return;
    WS.sendAction(actionInput.value);
  });

  document.getElementById('btnNewGame').addEventListener('click', async () => {
    const confirmed = await showConfirm(
      'Новая игра',
      'Начать новое приключение? Текущий прогресс будет утерян в тумане.'
    );
    if (confirmed) {
      FX.screenShake(2);
      FX.bloodVignette();
      setTimeout(() => newGame(), 300);
    }
  });

  document.getElementById('btnSave').addEventListener('click', saveGame);
  document.getElementById('btnLoad').addEventListener('click', showSaves);
  document.getElementById('btnCloseModal').addEventListener('click', closeSavesModal);

  savesModal.addEventListener('click', e => {
    if (e.target === savesModal) closeSavesModal();
  });

  document.addEventListener('keydown', e => {
    if (e.key === 'Escape') {
      if (!savesModal.hidden) closeSavesModal();
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
