package game

// DefaultPlayerName — имя игрока по умолчанию для новых сохранений.
const DefaultPlayerName = "Игрок"

// GameState описывает текущее состояние игрока и мира.
// Именно этот объект мы сериализуем в JSON для ИИ и парсим обратно.
type GameState struct {
	Condition      string   `json:"condition"`
	Sanity         string   `json:"sanity"`          // Уровень рассудка
	Inventory      []string `json:"inventory"`       // Список предметов
	Equipped       string   `json:"equipped"`        // Что сейчас в руках
	Gold           int      `json:"gold"`            // Валюта
	Location       string   `json:"location"`        // Название текущей комнаты/зоны
	CurrentQuest   string   `json:"current_quest"`   // Текущая глобальная цель
	ActiveQuests   []string `json:"active_quests"`   // Активные квесты
	CompletedQuests []string `json:"completed_quests"` // Завершённые квесты
	QuestLog       []string `json:"quest_log"`       // Записи в журнале
	Skills         []string `json:"skills"`          // Навыки/заклинания
	Message        string   `json:"message"`         // Ответ ИИ для игрока
}

// NewPlayer создает начальное состояние для нового игрока
func NewPlayer() *GameState {
	return &GameState{
		Condition:      "Здоров",
		Sanity:         "Стабильный",
		Inventory:      []string{"Фляга с водой", "Огниво"},
		Equipped:       "Кулаки",
		Gold:           0,
		Location:       "Разрушенные ворота монастыря",
		CurrentQuest:   "Найти укрытие до наступления ночи",
		ActiveQuests:   []string{"Найти укрытие до наступления ночи"},
		CompletedQuests: []string{},
		QuestLog:       []string{},
		Skills:         []string{},
		Message:        "Вы стоите перед древними вратами. Холодный туман пробирает до костей.",
	}
}
