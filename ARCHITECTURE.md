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
        system.go     # World, Systems, Engine (CreatureAISystem, CreatureMovementSystem)
        /board        # Plateau, grilles, positions (gère la géométrie)
        /entity       # Identités, Manager, États (TileState, Type)
        /component    # Données ECS (Lifecycle, Matchable...)
        /creature     # Créatures, IA et mouvements avancés
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

**Architecture interne :**

Le domaine utilise une architecture **Entity-Component-System (ECS)** améliorée :

- **Entities** : Chaque entité (créature, ressource, structure, piège) possède :
  - Un identifiant unique (ID)
  - Une position sur la grille
  - Un état (TileState : Hidden, Revealed, Matched, Blocked)
  - Des composants optionnels (Lifecycle, Matchable, CreatureAI, etc.)

- **Board/Grid** : Gère la géométrie du plateau
  - Chaque tuile contient une référence optionnelle à une entité
  - Les tuiles ne portent plus d'état ; c'est l'entité qui le porte
  - Permet la recherche rapide des entités par position

- **Systems** : Mettent à jour l'état du monde
  - **CreatureAISystem** : Gère les comportements de base des créatures
  - **CreatureMovementSystem** : Implémente le système de mouvement avancé (triggers, navigation, modes)
  - **ResourceLifecycleSystem** : Gère la maturation des ressources

- **Types d'entités** :
  - `TypeResource` : Ressources récoltables
  - `TypeCreature` : Créatures avec IA
  - `TypeStructure` : Structures fixes (terriers, etc.)
  - `TypeArtefact` : Objets spéciaux
  - `TypeTrap` : Pièges / tuiles vides

```go
// Exemple: Créer un monde et spawner des entités
world := domain.NewWorld()
world.CreateGrid("forest", 6, 6, domain.BiomeForest)
world.SpawnResource("forest", "dreamberry", entity.Position{X: 1, Y: 1})
world.SpawnCreature("forest", "lumifly", entity.Position{X: 3, Y: 3})
```

### 2. Usecase (Actions)

Encapsule les actions joueur en Commandes. Les commandes manipulent les entités et mettent à jour leur état :

```go
revealCmd := &usecase.RevealTileCommand{
    World:         world,
    GridID:        "forest",
    Position:      board.Position{X: x, Y: y},
    FlipDirection: domain.FlipTop,
}
if revealCmd.CanExecute() {
    entity, err := revealCmd.Execute() // Retourne l'entité révélée
}
```

**Commandes principales :**
- `RevealTileCommand` : Révèle une entité (passe son état de Hidden à Revealed)
- `MatchTilesCommand` : Appaire deux entités (passe leur état à Matched)
- `SwitchGridCommand` : Change de grille active

### 3. Infrastructure

- **Assets**: Cache d'images, génération de placeholders
  - Génération procédurale des tuiles avec thèmes visuels
  - Thèmes disponibles : Default (bleu-violet), Forest (forestier), Cave (obscur), etc.
  - Motifs visuels différents pour chaque état (`Hidden`, `Revealed`, `Matched`)
  - TODO: Remplacer par des assets finaux avant release
  
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

Pour les révélations de tuiles, le bus transporte aussi les informations nécessaires au rendu : `grid_id`, `position`, `entity_id` et `flip_direction`.

```go
eventBus.Publish(event.NewEntityRevealedEvent(position, entityID, gridID, flipDirection))
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

### Jeu de base

| Touche | Action |
|--------|--------|
| Click | Révéler tuile / Sélectionner |
| M | Matcher la sélection |
| Espace | Fin de tour |
| Échap | Désélectionner |

### Navigation

| Touche | Action |
|--------|--------|
| 1-9 | Changer de grille (grid 1-9) |
| R | Réinitialiser la rotation |
| + / - | Rotation horaire / anti-horaire |
| \ | Retour au menu |

### Debug / Test

| Touche | Action |
|--------|--------|
| S | Spawner des entités de test (ressources + lumifly) |
| Shift+S | Spawner toutes les créatures de test |
| C | Nettoyer le plateau |
| F1 | Info debug (FPS, entités) |
| F2 | Afficher les profils de mouvement |
| F3 | Forcer le prochain tour |
| F5 | Révéler toutes les tuiles (cheat debug) |
| F6 | Cacher toutes les tuiles (cheat debug) |
| F9 | Spawn créature aléatoire |
| F10 | Toggle mouvement automatique ON/OFF |

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

### 3. Système de Mouvement des Créatures

Le système de mouvement avancé (`CreatureMovementSystem`) permet de définir finement le comportement de déplacement des créatures via des profils configurables.

#### Structure du MovementProfile

```go
// Dans domain/creature/movement.go
type MovementProfile struct {
    Trigger     MovementTrigger    // Quand se déplacer
    Navigation  NavigationLogic    // Où aller
    Mode        MovementMode       // Comment se déplacer
    Frequency   MovementFrequency  // À quelle fréquence
    Orientation Orientation        // Direction du regard
    Collision   CollisionHandler   // Gestion des obstacles
}
```

#### Types de déclencheurs (Trigger)

| Type | Description |
|------|-------------|
| `TriggerPassive` | Aucun mouvement (ressource fixe) |
| `TriggerAuto` | Se déplace à la fin de chaque tour |
| `TriggerOnReveal` | Se déplace dès qu'elle est révélée |
| `TriggerOnEcho` | Se déplace si une autre tuile est révélée |
| `TriggerProximity` | Se déplace si action dans rayon N cases |

#### Types de navigation

| Type | Description |
|------|-------------|
| `NavWander` | Errance aléatoire |
| `NavPatrol` | Suit un itinéraire prédéfini |
| `NavOrientation` | Selon la direction du regard |
| `NavAttraction` | Vise une cible (joueur, ressource...) |
| `NavRepulsion` | S'éloigne d'une cible |

#### Modes de déplacement

| Mode | Description |
|------|-------------|
| `ModeBento` | Déplacement visible (glissement) |
| `ModeShadow` | Déplacement invisible (face cachée) |
| `ModeSwap` | Échange de position avec la cible |
| `ModeOver` | Au-dessus des tuiles (vole) |
| `ModeUnder` | Sous les tuiles (terrier) |

#### Gestion des collisions

| Type | Description |
|------|-------------|
| `CollideStop` | S'arrête devant l'obstacle |
| `CollideBounce` | Rebondit (change d'orientation 180°) |
| `CollideSlide` | Glisse le long de l'obstacle |
| `CollidePhase` | Traverse certains types de tuiles |

#### Créer une créature avec un profil de mouvement

```go
// Utilisation des profils prédéfinis
specter, _ := world.SpawnCreature("cave", "specter", pos)

// Création d'un patrouilleur personnalisé
route := []entity.Position{
    {X: 1, Y: 1}, {X: 1, Y: 5}, 
    {X: 5, Y: 5}, {X: 5, Y: 1},
}
warden, _ := factory.CreatePatroller("stonewarden", pos, route)

// Profil personnalisé
profile := &creature.MovementProfile{
    Trigger: creature.MovementTrigger{
        Type: creature.TriggerProximity,
        Radius: 3,
    },
    Navigation: creature.NavigationLogic{
        Type: creature.NavRepulsion,
        Target: creature.TargetPlayer,
    },
    Mode: creature.MovementMode{
        Type: creature.ModeShadow,
    },
    Frequency: creature.MovementFrequency{
        Type: creature.FreqVelocity,
        Velocity: 2,
    },
    Collision: creature.CollisionHandler{
        Type: creature.CollideSlide,
    },
}
```

### 4. Nouveau système ECS

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
