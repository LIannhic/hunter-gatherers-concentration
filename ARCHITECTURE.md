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
        game.go       # Ré-export des types (Façade)
        /system       # Logique centrale (World, Engine, ECS Systems)
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
  - Une position sur une grille
  - Un état (TileState : Hidden, Revealed, Matched, Blocked, Cumuled)
  - Des tags dynamiques pour le comportement ou le rendu (ex: `moss_lure`, `dangerous_on_reveal`)
  - Des composants optionnels (Lifecycle, Matchable, CreatureAI, etc.)
  - **Creature** : Possède une `ThreatZone` définissant ses angles d'attaque (Forward, Backward, Left, Right). Supporte **8 directions cardinales et ordinales** transformées par son orientation unifiée (intrinsèque + D4).

- **Board/Grid** : Gère la géométrie du plateau
  - Chaque tuile contient une référence optionnelle à une entité.
  - Les tuiles ne portent plus d'état ; c'est l'entité qui le porte.
  - Plusieurs entités peuvent être empilées sur une même parcelle (`EntitiesID []string`).
  - Les actions de masquage (Skip, Fin de tour, F6) s'appliquent à **toute la pile** d'une parcelle.
  - Permet la recherche rapide des entités par position.
  - **Inventory Grid** : L'inventaire est désormais une grille logicielle (`InventoryGridID`), permettant un traitement spatial uniforme (hover, highlights).
  - **Tilt (Pente)** : Chaque parcelle possède une direction de pente utilisée pour définir l'animation de fermeture "naturelle" des tuiles. Les transformations sont cumulatives (`apply * current`).
  - **Cumul (Merge)** : Les entités peuvent être fusionnées pour augmenter leur `CumulationLevel` (0 à 2). Cela influence les règles de match et le rendu (échelle x1.15 par niveau si révélé).

- **Systems** : Mettent à jour l'état du monde via l'**Engine** (`engine.go`).
  - **Engine** : Chef d'orchestre du domaine. Il sépare la logique en deux cycles :
    - `Update()` : Cycle par tour (IA, maturation, fin de tour).
    - `UpdateFrame(dt)` : Cycle temps réel à 60 FPS (Timers, évènements UI, prévisualisation).
  - **CreatureAISystem** : Gère les comportements de base des créatures
  - **CreatureMovementSystem** : Implémente le système de mouvement avancé (triggers, navigation, modes)
  - **ResourcePropagationSystem** : Gère la multiplication des ressources sur les cases adjacentes. Émet l'événement `ResourcePropagated` enrichi des positions `from` et `to` pour l'UI.
  - **ResourceLifecycleSystem** : Gère la maturation des ressources. Émet des logs détaillés sur les transitions de stade.
  - **ToxicitySystem** : Calcule les dégâts de poison cumulés et dégressifs infligés au joueur par les ressources révélées (ex: Dreamberry stade 4).
  - **LootSystem** : Transforme les matches réussis en entités `TypeLoot` et les place sur la grille d'inventaire. Le butin hérite du niveau de cumul de la paire.
  - **TrackSystem** : Gère la décomposition temporelle des traces

- **Animations Organiques** :
  - **Propagation (Division Cellulaire)** : Lorsqu'une ressource se multiplie, une animation de type `propagate` est déclenchée. Elle simule une division organique en deux phases :
    1. **Phase d'extension** (progress 0.0 à 0.5) : Les angles de "tête" de la nouvelle tuile s'élancent vers la destination tandis que la "traîne" reste ancrée.
    2. **Phase de stabilisation** (progress 0.5 à 1.0) : La tête se fixe et la traîne rejoint la position cible pour restaurer la forme carrée.
    - **Filament élastique** : Un quadrilatère blanc semi-transparent relie les deux tuiles, s'affinant sur l'axe de déplacement durant la phase 2 avant de disparaître (rupture de tension).
    - **Brouillard de Guerre** : L'animation s'effectue exclusivement sur le **DOS** de la tuile (`tile_hidden`) pour respecter le secret du Memory en fin de tour.


- **Interface `Hoverable`** : Unifie l'interaction de survol pour les tuiles, les sorties de navigation et les objets de l'inventaire. Centralise la logique d'autorisation (ex: blocage des tuiles scellées).

- **TurnTimer** (`timer.go`) : Compte à rebours temps réel par tour
  - Décrémente à chaque frame (60 fps)
  - Déclenche un auto-skip à l'expiration
  - Phase de panique (< 3s) utilisée pour les feedbacks visuels (pulse Sanity Gauge)
  - Durée maximale synchronisée avec `meta.DifficultySettings.TurnTimerDuration`

- **Suivi de Progression et Score** :
  - `TotalExperience` : Cumul de toute l'expérience acquise durant une session (Matchs + Butin final). Utilisé comme base pour le calcul du Score dans la persistance.
  - `Experience` : XP relative au niveau actuel, utilisée pour le système de Level-up.

- **Buffs et Protection** :
  - `ImmunityTurns` : Géré dans `Player`, permet de bloquer tous les dégâts. Utilisé par l'effet du Shadowstalker.

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
- `MatchTilesCommand` : Tente d'appairer deux entités identiques de même niveau de cumul. Applique la **Matrice de Dégâts** (pénalité si match invalide ou skip de match valide). Pour le moment, seul l'appairage par similarité est actif.
- `MergeTilesCommand` : Fusionne deux entités identiques normales en une seule version cumulée.
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

#### Déploiement du Portail et Effet Séisme

Lors du déploiement (`DeployPortablePortalAt`), la méthode `clear3x3DeploymentArea` est appelée systématiquement. Elle retire toutes les entités des 8 parcelles adjacentes pour garantir une zone de sécurité visuelle et logique.

