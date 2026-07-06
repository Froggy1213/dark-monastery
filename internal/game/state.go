package game

// GameState описывает текущее состояние игрока и мира.
// Именно этот объект мы будем сериализовать в JSON для ИИ и парсить обратно.
type GameState struct {
	Condition    string   `json:"condition"`
	Sanity       string   `json:"sanity"`        // Уровень рассудка (от 0 до 100)
	Inventory    []string `json:"inventory"`     // Список предметов ("Ржавый меч", "Факел")
	Equipped     string   `json:"equipped"`      // Что сейчас в руках
	Gold         int      `json:"gold"`          // Валюта (монеты, осколки душ и т.д.)
	Location     string   `json:"location"`      // Название текущей комнаты/зоны
	CurrentQuest string   `json:"current_quest"` // Текущая глобальная цель
	Message      string   `json:"message"`       // Сгенерированный текст (ответ ИИ для игрока)
}

// NewPlayer создает начальное состояние для нового игрока
func NewPlayer() *GameState {
	return &GameState{
		Condition:    "Здоров",
		Sanity:       "Стабильный",
		Inventory:    []string{"Фляга с водой", "Огниво"},
		Equipped:     "Кулаки",
		Gold:         0,
		Location:     "Разрушенные ворота монастыря",
		CurrentQuest: "Найти укрытие до наступления ночи",
		Message:      "Вы стоите перед древними вратами. Холодный туман пробирает до костей.",
	}
}
