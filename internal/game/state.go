package game

// DefaultPlayerName is the default player name for new saves.
const DefaultPlayerName = "Player"

// GameState describes the current state of the player and the world.
// This is the object we serialize to JSON for the AI and parse back.
type GameState struct {
	Condition       string   `json:"condition"`
	Sanity          string   `json:"sanity"`           // Sanity level
	Inventory       []string `json:"inventory"`        // List of items
	Equipped        string   `json:"equipped"`         // What is currently held
	Gold            int      `json:"gold"`             // Currency
	Location        string   `json:"location"`         // Name of the current room/area
	CurrentQuest    string   `json:"current_quest"`    // Current global goal
	ActiveQuests    []string `json:"active_quests"`    // Active quests
	CompletedQuests []string `json:"completed_quests"` // Completed quests
	QuestLog        []string `json:"quest_log"`        // Journal entries
	Skills          []string `json:"skills"`           // Skills/spells
	Message         string   `json:"message"`          // AI response for the player
	Choices         []string `json:"choices"`          // AI-generated choices for decision points
}

// NewPlayer creates the starting state for a new player
func NewPlayer() *GameState {
	return &GameState{
		Condition:      "Healthy",
		Sanity:         "Stable",
		Inventory:      []string{"Water flask", "Flint"},
		Equipped:       "Fists",
		Gold:           0,
		Location:       "Ruined Monastery Gates",
		CurrentQuest:   "Find shelter before nightfall",
		ActiveQuests:   []string{"Find shelter before nightfall"},
		CompletedQuests: []string{},
		QuestLog:       []string{},
		Skills:         []string{},
		Message:        "You stand before the ancient gates. A cold mist chills you to the bone.",
	}
}