#### Feedback Graphique (Vortex)

Un shader spécial `vortex.kage` est déclenché par l'application (`app.go`) uniquement lorsque le portail est actif (`IsVictoryTimerActive`). Le shader utilise les coordonnées réelles du portail (via `GetTileCenter`) pour ancrer la distorsion.

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
- **Atlas des Assets** : Fenêtre modale (T) paginée pour le debug visuel. Utilise un système de boutons pour la navigation et la fermeture.

Séparation des responsabilités :
- **Renderer**: Dessine le plateau central avec espacement dynamique.
  - **Gestion de la Profondeur** : Les zones de messages sont rendues en premier pour être couvertes par les fenêtres modales (Z-indexing logique).
  - **Système de Calques (Depth Illusion)** : Utilise trois strates conceptuelles (**Under**, **Normal**, **Over**).
  - **Calcul Dynamique** : Utilise `getTileCenter` pour aligner parfaitement les strates et les traces dans les interstices, supportant la rotation globale et les variations d'espacement (3x3 à 6x6).
  - **Espaces de Coordonnées** :
    - **Plateau (Board)** : 525x525. Contient les tuiles et les traces.
    - **Tapis de Jeu (Playmat)** : 700x700. Contient le plateau, les boutons et les **Effets Plein Écran** (ex: Scanner de l'Echo Hound).
- **Input**: Capture les événements (clavier, souris, tactile), gère la navigation entre les zones et les raccourcis clavier. Supporte les interactions mobiles (Wasm) via le défilement de l'inventaire par glissement (Drag-to-scroll) et l'appui long pour la suppression.
- **HUD**: Orchestre l'affichage des informations fixes et des fenêtres volantes (ex: Statistiques des zones).
  - **Système de Messages Défilants**: Gère deux zones de notification indépendantes (**Gauche** et **Droite**) avec des files d'attente prioritaires. Chaque message défile de droite à gauche deux fois avant de disparaître.
    - **Zone Gauche**: Affiche les messages narratifs et les effets d'utilisation d'objets.
    - **Zone Droite**: Affiche les feedbacks de gameplay immédiats (Confrontations, erreurs de match).
- **EffectRenderer** (`renderer/effect_renderer.go`) : Gère les shaders globaux (Wave, Heat, Bubble, Blur) avec un système de ping-pong buffers. L'intensité des effets est couplée dynamiquement à la **Santé Mentale** du joueur. Peut être forcé via la **Console de Debug**.

- **DebugWindow** (`ui/debug/window.go`) : Console de débogage interactive (F12) permettant de modifier les statistiques, la difficulté, et de filtrer les entités spawnables.

- **ActionButtons** (`ui/actionbuttons/manager.go`) : Manager purement réactif qui recalcule à chaque frame l'état des 4 boutons d'action (Match, Skip, Turn, Menu) en fonction du nombre de tuiles retournées et des troubles cognitifs actifs du joueur. Applique des transformations de coordonnées (scrambling) et gère le remplissage temporel du bouton Skip. Coordonne le **Feedback de Coût** vers les jauges du HUD.

### 5. App (Wiring)

Connecte tout ensemble :
```go
app.NewApplication() // Crée world, assets, renderer, input...
```

La couche `App` gère également le flux de navigation pré-jeu, notamment la **sélection de difficulté** qui est déclenchée lors du clic sur "DEMARRER" (pour les nouveaux joueurs) ou via le menu de profil.

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

### Jeu de base (Actions directes)

| Action | Touche |
|--------|--------|
| Révéler tuile | Click gauche (Plateau) |
| Matcher (valider paire) | M ou Bouton MATCH |
| Skip (si 2 tuiles révélées) | Espace ou Bouton SKIP |
| Fin de tour forcée | Espace (sans match) ou Bouton TURN |
| Naviguer entre les zones | ZQSD / WASD / Flèches |
| Rotation plateau (Visuel) | + (Horaire) / - (Anti-horaire) |
| Reset rotation | R |

### Gestion et Debug

| Action | Touche |
|--------|--------|
| Inventaire (Usage/Détails) | Click gauche / L |
| Statistiques zones | I |
| Menu / Abandon | Échap ou \ |
| Changer de grille | 1-9 |
| Difficulté | F1 à F4 |
| Console de Debug | F12 |
| Révéler tout (Cheat) | F5 |
| Cacher tout (Cheat) | F6 |
| Spawn entités (Debug) | S / Shift+S / F9 |
| Nettoyer plateau (Cheat) | C |

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
| `NavRelative` | Par rapport à son orientation absolue (saute si bloqué) |
| `NavOrientation` | Selon la direction du regard |
| `NavAttraction` | Vise une cible spécifique (Ressource par nom, Joueur, etc.) |
| `NavRepulsion` | S'éloigne de la cible |

#### Ciblage par stade (Lifecycle Filtering)

Les créatures peuvent désormais filtrer leurs cibles selon leur stade de développement via la propriété `ExcludedStages` du `NavigationLogic`. 
Exemple: Les Lumiflies ignorent les Dreamberries au stade 1 et 4.

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

## Compiler votre projet en WebAssembly

```bash
# 1. On renomme pour "cacher" le fichier Windows à Go
Rename-Item ./cmd/game/rsrc.syso rsrc.syso.bak

# 2. On compile pour le WebAssembly
$env:GOOS="js"; $env:GOARCH="wasm"; go build -o hgcv0.2_basic-incursion.wasm ./cmd/game

# 3. On remet le nom d'origine pour que votre version Windows fonctionne à nouveau
Rename-Item ./cmd/game/rsrc.syso.bak rsrc.syso
```

