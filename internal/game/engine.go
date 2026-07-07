package game

// AIClient — интерфейс для AI-движка игры.
// Позволяет подменять реализацию (Gemini, мок для тестов, будущие провайдеры).
type AIClient interface {
	GenerateNextTurn(state *GameState, action string) (*GameState, error)
}

// Engine управляет игровым циклом: принимает состояние и действие игрока,
// делегирует AI-клиенту генерацию следующего хода.
type Engine struct {
	aiClient AIClient
}

// NewEngine создаёт новый экземпляр игрового движка.
func NewEngine(aiClient AIClient) *Engine {
	return &Engine{aiClient: aiClient}
}

// ProcessTurn обрабатывает один ход: отправляет состояние и действие в AI,
// возвращает обновлённое состояние.
func (e *Engine) ProcessTurn(state *GameState, action string) (*GameState, error) {
	return e.aiClient.GenerateNextTurn(state, action)
}

// AIClient возвращает текущий AI-клиент.
// Используется для настройки контекста памяти перед вызовом ProcessTurn.
func (e *Engine) AIClient() AIClient {
	return e.aiClient
}
