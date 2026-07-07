/**
 * Тёмный Монастырь — Game State Module
 * ======================================
 * Central state management: session, turn counter, thinking flag.
 * Exposes window.GameState for cross-module access.
 */
(function () {
  'use strict';

  const state = {
    sessionId: null,
    isThinking: false,
    previousState: null, // for diffing and triggering effects
    currentTurnCount: -1,
  };

  window.GameState = {
    get sessionId() { return state.sessionId; },
    set sessionId(id) { state.sessionId = id; },

    get isThinking() { return state.isThinking; },
    set isThinking(v) { state.isThinking = v; },

    get previousState() { return state.previousState; },
    set previousState(s) { state.previousState = s; },

    get currentTurnCount() { return state.currentTurnCount; },
    set currentTurnCount(n) { state.currentTurnCount = n; },
  };

})();
