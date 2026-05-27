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
        timer.go      # Compte à rebours temps réel par tour (TurnTimer)
        /board        # Plateau, grilles, positions (gère la géométrie)
        /entity       # Identités, Manager, États (TileState, Type)
        /component    # Données ECS (Lifecycle, Matchable...)
        /creature     # Créatures, IA et mouvements avancés
        /resource     # Ressources récoltables
        /event        # Bus d'événements
        /player       # Stats, inventaire, altérations d'état (StatusEffects)
        /meta         # Progression entre missions, paramètres de difficulté
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
        /actionbuttons # Gestionnaire réactif des 4 boutons d'action du Playmat
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
  - Des tags dynamiques pour le comportement ou le rendu (ex: `moss_lure`, `dangerous_on_reveal`)
  - Des composants optionnels (Lifecycle, Matchable, CreatureAI, etc.)
  - **Creature** : Possède une `ThreatZone` définissant ses angles d'attaque (face, côtés, etc.).

- **Board/Grid** : Gère la géométrie du plateau
  - Chaque tuile contient une référence optionnelle à une entité
  - Les tuiles ne portent plus d'état ; c'est l'entité qui le porte
  - Permet la recherche rapide des entités par position
  - **Tilt (Pente)** : Chaque parcelle possède une direction de pente utilisée pour définir l'animation de fermeture "naturelle" des tuiles.

- **Systems** : Mettent à jour l'état du monde
  - **CreatureAISystem** : Gère les comportements de base des créatures
  - **CreatureMovementSystem** : Implémente le système de mouvement avancé (triggers, navigation, modes)
  - **ResourceLifecycleSystem** : Gère la maturation des ressources
  - **LootSystem** : Transforme les matches réussis en objets d'inventaire
  - **TrackSystem** : Gère la décomposition temporelle des traces

- **TurnTimer** (`timer.go`) : Compte à rebours temps réel par tour
  - Décrémente à chaque frame (60 fps)
  - Déclenche un auto-skip à l'expiration
  - Phase de panique (< 3s) utilisée pour les feedbacks visuels (pulse Sanity Gauge)
  - Durée maximale synchronisée avec `meta.DifficultySettings.TurnTimerDuration`

- **StatusEffects** (`player/status.go`) : Altérations mentales du joueur
  - `Aphasia`, `Agnosia`, `Ataxia`, `Amnesia`
  - Interceptées par le rendu UI pour scrambler les coordonnées/labels des boutons d'action

- **Types d'entités** :
  - `TypeResource` : Ressources récoltables
  - `TypeCreature` : Créatures avec IA. Supporte **8 directions cardinales et ordinales** pour la détection de menace précise.
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
- `RevealTileCommand` : Révèle une entité, met à jour la position périphérique du joueur et vérifie la **Confrontation** (dégâts si dans la `ThreatZone`).
- `MatchTilesCommand` : Tente d'appairer deux entités et applique la **Matrice de Dégâts** (pénalité si match invalide ou skip de match valide).
- `SwitchGridCommand` : Change de grille active.
- `UsePortablePortalCommand` : Active un portail portable pour créer une zone de dégagement et extraire le joueur du plan

### 3. Infrastructure

- **Assets**: Cache d'images, génération de placeholders
  - Génération procédurale des tuiles avec thèmes visuels
  - Thèmes disponibles : Default (bleu-violet), Forest (forestier), Cave (obscur), etc.
  - Motifs visuels différents pour chaque état (`Hidden`, `Revealed`, `Matched`)
  - TODO: Remplacer par des assets finaux avant release
  
- **Loader**: Configuration depuis JSON (avec fallback par défaut)

### Système de Portail Portable

Le portail portable permet au joueur d'extraire rapidement du plan actif. Ce système gère :

#### Recherche de Zone de Dégagement (3x3)

```go
// Cherche une zone 3x3 libre n'importe où sur la grille
pos, ok := world.FindAvailable3x3DeploymentArea(gridID)

// Trouve la meilleure zone 3x3 (minimise les obstructions)
pos, ok := world.findBest3x3DeploymentArea(gridID)
```

#### Déploiement du Portail

```go
// Déploie à la meilleure position trouvée automatiquement
portal, err := world.DeployPortablePortal(gridID)

// Déploie à une position spécifique (centre de la zone 3x3)
portal, err := world.DeployPortablePortalAt(gridID, center)
```

#### Activation via Commande

```go
cmd := &usecase.UsePortablePortalCommand{
    World:  world,
    GridID: "forest",
    Center: board.Position{X: 3, Y: 3}, // Position du centre, ou négatif pour auto
}
if cmd.CanExecute() {
    err := cmd.Execute()
}
```

Le système valide que :
- Une zone 3x3 est disponible (aucune tuile obstruée ou occupée)
- Le joueur possède un portail portable en inventaire
- La grille cible existe

