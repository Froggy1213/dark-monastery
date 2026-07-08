/**
 * Тёмный Монастырь — UI Module
 * ==============================
 * All DOM rendering: state updates, story log, thinking indicator.
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
  // STATE UPDATE — diff-based stat rendering
  // ===============================================================

  function updateState(state, turnCount) {
    const prev = GS.previousState;

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
      const lower = sanity.toLowerCase();
      sanityIcon.className = 'sanity-dot';
      if (lower.includes('безум') || lower.includes('хаос') || lower.includes('грани')) {
        sanityIcon.classList.add('sanity-dot--broken');
      } else if (lower.includes('нестабил') || lower.includes('тревог') || lower.includes('паран')) {
        sanityIcon.classList.add('sanity-dot--frayed');
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
        FX.floatingNumber(goldRect.left + goldRect.width / 2, goldRect.top, '+' + (newGold - prevGold), 'gold');
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
      const prevQuests = prev.active_quests || [];
      const completedQuests = prevQuests.filter(q => !(state.active_quests || []).includes(q));
      completedQuests.forEach(() => {
        FX.questBanner('⚔ Квест выполнен!');
        FX.levelUpRays();
      });

      const addedQuests = (state.active_quests || []).filter(q => !prevQuests.includes(q));
      if (addedQuests.length > 0) {
        FX.questBanner('📜 Новый квест');
      }
    }

    // Store for next diff
    GS.previousState = JSON.parse(JSON.stringify(state));
  }

  // ===============================================================
  // STORY LOG
  // ===============================================================

  function addStoryEntry(text, isResponse, isAction, isError) {
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
  // EXPOSE PUBLIC API
  // ===============================================================

  window.GameUI = {
    updateState: updateState,
    addStoryEntry: addStoryEntry,
    updateThinking: updateThinking,
    scrollToBottom: scrollToBottom,
    enableInput: enableInput,
  };

})();
