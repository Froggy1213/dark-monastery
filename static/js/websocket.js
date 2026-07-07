/**
 * Тёмный Монастырь — WebSocket Module
 * =====================================
 * WebSocket connection, reconnection, message handling.
 * Depends on: window.GameState, window.DarkFX, window.GameUI
 */
(function () {
  'use strict';

  const GS = window.GameState;
  const FX = window.DarkFX;
  const UI = window.GameUI;

  let ws = null;
  let wsReconnectTimer = null;

  function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const sid = GS.sessionId ? '?session_id=' + GS.sessionId : '';
    const url = `${protocol}//${window.location.host}/ws/game${sid}`;

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
        GS.isThinking = false;
        UI.updateThinking(false);
        if (data.turn_count !== GS.currentTurnCount) {
          UI.updateState(data.state, data.turn_count);
          UI.addStoryEntry(data.state.message, true);
          GS.currentTurnCount = data.turn_count;
        }
        break;

      case 'thinking':
        GS.isThinking = true;
        UI.updateThinking(true);
        break;

      case 'error':
        GS.isThinking = false;
        UI.updateThinking(false);
        UI.addStoryEntry('⚠ ' + data.message, false, false, true);
        break;

      case 'pong':
        break;

      default:
        console.log('[WS] Unknown type:', data.type);
    }
  }

  function sendAction(text) {
    if (!text.trim()) return;

    FX.sendActionEffect();
    UI.addStoryEntry(text, false, true);

    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({
        type: 'action',
        text: text,
        session_id: GS.sessionId,
      }));
    } else {
      // Fallback HTTP
      fetch('/api/game/action', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: GS.sessionId, action: text }),
      })
        .then(r => r.json())
        .then(data => {
          GS.isThinking = false;
          UI.updateThinking(false);
          if (data.error) {
            UI.addStoryEntry('⚠ ' + data.error, false, false, true);
          } else {
            UI.updateState(data.state, data.turn_count);
            UI.addStoryEntry(data.state.message, true);
            GS.currentTurnCount = data.turn_count;
          }
        })
        .catch(err => {
          GS.isThinking = false;
          UI.updateThinking(false);
          UI.addStoryEntry('⚠ Ошибка соединения: ' + err.message, false, false, true);
        });
    }

    GS.isThinking = true;
    UI.updateThinking(true);

    const actionInput = document.getElementById('actionInput');
    if (actionInput) {
      actionInput.value = '';
      actionInput.disabled = true;
    }
  }

  // Expose public API
  window.GameWS = {
    connect: connectWebSocket,
    sendAction: sendAction,
  };

})();
