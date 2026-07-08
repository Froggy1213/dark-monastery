package game

// AIClient is an interface for the AI game engine.
// Allows swapping implementations (Gemini, mock for tests, future providers).
type AIClient interface {
	GenerateNextTurn(state *GameState, action string) (*GameState, error)
}

// Engine manages the game loop: receives state and player action,
// delegates to the AI client for generating the next turn.
type Engine struct {
	aiClient AIClient
}

// NewEngine creates a new game engine instance.
func NewEngine(aiClient AIClient) *Engine {
	return &Engine{aiClient: aiClient}
}

// ProcessTurn processes one turn: sends the state and action to the AI,
// returns the updated state.
func (e *Engine) ProcessTurn(state *GameState, action string) (*GameState, error) {
	return e.aiClient.GenerateNextTurn(state, action)
}

// AIClient returns the current AI client.
// Used to configure memory context before calling ProcessTurn.
func (e *Engine) AIClient() AIClient {
	return e.aiClient
}
