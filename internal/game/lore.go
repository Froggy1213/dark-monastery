package game

import (
	"fmt"
	"strings"
)

// LoreBook contains the static lore of the game world — factions, locations, NPCs, legends.
// This data is injected into the system prompt so Gemini knows the game world.
type LoreBook struct {
	WorldName        string
	WorldDescription string
	Factions         []Faction
	Locations        map[string]Location
	MajorNPCs        []NPC
	Legends          []string
	RandomEvents     []string
}

// Faction — a faction/organization in the game world.
type Faction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Goal        string `json:"goal"`
	Symbol      string `json:"symbol"`
}

// Location — a key location in the world.
type Location struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Connections []string `json:"connections"`
	DangerLevel string   `json:"danger_level"` // "Low", "Medium", "High", "Deadly"
	Secrets     []string `json:"secrets"`
}

// NPC — a key character in the world.
type NPC struct {
	Name        string `json:"name"`
	Role        string `json:"role"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Faction     string `json:"faction"`
	Secret      string `json:"secret"`
}

// DefaultLore returns a pre-populated lore for the Dark Monastery.
func DefaultLore() *LoreBook {
	return &LoreBook{
		WorldName:        "Dark Monastery",
		WorldDescription: "An ancient monastery lost in the cursed mountains. Once, monks of the Order of the Fading Light prayed here, but now it is a haven of darkness, mad cultists, and beings from other worlds. The monastery dungeons descend deep into the earth, hiding ancient secrets and unimaginable horrors.",
		Factions: []Faction{
			{
				Name:        "Order of the Fading Light",
				Description: "The remnants of monks trying to contain the darkness of the monastery. Few in number, but they know ancient purification rituals.",
				Goal:        "Seal the Gates of the Abyss beneath the monastery.",
				Symbol:      "A dim candle in a circle",
			},
			{
				Name:        "Cult of the Crimson Dawn",
				Description: "Mad sectarians worshipping the ancient evil sleeping beneath the monastery. They make blood sacrifices and hunt lost travelers.",
				Goal:        "Awaken the Sleeper in the Abyss and gain eternal life.",
				Symbol:      "A red sun with black rays",
			},
			{
				Name:        "Mist Wanderers",
				Description: "A neutral faction of explorers and relic hunters. They know the monastery's secret passages and trade in rare artifacts.",
				Goal:        "Find the legendary artifact 'Heart of the Monastery'.",
				Symbol:      "A gray mask with empty eye sockets",
			},
			{
				Name:        "Guardians of the Threshold",
				Description: "Ghostly knights bound by an oath to guard the entrance to the dungeons even after death. They do not attack first, but are merciless to those who break their prohibitions.",
				Goal:        "Prevent anyone from reaching the Deep Gates.",
				Symbol:      "A black shield with a silver skull",
			},
		},
		Locations: map[string]Location{
			"Ruined Monastery Gates": {
				Name:        "Ruined Monastery Gates",
				Description: "Massive stone gates covered in moss and cracks. Above them is a half-erased fresco: a saint slaying a serpent. The air is heavy and cold.",
				Connections: []string{"Courtyard of Penance", "Watchtower"},
				DangerLevel: "Low",
				Secrets:     []string{"An ancient amulet of the Order of Light is hidden in a niche above the gates."},
			},
			"Courtyard of Penance": {
				Name:        "Courtyard of Penance",
				Description: "A spacious inner courtyard paved with cracked slabs. In the center is a dried-up fountain shaped like a weeping angel. Traces of bonfires — the Mist Wanderers often stop here.",
				Connections: []string{"Ruined Monastery Gates", "Main Hall", "Novice Cells"},
				DangerLevel: "Medium",
				Secrets:     []string{"A hidden cache of the Wanderers is concealed in the fountain.", "One of the slabs leads to an underground passage."},
			},
			"Main Hall": {
				Name:        "Main Hall",
				Description: "A huge hall with a collapsed vault. The remnants of an altar are visible, stained with something dark. On the walls are frescoes depicting the monastery's history, but many faces are erased or replaced with eerie symbols.",
				Connections: []string{"Courtyard of Penance", "Library", "Crypt"},
				DangerLevel: "High",
				Secrets:     []string{"The altar conceals a mechanism that opens a passage to the Crypt.", "The frescoes change at night, pointing the way to a secret room."},
			},
			"Library": {
				Name:        "Library",
				Description: "Endless rows of shelves with ancient tomes. Many books crumble to dust at the touch. The air smells of old paper and sulfur.",
				Connections: []string{"Main Hall", "Astronomer's Tower"},
				DangerLevel: "High",
				Secrets:     []string{"The forbidden section is hidden behind an illusory wall.", "The last abbot's diary describes the ritual for sealing the Gates."},
			},
			"Crypt": {
				Name:        "Crypt",
				Description: "A damp dungeon with rows of stone sarcophagi. Some lids are shifted. A barely audible whisper comes from the depths.",
				Connections: []string{"Main Hall", "Deep Gates"},
				DangerLevel: "Deadly",
				Secrets:     []string{"The body of a Saint rests in one of the sarcophagi — untouched by decay.", "The whisper is the voices of imprisoned demons."},
			},
			"Deep Gates": {
				Name:        "Deep Gates",
				Description: "A huge stone door covered in glowing runes. Beyond it lies the Abyss, the source of all the monastery's darkness. The air vibrates with tension.",
				Connections: []string{"Crypt"},
				DangerLevel: "Deadly",
				Secrets:     []string{"The Gates can only be opened with the Heart-Key.", "Beyond the Gates sleeps an ancient being — the Sleeper in the Abyss."},
			},
		},
		MajorNPCs: []NPC{
			{
				Name:        "Abbot Morgan",
				Role:        "Last prior of the monastery (ghost)",
				Description: "A tall, translucent figure in a tattered robe. Speaks slowly, with pain. Cursed to wander the ruins forever.",
				Location:    "Main Hall",
				Faction:     "Order of the Fading Light",
				Secret:      "Knows where the Heart-Key is hidden, but will only give it to a worthy person.",
			},
			{
				Name:        "Priestess Morrigan",
				Role:        "Leader of the Cult of the Crimson Dawn",
				Description: "A young woman with mad eyes in a crimson robe. Speaks in a sing-song voice, often laughs without reason.",
				Location:    "Crypt",
				Faction:     "Cult of the Crimson Dawn",
				Secret:      "Was once a novice of the Order of Light. Her fall is the result of the Abyss's influence.",
			},
			{
				Name:        "Gareth the Wanderer",
				Role:        "Merchant and informant",
				Description: "A thin man in a gray hooded cloak. Always appears unexpectedly. Knows all the rumors and secrets of the monastery.",
				Location:    "Courtyard of Penance",
				Faction:     "Mist Wanderers",
				Secret:      "In truth, a former inquisitor who fled the church.",
			},
		},
		Legends: []string{
			"They say that during the full moon in the Main Hall you can hear the choir of ghostly monks.",
			"The Sleeper in the Abyss dreams — and these dreams become reality within the monastery walls.",
			"Whoever lights the three Ritual Candles in the Crypt can speak with the dead.",
			"The Heart-Key is not an object, but a living creature that has taken the form of a stone.",
			"Everyone who tried to leave the monastery after sunset returned to the gates an hour later, not remembering the way.",
		},
		RandomEvents: []string{
			"A sudden gust of icy wind extinguishes all lights.",
			"A child's crying comes from the darkness, abruptly cut off.",
			"A bloody inscription appears on the wall that wasn't there a moment ago.",
			"You hear footsteps behind you, but turning around, you see only emptiness.",
			"A fresco on the wall briefly comes to life — the depicted saint turns his head toward you.",
			"The floor trembles beneath your feet, and a low rumble emanates from the depths.",
		},
	}
}

// LorePrompt creates a text description of the lore for insertion into the system prompt.
func (l *LoreBook) LorePrompt() string {
	var sb strings.Builder

	sb.WriteString("WORLD:\n")
	sb.WriteString(l.WorldDescription)
	sb.WriteString("\n\n")

	sb.WriteString("FACTIONS:\n")
	for _, f := range l.Factions {
		fmt.Fprintf(&sb, "- %s: %s Goal: %s\n", f.Name, f.Description, f.Goal)
	}

	sb.WriteString("\nKEY LOCATIONS:\n")
	for _, loc := range l.Locations {
		fmt.Fprintf(&sb, "- %s (danger: %s): %s\n", loc.Name, loc.DangerLevel, loc.Description)
		if len(loc.Connections) > 0 {
			sb.WriteString("  Connected to: ")
			sb.WriteString(strings.Join(loc.Connections, ", "))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\nKEY CHARACTERS (NPCs):\n")
	for _, npc := range l.MajorNPCs {
		fmt.Fprintf(&sb, "- %s (%s): %s\n", npc.Name, npc.Role, npc.Description)
	}

	sb.WriteString("\nLEGENDS AND RUMORS:\n")
	for _, legend := range l.Legends {
		fmt.Fprintf(&sb, "- %s\n", legend)
	}

	return sb.String()
}
