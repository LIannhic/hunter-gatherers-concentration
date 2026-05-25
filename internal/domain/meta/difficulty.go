package meta

type DifficultyLevel string

const (
	LevelEasy     DifficultyLevel = "Easy"     // Révèle tout au début
	LevelNormal   DifficultyLevel = "Normal"   // Révèle une tuile sur deux
	LevelHard     DifficultyLevel = "Hard"     // Rien n'est révélé
	LevelInsane   DifficultyLevel = "Insane"   // Révèle, mais se recache très vite
	LevelPlaytest DifficultyLevel = "Playtest" // Mode de test dense
)

type DifficultySettings struct {
	Level             DifficultyLevel
	PreviewDuration   float64 // Temps avant que les tuiles ne se recachent
	PreviewRatio      float64 // Pourcentage de tuiles à montrer (1.0 = 100%)
	NavThreshold      float64 // Pourcentage de paires à trouver pour ouvrir les sorties
	TurnTimerDuration float64 // Durée max du compte à rebours par tour (en secondes)
}

func GetSettings(level DifficultyLevel) DifficultySettings {
	switch level {
	case LevelEasy:
		return DifficultySettings{Level: LevelEasy, PreviewDuration: 2.0, PreviewRatio: 1.0, NavThreshold: 0.25, TurnTimerDuration: 15.0}
	case LevelNormal:
		return DifficultySettings{Level: LevelNormal, PreviewDuration: 1.5, PreviewRatio: 0.5, NavThreshold: 0.50, TurnTimerDuration: 10.0}
	case LevelHard:
		return DifficultySettings{Level: LevelHard, PreviewDuration: 1.0, PreviewRatio: 0.8, NavThreshold: 0.75, TurnTimerDuration: 5.0}
	case LevelInsane:
		return DifficultySettings{Level: LevelInsane, PreviewDuration: 0.5, PreviewRatio: 1.0, NavThreshold: 1.00, TurnTimerDuration: 5.0}
	default:
		return GetSettings(LevelNormal)
	}
}
