# Architecture du projet

Ce document décrit l'architecture complète de "Hunter-Gatherers Concentration".

```
cmd/game              # Point d'entrée
    main.go           # Lance la boucle Ebiten

/internal
    /app              # Orchestration haut niveau
        app.go        # Wiring des dépendances, callbacks
        
    /game             # Implémentation de l'interface ebiten.Game
        game.go       # Adapte l'app pour Ebiten
        
    /domain           # Cœur métier (pur, testable)
        README.md     # Documentation des patterns
        game.go       # Ré-export des types
        system.go     # World, Systems, Engine
        /board        # Plateau, tuiles, positions
        /entity       # Identités, Manager
        /component    # Données ECS (Lifecycle, Matchable...)
        /creature     # Créatures et IA
        /resource     # Ressources récoltables
        /event        # Bus d'événements
        /player       # Stats, inventaire
        /meta         # Progression entre missions
        /association  # Système de Memory (Strategy pattern)
        
    /usecase          # Actions applicatives
        commands.go   # Command pattern (RevealTile, MatchTiles...)
        
    /infrastructure   # Détails techniques
        /assets       # Gestion des sprites/couleurs
        /loader       # Chargement JSON/config
        
    /ui               # Interface utilisateur
        /renderer     # Dessin du plateau
        /input        # Gestion souris/clavier
        /hud          # Affichage infos
```

## Flux de données

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Input     │────▶│  Usecase    │────▶│   Domain    │
│ (Souris/    │     │  (Command)  │     │  (World)    │
│  Clavier)   │     │             │     │             │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                                               ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Ebiten    │◀────│    Game     │◀────│  Renderer   │
│   (Affiche) │     │   (Loop)    │     │   (HUD)     │
└─────────────┘     └─────────────┘     └─────────────┘
```

## Couches

### 1. Domain (Cœur métier)

**Aucune dépendance externe** (sauf uuid). Contient :
- La logique pure du jeu
- Les règles métier (associations, maturation...)
- Les tests unitaires

```go
// Exemple: Créer un monde et spawner des entités
world := domain.NewWorld(6, 6)
world.SpawnResource("dreamberry", domain.Position{X: 1, Y: 1})
world.SpawnCreature("lumifly", domain.Position{X: 3, Y: 3})
```

### 2. Usecase (Actions)

Encapsule les actions joueur en Commandes :

```go
revealCmd := &usecase.RevealTileCommand{
    World:    world,
    Position: board.Position{X: x, Y: y},
}
if revealCmd.CanExecute() {
    revealCmd.Execute()
}
```

### 3. Infrastructure

- **Assets**: Cache d'images, génération de placeholders
- **Loader**: Configuration depuis JSON (avec fallback par défaut)

### 4. UI

Séparation des responsabilités :
- **Renderer**: Dessine uniquement, pas de logique
- **Input**: Capture les événements, délègue aux usecases
- **HUD**: Affiche les informations (découplé du renderer)

### 5. App (Wiring)

Connecte tout ensemble :
```go
app.NewApplication() // Crée world, assets, renderer, input...
```

## Patterns utilisés

### Clean Architecture
- Domain au centre, sans dépendances externes
- UI et Infrastructure dépendent du Domain
- Flux de contrôle: UI → Usecase → Domain

### Command Pattern
Chaque action est une commande avec `CanExecute()` et `Execute()` :
- Facile à tester
- Peut être mise en file d'attente
- Annulation possible (à implémenter)

### ECS (Entity-Component-System)
- **Entity**: Identité + Position
- **Component**: Données (Lifecycle, Matchable...)
- **System**: Logique (LifecycleSystem, CreatureAISystem)

### Observer Pattern
Event Bus pour la communication :
```go
eventBus.Subscribe(CreatureMoved, handler)
eventBus.Publish(NewCreatureMovedEvent(...))
```

## Lancer le jeu

```bash
# Développement
go run ./cmd/game

# Build
go build -o game ./cmd/game
./game

# Tests
go test ./internal/domain/... -v
```

## Contrôles

| Touche | Action |
|--------|--------|
| Click | Révéler tuile / Sélectionner |
| M | Matcher la sélection |
| S | Spawner des entités de test |
| C | Nettoyer le plateau |
| Espace | Fin de tour |
| Échap | Désélectionner |

## Ajouter une fonctionnalité

### 1. Nouveau type de ressource

Dans `domain/resource/resource.go` :
```go
case "crystal_shard":
    r.SetLifecycle(component.Lifecycle{...})
    r.SetValue(component.Value{...})
```

Dans `infrastructure/assets/manager.go` :
```go
// Ajouter une icône
crystalImg := ebiten.NewImage(size, size)
// ... dessiner
c.images["resource_crystal_shard"] = crystalImg
```

### 2. Nouvelle action

Dans `usecase/commands.go` :
```go
type HarvestResourceCommand struct {
    World    *domain.World
    Position board.Position
}

func (c *HarvestResourceCommand) Execute() error {
    // Logique de récolte
}
```

Dans `ui/input/handler.go` :
```go
if inpututil.IsKeyJustPressed(ebiten.KeyH) {
    cmd := &usecase.HarvestResourceCommand{...}
    cmd.Execute()
}
```

### 3. Nouveau système ECS

Dans `domain/system.go` :
```go
type WeatherSystem struct{}

func (s *WeatherSystem) Priority() int { return 5 }
func (s *WeatherSystem) Update(world *World) {
    // Modifier les ressources selon la météo
}
```

Dans `domain/game.go` (si besoin de ré-exporter).

## Tests

```bash
# Tous les tests
go test ./...

# Seul le domain (rapide)
go test ./internal/domain/... -v

# Avec couverture
go test ./internal/domain/... -cover
```