### 4. UI

Le jeu utilise une résolution logique fixe de **1280x720**. L'interface est divisée en plusieurs zones gérées par le `HUD` :
- **Portrait** (270x270) : Affiche les statistiques du personnage, les contrôles et le contenu dynamique de la zone.
- **Inventaire** (270x420) : Grille 3x4 pour les objets récoltés.
- **Playmat** (700x700) : Zone centrale contenant le plateau de jeu (525x525), les boutons d'action et les indicateurs de sortie.
- **Gauges** (270x420) : Barres verticales de Santé, Mana et Santé Mentale.
- **Minimap** (270x270) : Carte interactive du Plan de Rêve.

Séparation des responsabilités :
- **Renderer**: Dessine le plateau central avec espacement dynamique (remplit toujours l'espace de 525x525 quelle que soit la taille de la grille). 
  - **Système de Calques (Depth Illusion)** : Utilise trois strates conceptuelles (**Under**, **Normal**, **Over**) pour gérer l'ordre d'affichage des traces, des tuiles et des entités en mouvement.
  - **Calcul Dynamique** : Utilise `getTileCenter` pour aligner parfaitement les strates et les traces dans les interstices, supportant la rotation globale et les variations d'espacement (3x3 à 6x6).
  - **Espaces de Coordonnées** :
    - **Plateau (Board)** : 525x525. Contient les tuiles et les traces.
    - **Tapis de Jeu (Playmat)** : 700x700. Contient le plateau, les boutons et les **Effets Plein Écran** (ex: Scanner de l'Echo Hound).
- **Input**: Capture les événements, gère la navigation entre les zones et les raccourcis clavier.
- **HUD**: Orchestre l'affichage des informations fixes et des fenêtres volantes (ex: Statistiques des zones).
- **ActionButtons** (`ui/actionbuttons/manager.go`) : Manager purement réactif qui recalcule à chaque frame l'état des 4 boutons d'action (Match, Skip, Turn, Menu) en fonction du nombre de tuiles retournées et des troubles cognitifs actifs du joueur. Applique des transformations de coordonnées (scrambling) et gère le remplissage temporel du bouton Skip.

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

| Action | Touche |
|--------|--------|
| Révéler tuile | Click souris |
| Matcher (valider paire) | M |
| Skip (quand 2 tuiles retournées) | Espace |
| Naviguer zones | ZQSD / Flèches |
| Statistiques zones | I |
| Fin de tour | Espace (hors match en cours) |
| Menu / Abandon | Échap |
| Changer de grille | 1-9 |
| Difficulté | F1 à F4 |
| Révéler tout (Cheat) | F5 |
| Cacher tout (Cheat) | F6 |
| Rotation plateau | + / - |
| Reset rotation | R |
| Spawn entités (debug) | S |
| Spawn toutes créatures (debug) | Shift+S |
| Nettoyer plateau (debug) | C |
| Retour menu | \ |

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

#### Bestiaire (Exemples)

| Créature | Déclencheur | Navigation | Perception | Mode |
|----------|-------------|------------|------------|------|
| **Lumifly** | Auto | Errance | Manifest | Over |
| **Shadowstalker** | Proximité | Attraction | Cloaked | Normal |
| **Echo Hound** | Echo | Attraction | Manifest | Normal |
| **Burrower** | Vue | Errance | Manifest | Under |
| **Specter** | Echo | Errance | Cloaked | Under |

#### Types de navigation

| Type | Description |
|------|-------------|
| `NavWander` | Errance directionnelle |
| `NavPatrol` | Suit un itinéraire prédéfini |
| `NavRelative` | Par rapport à sa position et son orientation |
| `NavOrientation` | Selon la direction du regard |
| `NavAttraction` | Vise une cible spécifique |
| `NavRepulsion` | S'éloigne de la cible |

#### Modes de déplacement

| Mode         | Description |
|--------------|-------------|
| `ModeNormal` | Déplacement standard au sol |
| `ModeSwap`   | Interversion physique de deux tuiles |
| `ModeOver`   | Passe au-dessus des autres tuiles (Tag "flying") |
| `ModeUnder`  | Passe en dessous des autres tuiles (Tag "burrowed") |

#### NOUVEAU : Règles de Perception

Régit comment le monde/joueur perçoit les actions d'une entité via son `PerceptionProfile`.

| Paramètre | Options | Description |
|-----------|---------|-------------|
| **Stealth** | `Manifest`, `Cloaked` | Visibilité de la translation (Bento vs Shadow) |
| **Acoustic** | `Silent`, `Echo` | Émission d'un stimulus sonore lors du mouvement |
| **Traces** | `LeavesTracks` | Si `true`, génère des entités `TypeTrack` |

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
