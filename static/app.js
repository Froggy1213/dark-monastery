/**
 * Тёмный Монастырь — Game Application
 * =====================================
 * Core game logic, WebSocket communication, state management.
 * Integrates with DarkFX (effects.js) for visual effects.
 */

(function () {
  'use strict';

  const FX = window.DarkFX;

  // ===============================================================
  // STATE
  // ===============================================================

  let sessionId = null;
  let isThinking = false;
  let previousState = null; // for diffing and triggering effects

  // ===============================================================
  // DOM REFERENCES
  // ===============================================================

  const storyInner = document.getElementById('storyInner');
  const storyLog = document.getElementById('storyLog');
  const actionForm = document.getElementById('actionForm');
  const actionInput = document.getElementById('actionInput');
  const thinkingIndicator = document.getElementById('thinkingIndicator');
  const savesModal = document.getElementById('savesModal');
  const savesList = document.getElementById('savesList');

  // ===============================================================
  // WEBSOCKET CONNECTION
  // ===============================================================

  let ws = null;
  let wsReconnectTimer = null;

  function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${protocol}//${window.location.host}/ws/game${sessionId ? '?session_id=' + sessionId : ''}`;

    ws = new WebSocket(url);

    ws.onopen = () => {
      console.log('[WS] Connected');
      if (wsReconnectTimer) {
        clearTimeout(wsReconnectTimer);
        wsReconnectTimer = null;
      }
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        handleWSMessage(data);
      } catch (err) {
        console.error('[WS] Parse error:', err);
      }
    };

    ws.onclose = () => {
      console.log('[WS] Disconnected');
      if (!wsReconnectTimer) {
        wsReconnectTimer = setTimeout(connectWebSocket, 3000);
      }
    };

    ws.onerror = (err) => {
      console.error('[WS] Error:', err);
    };
  }

  function handleWSMessage(data) {
    switch (data.type) {
      case 'update':
        isThinking = false;
        updateThinking(false);
        updateState(data.state, data.turn_count);
        addStoryEntry(data.state.message, true);
        break;

      case 'thinking':
        isThinking = true;
        updateThinking(true);
        break;

      case 'error':
        isThinking = false;
        updateThinking(false);
        addStoryEntry('⚠ ' + data.message, false, false, true);
        break;

      case 'pong':
        break;

      default:
        console.log('[WS] Unknown type:', data.type);
    }
  }

  // ===============================================================
  // SEND ACTION
  // ===============================================================

  function sendAction(text) {
    if (!text.trim()) return;

    // Visual feedback
    FX.sendActionEffect();

    // Show player action in log
    addStoryEntry(text, false, true);

    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({
        type: 'action',
        text: text,
        session_id: sessionId,
      }));
    } else {
      // Fallback HTTP
      fetch('/api/game/action', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: sessionId, action: text }),
      })
        .then(r => r.json())
        .then(data => {
          isThinking = false;
          updateThinking(false);
          if (data.error) {
            addStoryEntry('⚠ ' + data.error, false, false, true);
          } else {
            updateState(data.state, data.turn_count);
            addStoryEntry(data.state.message, true);
          }
        })
        .catch(err => {
          isThinking = false;
          updateThinking(false);
          addStoryEntry('⚠ Ошибка соединения: ' + err.message, false, false, true);
        });
    }

    isThinking = true;
    updateThinking(true);
    actionInput.value = '';
    actionInput.disabled = true;
  }

  // ===============================================================
  // UI UPDATES
  // ===============================================================

  function updateState(state, turnCount) {
    const prev = previousState;

    // --- Location ---
    const locationEl = document.getElementById('statLocation');
    if (locationEl.textContent !== (state.location || '—')) {
      locationEl.textContent = state.location || '—';
      FX.flashStat('statLocation');
      if (prev && prev.location !== state.location) {
        FX.locationTransition();
      }
    }

    // --- Condition ---
    const condEl = document.getElementById('statCondition');
    if (condEl.textContent !== (state.condition || '—')) {
      condEl.textContent = state.condition || '—';
      FX.flashStat('statCondition');
    }

    // --- Sanity ---
    const sanityEl = document.getElementById('statSanity');
    const sanityIcon = document.getElementById('sanityIcon');
    const sanity = state.sanity || '—';
    if (sanityEl.textContent !== sanity) {
      sanityEl.textContent = sanity;
      FX.flashStat('statSanity');
      // Change icon based on state
      const lower = sanity.toLowerCase();
      if (lower.includes('нестабил') || lower.includes('тревож')) {
        sanityIcon.textContent = '🜏';
        sanityIcon.style.color = '#c0392b';
      } else if (lower.includes('безум') || lower.includes('хаос')) {
        sanityIcon.textContent = '☠';
        sanityIcon.style.color = '#8b1a1a';
      } else {
        sanityIcon.textContent = '🜏';
        sanityIcon.style.color = '';
      }
    }

    // --- HP --- with damage/heal effects
    const hp = state.hp ?? 0;
    const maxHp = state.max_hp ?? 20;
    const hpPct = maxHp > 0 ? Math.max(0, Math.min(100, (hp / maxHp) * 100)) : 100;
    const hpBar = document.getElementById('hpBar');
    const hpText = document.getElementById('hpText');

    hpBar.style.width = hpPct + '%';
    hpText.textContent = `${hp}/${maxHp}`;

    if (prev) {
      const prevHp = prev.hp ?? 0;
      const hpDiff = hp - prevHp;

      if (hpDiff < 0) {
        // DAMAGE taken!
        FX.flashStat('hpText', 'damage');
        FX.screenShake(Math.min(3, Math.abs(hpDiff) / 3));
        FX.bloodVignette();

        // Floating damage number
        const hpRect = hpBar.getBoundingClientRect();
        FX.floatingNumber(
          hpRect.left + hpRect.width / 2,
          hpRect.top,
          hpDiff.toString(),
          'damage'
        );

        // Blood particle burst
        FX.particleBurst(hpRect.left + hpRect.width * (hpPct / 100), hpRect.top + hpRect.height / 2, {
          count: Math.min(30, Math.abs(hpDiff) * 3),
          color: { r: 180, g: 30, b: 30 },
          speed: 2,
          size: 2,
          life: 50,
        });

        // Critical health warning
        if (hpPct <= 25) {
          hpBar.style.boxShadow = '0 0 20px rgba(192, 57, 43, 0.6)';
        }
      } else if (hpDiff > 0) {
        // HEALED!
        FX.flashStat('hpText', 'heal');
        FX.healGlow();

        const hpRect = hpBar.getBoundingClientRect();
        FX.floatingNumber(
          hpRect.left + hpRect.width / 2,
          hpRect.top,
          '+' + hpDiff,
          'heal'
        );

        FX.particleRise(hpRect.left + hpRect.width / 2, hpRect.top, {
          count: 10,
          color: { r: 74, g: 160, b: 58 },
          size: 1.5,
          life: 60,
        });

        hpBar.style.boxShadow = '';
      }
    }

    // --- Mana ---
    const mana = state.mana ?? 0;
    const maxMana = 100; // Assume max mana
    const manaPct = Math.max(0, Math.min(100, (mana / maxMana) * 100));
    const manaBar = document.getElementById('manaBar');
    const manaText = document.getElementById('manaText');

    manaBar.style.width = manaPct + '%';
    manaText.textContent = mana;

    if (prev && (prev.mana ?? 0) !== mana) {
      FX.flashStat('manaText');
      if (mana > (prev.mana ?? 0)) {
        // Mana gain — blue sparkle
        const mRect = manaBar.getBoundingClientRect();
        FX.particleBurst(mRect.left + mRect.width / 2, mRect.top + mRect.height / 2, {
          count: 8,
          color: { r: 58, g: 120, b: 200 },
          speed: 1,
          size: 1.5,
          life: 40,
          glow: true,
        });
      }
    }

    // --- Gold ---
    const goldEl = document.getElementById('statGold');
    const prevGold = prev ? (prev.gold ?? 0) : 0;
    const newGold = state.gold ?? 0;

    if (prevGold !== newGold) {
      goldEl.textContent = newGold;

      if (newGold > prevGold) {
        FX.flashStat('statGold');
        const goldRect = goldEl.getBoundingClientRect();
        FX.goldSparkles(goldRect.left + goldRect.width / 2, goldRect.top + goldRect.height / 2, 6);
        FX.floatingNumber(
          goldRect.left + goldRect.width / 2,
          goldRect.top,
          '+' + (newGold - prevGold),
          'gold'
        );
      } else {
        FX.flashStat('statGold', 'damage');
      }
    } else {
      goldEl.textContent = newGold;
    }

    // --- Equipped ---
    const eqEl = document.getElementById('statEquipped');
    if (eqEl.textContent !== (state.equipped || '—')) {
      eqEl.textContent = state.equipped || '—';
      FX.flashStat('statEquipped');
      // Weapon equip flash
      FX.runeFlash();
    }

    // --- Inventory ---
    const invList = document.getElementById('statInventory');
    const newInv = (state.inventory || []).map(i => esc(i));
    invList.innerHTML = newInv.map(i => `<li>${i}</li>`).join('');

    if (prev) {
      const prevInv = prev.inventory || [];
      const addedItems = (state.inventory || []).filter(i => !prevInv.includes(i));
      if (addedItems.length > 0) {
        FX.questBanner('🗡 ' + addedItems.join(', '));
      }
    }

    // --- Skills ---
    const skillsList = document.getElementById('statSkills');
    const newSkills = (state.skills || []).map(s => esc(s));
    skillsList.innerHTML = newSkills.map(s => `<li>${s}</li>`).join('');

    if (prev) {
      const prevSkills = prev.skills || [];
      const addedSkills = (state.skills || []).filter(s => !prevSkills.includes(s));
      if (addedSkills.length > 0) {
        FX.levelUpRays();
        FX.questBanner('✦ Навык: ' + addedSkills.join(', '));
      }
    }

    // --- Quests ---
    const questList = document.getElementById('statQuests');
    const newQuests = (state.active_quests || []).map(q => esc(q));
    questList.innerHTML = newQuests.map(q => `<li>${q}</li>`).join('');

    if (prev) {
      // Detect completed quests
      const prevQuests = prev.active_quests || [];
      const completedQuests = prevQuests.filter(q => !(state.active_quests || []).includes(q));
      completedQuests.forEach(q => {
        FX.questBanner('⚔ Квест выполнен!');
        FX.levelUpRays();
      });

      // Detect new quests
      const addedQuests = (state.active_quests || []).filter(q => !prevQuests.includes(q));
      if (addedQuests.length > 0) {
        FX.questBanner('📜 Новый квест');
      }
    }

    // Store state for next diff
    previousState = JSON.parse(JSON.stringify(state));
  }

  // ===============================================================
  // STORY LOG
  // ===============================================================

  function addStoryEntry(text, isResponse, isAction, isError) {
    // Remove placeholder
    const placeholder = storyInner.querySelector('.story-placeholder');
    if (placeholder) placeholder.remove();

    const entry = document.createElement('div');
    entry.className = 'story-entry';

    if (isAction) {
      const actionEl = document.createElement('div');
      actionEl.className = 'story-action';
      actionEl.textContent = text;
      entry.appendChild(actionEl);
    } else if (isError) {
      const msgEl = document.createElement('div');
      msgEl.className = 'story-message story-message--error';
      msgEl.textContent = text;
      entry.appendChild(msgEl);
    } else {
      const msgEl = document.createElement('div');
      msgEl.className = isResponse ? 'story-message story-message--first' : 'story-message story-message--system';

      if (isResponse && text.length > 30) {
        // Use typewriter effect for AI responses
        entry.appendChild(msgEl);
        storyInner.appendChild(entry);
        FX.typewriterText(msgEl, text, 12);
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
    setTimeout(() => {
      storyLog.scrollTop = storyLog.scrollHeight;
    }, 50);
  }

  function enableInput() {
    actionInput.disabled = false;
    actionInput.focus();
  }

  // ===============================================================
  // THINKING INDICATOR
  // ===============================================================

  function updateThinking(show) {
    if (show) {
      thinkingIndicator.classList.add('thinking-indicator--visible');
      actionInput.disabled = true;
    } else {
      thinkingIndicator.classList.remove('thinking-indicator--visible');
    }
  }

  // ===============================================================
  // SESSION MANAGEMENT
  // ===============================================================

  async function newGame() {
    try {
      const resp = await fetch('/api/game/new', { method: 'POST' });
      const data = await resp.json();

      if (data.error) {
        addStoryEntry('⚠ ' + data.error, false, false, true);
        return;
      }

      sessionId = data.session_id;
      previousState = null;
      storyInner.innerHTML = '';

      updateState(data.state, data.turn_count);
      addStoryEntry(data.state.message, true);

      // Grand entrance effect
      FX.particleBurst(window.innerWidth / 2, window.innerHeight / 2, {
        count: 40,
        color: { r: 200, g: 169, b: 110 },
        speed: 3,
        size: 2,
        life: 80,
        glow: true,
      });

      // Reconnect WebSocket
      if (ws) ws.close();
      connectWebSocket();
    } catch (err) {
      addStoryEntry('⚠ Ошибка: ' + err.message, false, false, true);
    }
  }

  async function saveGame() {
    if (!sessionId) return;

    try {
      const resp = await fetch('/api/game/save', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: sessionId }),
      });
      const data = await resp.json();

      if (data.status === 'ok') {
        addStoryEntry('💾 Игра сохранена в свиток.', false, false, false);
        FX.runeFlash();
        FX.questBanner('💾 Сохранено');
      }
    } catch (err) {
      addStoryEntry('⚠ Ошибка сохранения: ' + err.message, false, false, true);
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
        addStoryEntry('⚠ ' + data.error, false, false, true);
        return;
      }

      sessionId = data.session_id;
      previousState = null;
      storyInner.innerHTML = '';

      updateState(data.state, data.turn_count);
      addStoryEntry('📂 Свиток загружен.', false, false, false);
      addStoryEntry(data.state.message, true);

      FX.runeFlash();
      FX.locationTransition();

      if (ws) ws.close();
      connectWebSocket();
      closeSavesModal();
    } catch (err) {
      addStoryEntry('⚠ Ошибка загрузки: ' + err.message, false, false, true);
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
      addStoryEntry('⚠ Ошибка: ' + err.message, false, false, true);
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
    if (isThinking) return;
    sendAction(actionInput.value);
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

  // --- Keyboard shortcut: Escape to close modals ---
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
