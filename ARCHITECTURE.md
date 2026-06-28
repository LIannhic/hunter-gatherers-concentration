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

- **Manager** : Centralise la gestion des entités.
  - **Mise en cache** : Utilise un cache (`cacheByType` + `dirtyTypes`) pour `GetByType` afin d'éviter les tris et allocations inutiles à chaque appel. Le cache est invalidé uniquement lors d'un `Register` ou `Remove`.

- **Board/Grid** : Gère la géométrie du plateau
  - Chaque tuile contient une référence optionnelle à une entité.
  - Les tuiles ne portent plus d'état ; c'est l'entité qui le porte.
  - Plusieurs entités peuvent être empilées sur une même parcelle (`EntitiesID []string`).
  - Les actions de masquage (Skip, Fin de tour, F6) s'appliquent à **toute la pile** d'une parcelle.
  - Permet la recherche rapide des entités par position.
  - **Inventory Grid** : L'inventaire est désormais une grille logicielle (`InventoryGridID`), permettant un traitement spatial uniforme (hover, highlights).
  - **Tilt (Pente)** : Chaque parcelle possède une direction de pente utilisée pour définir l'animation de fermeture "naturelle" des tuiles. Les transformations sont cumulatives (`apply * current`).
  - **Transformation D4 Unifiée** : Les UVs des tuiles (face et dos) sont transformés par la D4 de l'entité, y compris pour les créatures et ressources. Les croix et silhouettes suivent la rotation/miroir. Le dos utilise un miroir horizontal supplémentaire (`1.0 - u`).
  - **Cumul (Merge)** : Les entités peuvent être fusionnées pour augmenter leur `CumulationLevel` (0 à 2+). Cela influence les règles de match et le rendu (échelle x1.15 par niveau + bordures concentriques colorées si révélé).

- **Systems** : Mettent à jour l'état du monde via l'**Engine** (`engine.go`).
  - **Engine** : Chef d'orchestre du domaine. Il sépare la logique en deux cycles :
    - `Update()` : Cycle par tour (IA, maturation, fin de tour).
    - `UpdateFrame(dt)` : Cycle temps réel à 60 FPS (Timers, évènements UI, prévisualisation).
  - **CreatureAISystem** : Gère les comportements de base des créatures
  - **CreatureMovementSystem** : Implémente le système de mouvement avancé (triggers, navigation, modes). **Filtre par grille** : stocke les révélations dans `RevealedTile{Position, GridID}`. `TriggerOnReveal`, `TriggerOnEcho`, `TriggerProximity` ne réagissent qu'aux événements sur la grille de la créature. **Wandering fallback** : les créatures avec `NavAttraction`/`NavRepulsion` errent aléatoirement quand leur trigger ne se déclenche pas ET que leur cible n'existe pas sur la grille.
  - **ResourcePropagationSystem** : Gère la multiplication des ressources sur les cases adjacentes. Émet l'événement `ResourcePropagated` enrichi des positions `from` et `to` pour l'UI.
  - **ResourceLifecycleSystem** : Gère la maturation des ressources. Supporte les **cycles cycliques** (`Cyclic: true`) : les ressources reviennent au stade 0 après le stade maximum. La propagation a lieu avant le reset du cycle.
  - **ToxicitySystem** : Calcule les dégâts de poison cumulés et dégressifs infligés au joueur par les ressources révélées (ex: Dreamberry stade 4). **Ne vérifie que la grille actuelle** (`world.CurrentGridID`).
  - **LootSystem** : Transforme les matches réussis en entités `TypeLoot` et les place sur la grille d'inventaire. Le butin hérite du niveau de cumul de la paire.
  - **TrackSystem** : Gère la décomposition temporelle des traces (décrément de `Duration` à chaque tour, suppression définitive à 0).
  - **AggressionSystem** : Calcule l'agressivité modulaire des créatures (Priority 1, avant mouvement). S'abonne à `TileRevealed` (reason: "player_action") pour incrémenter `RevealCount` et mettre à jour le facteur "reveals". S'abonne à `CreatureMoved` pour gérer la "patience" (+2% par pas, +10% par rebond). Déclenche l'attaque si agressivité totale ≥ 100 à la fin de l'animation de flip.

- **CreatureAttackEffectSystem** : Centralise les conséquences logiques mondiales des attaques réussies. S'abonne à `CreatureAttacked` et applique des changements permanents ou majeurs au monde (ex: rotation de grille du Stonewarden, déclenchement des troubles cognitifs Aphasia, Ataxia, Agnosia, Amnesia, Vertigo — uniquement si `hit_target` présent dans le payload, i.e. attaque qui touche le joueur).

- **ComboSystem** (Priority 10) : Gère le système de combo (associations consécutives). S'abonne à `TileMatched` pour incrémenter le compteur et calculer la juiciness (1-5). S'abonne à `TileMerged` pour publier un message sans incrémenter. S'abonne à `PlayerDamaged` pour réinitialiser le combo en cas d'erreur (`invalid_match`, `skipped_valid_match`). Publie `ComboTriggered` via `PublishImmediate` (sinon perdu dans `ProcessQueue`). Le message est rendu par le HUD dans la `ComboZone` (270×40px, en haut à droite) avec un fond coloré par niveau, un outline noir 8-directions, et un slide-in depuis la droite.

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
  - **Remaining** : Utilisé par `TriggerLumiflyEffect` pour calculer la durée de persistance des silhouettes dorées (temps restant du tour, pas la durée totale).
  - **Time Scaling** : Vitesse adaptée selon contexte dans `Engine.UpdateFrame` :
    - Grille vide → `Stop()` (0%)
    - Preview/Animation → 50% vitesse (`dt * 0.5`)
    - Normal → 100%

- **World.ActiveAnimationCount** : Compteur d'animations actives (flip, slide, attack). Incrémenté par `AnimationStarted`, décrémenté par `AnimationEnded` (handlers dans `BoardRenderer.SubscribeToEvents`).

- **Suivi de Progression et Score** :
  - `TotalExperience` : Cumul de toute l'expérience acquise durant une session (Matchs + Butin final). Utilisé comme base pour le calcul du Score dans la persistance.
  - `Experience` : XP relative au niveau actuel, utilisée pour le système de Level-up.

- **Buffs et Protection** :
  - `ImmunityTurns` : Géré dans `Player`, permet de bloquer tous les dégâts. Utilisé par l'effet du Shadowstalker.

- **StatusEffects** (`player/status.go`) : Altérations mentales du joueur
  - `Aphasia`, `Agnosia`, `Ataxia`, `Amnesia`, `Vertigo`
  - Interceptées par le rendu UI pour scrambler les coordonnées/labels des boutons d'action

- **Types d'entités** :
  - `TypeResource` : Ressources récoltables
  - `TypeCreature` : Créatures avec IA. Supporte **8 directions cardinales et ordinales** pour la détection de menace précise.
  - `TypeStructure` : Structures fixes (terriers, etc.)
  - `TypeArtefact` : Objets spéciaux
  - `TypeTrap` : Pièges / tuiles vides. Tirés aléatoirement dans le pool global de population (plus de paires fixes).

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
- `RevealTileCommand` : Révèle une entité, met à jour la position périphérique du joueur.
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

### Paramètres de Difficulté Étendus

La structure `DifficultySettings` (dans `domain/meta/difficulty.go`) inclut désormais :

```go
type DifficultySettings struct {
	PreviewDuration    float64 // Durée d'affichage de la prévisualisation (s)
	PreviewRatio       float64 // % de tuiles montrées (1.0 = 100%)
	NavThreshold       float64 // % de paires pour ouvrir les sorties
	TurnTimerDuration  float64 // Durée max du timer par tour (s)
	MaxSafeReveals     int     // Nb révélations sûres avant agressivité 100%
	AggressionMult     float64 // Multiplicateur global d'agressivité (futur)
}
```

| Niveau   | Timer | Preview | NavThreshold | MaxSafeReveals | AggressionMult |
|----------|-------|---------|--------------|----------------|----------------|
| Easy     | 15.0  | 1.3s    | 0.5          | 3              | 0.5            |
| Normal   | 10.0  | 0.8s    | 0.6          | 2              | 1.0            |
| Hard     | 5.0   | 0.3s    | 0.7          | 1              | 1.5            |
| Insane   | 5.0   | 0.1s    | 0.8          | 0              | 2.0            |

**Logique MaxSafeReveals** : Chaque révélation joueur ajoute `100 / (MaxSafeReveals + 1)` % d'agressivité. À 0 (Insane), 1 clic = 100% = attaque immédiate.

---

### Time Scaling (Gestion Dynamique du Temps)

Le `TurnTimer` adapte sa vitesse selon le contexte dans `Engine.UpdateFrame(dt)` :

| Condition | Time Scale | Comportement |
|-----------|------------|--------------|
| **Grille vide** (aucune Resource/Creature) | 0.0 | Timer **arrêté** (`Stop()`) |
| **Prévisualisation active** (`PreviewSystem.IsPreviewActive`) | 0.5 | Timer **ralenti 50%** |
| **Animation active** (`World.ActiveAnimationCount > 0`) | 0.5 | Timer **ralenti 50%** |
| **Normal** | 1.0 | Vitesse normale |

**Priorité** : Grille vide > Animation/Preview > Normal.

**Implémentation** :
- `World.ActiveAnimationCount` : Incrémenté par `AnimationStarted`, décrémenté par `AnimationEnded` (handlers dans `BoardRenderer.SubscribeToEvents`).
- `PreviewSystem.IsPreviewActive(gridID)` : Vérifie `previewTimers[gridID] > 0`.
- Détection grille vide : Aucune entité `TypeResource` ou `TypeCreature` sur la grille actuelle.

---

### Correction : Toxicité Locale

Le `ToxicitySystem` (Priority 6) ne vérifie maintenant que la **grille actuelle** (`world.CurrentGridID`) au lieu de `world.GridOrder`. Les Dreamberries stade 4 sur d'autres grilles ne causent plus de dégâts de poison.

---

### Correction : Déclencheurs de Mouvement par Grille

Le `CreatureMovementSystem` filtre maintenant les déclencheurs par grille :

- **Nouveau struct** `RevealedTile{Position, GridID}` stocke la grille de chaque révélation.
- `TriggerOnReveal` : Ne se déclenche que si révélation sur **même grille** + même position.
- `TriggerOnEcho` : Ne réagit qu'aux révélations sur **même grille**.
- `TriggerProximity` : Ne calcule la distance que pour les révélations sur **même grille**.

Cela corrige le bug où les Stone Wardens (et autres) bougeaient dans toutes les grilles quand le joueur révélait une tuile aux mêmes coordonnées ailleurs.

---

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

Lors du déploiement (`DeployPortablePortalAt`), la méthode `clear3x3DeploymentArea` est appelée systématiquement. Elle retire **toutes** les entités de la zone 3x3 (9 cases centrées sur le portail) pour garantir une zone de sécurité visuelle et logique.

#### Pénalités (Rêve Brisé + Taxe Butin)

Si la zone 3x3 contient **des entités** (ressources, créatures, structures) :
- **5 dégâts** au joueur (`applyDreamBreachPenalty` - Rêve Brisé)
- **Taxe Butin 50%** : suppression aléatoire de la moitié de l'inventaire (`applyPortablePortalLootTax`)

La détection utilise `hasEntitiesIn3x3Area()` qui compte **toutes** les entités (pas seulement les structures). Le mode `forced` est activé automatiquement.

#### Feedback Graphique (Vortex)

Un shader spécial `vortex.kage` est déclenché par l'application (`app.go`) lorsque le portail est actif (`IsVictoryTimerActive`). Le shader utilise les coordonnées réelles du portail (via `GetTileCenter`) pour ancrer la distorsion. Fonctionne sur **toutes les grilles** (pas seulement zones de départ/arrivée).

#### Aperçu au Curseur (Prévisualisation)

En mode portail portable, un cadre 3x3 suit le curseur :
- **Vert** (zone 3x3 vide) : Déploiement sans pénalité
- **Jaune** (entités dans la zone 3x3) : Avertissement pénalité + destruction entités + taxe butin

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
- Le centre est valide (`isValid3x3DeploymentCenter` : au moins 1 case des bords)
- Le joueur possède un portail portable en inventaire
- La grille cible existe

#### Nettoyage État (Redémarrage)

`ResetGameState()` désactive `portablePortalMode` pour éviter les fuites d'état entre parties.

### Correspondance Inter-Zones (Cross-Zone Matching)

Le jeu supporte maintenant l'appariement de tuiles situées sur **grilles différentes** (ex: zone de départ + zone intermédiaire).

#### Architecture

- **Handler** : `revealedEntities []string` stocke les **entity IDs** (stables à travers les rotations de grille) au lieu de positions. `revealedGridIDs []string` parallèle stocke la grille d'origine par tuile révélée.
- **Commande** : `MatchTilesCommand` a un champ `GridID2` (optionnel, défaut = `GridID`).
- **Exécution** : `processMatchAttempt()` résout les positions depuis les entity IDs via `h.world.Entities.Get()` au moment de l'usage.
- **Événements** : `TileRevealed` et `TileMatched` incluent `grid_id` pour le rendu.

#### Skip Inter-Zones

`processSkip()` utilise aussi les `revealedGridIDs` pour vérifier les paires valides manquées sur grilles différentes et appliquer les pénalités (dégâts créatures / mana ressources).

### Traces de Pas (Footsteps)

Système de traces visuelles laissées par les créatures lors de leurs déplacements.

#### Types de Traces

| Type | Calque | Position | Description |
|------|--------|----------|-------------|
| **Boue (Mud)** | Under | Interstice entre cases | Orientée selon direction mouvement |
| **Herbe Brisée** | Under | Case d'origine | Marque le départ |
| **Griffures** | Over | Case destination | Impact (calque supérieur) |
| **Empreintes** | Normal | Sous tuiles | Simule passage au sol |
| **Intent Beam** | Over | Entre créature et cible | Rouge = attaque, Blanc = menace |

#### Gestion

- **Entité `Track`** : Champs `OffsetX`, `OffsetY`, `Angle` pour positionnement précis aux bords des cases.
- **FIFO max 2** : `footstepTrackIDs []string` dans Handler — l'ancienne est supprimée quand on dépasse 2.
- **Nettoyage tour** : `ClearFootsteps()` appelé dans `OnTurnEnd` callback.
- **Indicateur de position** : `DrawPlayerPosition()` — affiche la position réelle du joueur (grid pos + ancrage) comme indicateur persistant au bord du plateau.

### 4. UI

Le jeu utilise une résolution logique fixe de **1280x720**. L'interface est divisée en plusieurs zones gérées par le `HUD` :
- **`ui/layout.go`** : Constantes de disposition et types UI partagés (ex: `TileActionState` pour les cadres d'action).
- **Portrait** (270x270) : Affiche les statistiques du personnage, les contrôles et le contenu dynamique de la zone.
- **Inventaire** (270x420) : Grille 3x4 pour les objets récoltés.
- **Playmat** (700x700) : Zone centrale contenant le plateau de jeu (525x525), les boutons d'action et les indicateurs de sortie.
- **Gauges** (270x420) : Barres verticales de Santé, Mana et Santé Mentale.
- **Minimap** (270x270) : Carte interactive du Plan de Rêve.
- **Atlas des Assets** : Fenêtre modale (T) paginée pour le debug visuel. Utilise un système de boutons pour la navigation et la fermeture.
- **Plein Écran** : Le jeu supporte le mode plein écran natif (F11 ou bouton dédié dans le HUD).

Séparation des responsabilités :
- **Renderer**: Dessine le plateau central avec espacement dynamique.
  - **Optimisation des Traces** : `PrepareFrame` pré-filtre une seule fois par frame les traces par couche (Under, Normal, Over) et par grille. Évite le scan complet des entités lors des appels de rendu stratifiés.
  - **Gestion de la Profondeur** : Les zones de messages sont rendues en premier pour être couvertes par les fenêtres modales (Z-indexing logique).
  - **Système de Calques (Depth Illusion)** : Utilise trois strates conceptuelles (**Under**, **Normal**, **Over**).
  - **Calcul Dynamique** : Utilise `getTileCenter` pour aligner parfaitement les strates et les traces dans les interstices, supportant la rotation globale et les variations d'espacement (3x3 à 6x6).
  - **Espaces de Coordonnées** :
    - **Plateau (Board)** : 525x525. Contient les tuiles et les traces.
    - **Tapis de Jeu (Playmat)** : 700x700. Contient le plateau, les boutons et les **Effets Plein Écran** (ex: Scanner de l'Echo Hound).
  - **Barre d'Agressivité** : Sur les créatures révélées (`Aggression > 0`), dessine une barre horizontale (40x4px) en bas de la tuile. Couleur dégradée : Orange (faible) → Rouge (100%). Fond semi-transparent noir.
  - **Cadres d'Action** : Rendu des cadres colorés (`StrokeRect` 3px) via `RenderTileActionFrame()`. Couleurs : Vert (interactive), Rouge (impossible), Orange (indisponible). Cadre permanent sur la 1ère tuile révélée, au survol pour les autres.
- **Input**: Capture les événements (clavier, souris, tactile), gère la navigation entre les zones et les raccourcis clavier. 
  - **Unification Cross-Platform** : Utilise `GetInteractionPosition()` (priorise le tactile sur le curseur) et `IsJustPressed()` / `IsJustReleased()` pour garantir une réactivité identique entre Desktop et WASM/Mobile.
  - **Interactions Tactiles** : Supporte le défilement de l'inventaire par glissement (Drag-to-scroll) et l'appui long pour la suppression.
  - **Cadres d'Action** : Système hybride de feedback visuel sur les tuiles. `computeTileActionState()` évalue l'état d'interaction (vert/rouge/orange) en croisant `isProcessing`, `TileState`, `ImmunityTurns`, animations de mouvement et mana. Rendu via `RenderTileActionFrame()` dans le renderer (cadre `StrokeRect` 3px). Cadre permanent sur la 1ère tuile révélée, au survol pour les autres.
- **HUD**: Orchestre l'affichage des informations fixes et des fenêtres volantes (ex: Statistiques des zones).
  - **Système de Messages Défilants**: Gère deux zones de notification indépendantes (**Gauche** et **Droite**) avec des files d'attente prioritaires. Chaque message défile de droite à gauche deux fois avant de disparaître.
    - **Zone Gauche**: Affiche les messages narratifs et les effets d'utilisation d'objets (ex: "Vous êtes déboussolé.", "Vous toussez du sang.", "La mémoire revient...").
    - **Zone Droite**: Affiche les feedbacks de gameplay immédiats (ex: "CONFRONTATION ! -10 HP", "TOXICITÉ ! -X HP", "AMNÉSIE ! X tours.", "MATCH INVALIDE !").
- **EffectRenderer** (`renderer/effect_renderer.go`) : Gère les shaders globaux via un système de ping-pong buffers. L'intensité des effets est couplée dynamiquement à la **Santé Mentale** du joueur. Peut être forcé via la **Console de Debug**.
  - **Séparation Attack/Biome** : Les shaders sont splités en deux méthodes pour un rendu correct :
    - `ProcessCreatureAttackEffects()` : Blur, Bubble, Vertige — appliqués **AVANT** le HUD pour ne pas affecter les fenêtres UI.
    - `ProcessBiomeEffects()` : Wave, Heat, Rain, Cave, Vortex — appliqués **APRÈS** le HUD (comportement original).
  - **Quake Shader** : Rendu **AVANT** tous les shaders (comme Scanner/Lumifly).
  - **Lumifly Shader** (`renderer/shader/lumifly.kage`) : Onde lumineuse dorée circulaire avec silhouettes d'entités. Centre calculé via `calculateTileScreenPos` pour un alignement parfait avec les tuiles. Rayon basé sur la diagonale d'une case (`√2`).
  - **Silhouettes sur le Dos** : Chaque tuile affiche la silhouette de son entité sur le dos (alpha variable), utilisée par le shader Lumifly pour les effets de révélation. Les UVs du dos sont transformés par la D4 avec miroir horizontal.

- **Quake Shader** (`renderer/shader/quake.kage`) : Effet visuel de séisme déclenché par le Stonewarden. Utilise un frame buffer 990×990 avec SubImage pour cropper en 700×700. Le ghost snapshot (ancienne orientation) est roté dans le shader et le résultat est nettoyement affiché sur le playmat.
  - **Snapshot** (`playmatSnapshot`) : Capture de 990×990 (playmat 700×700 + `QuakePadding` = 145px par côté) pour éviter les espaces vides lors de la rotation.
  - **Frame Buffer** (`quakeFrameBuffer`) : 990×990, reçoit la sortie du shader, puis `SubImage` centre 700×700 affiché à `(PlaymatX, PlaymatY)`.
  - **Uniforms** : `RotationAngle` (Pi/2 ou -Pi/2), `Clockwise` (bool), `GhostSize` [990,990], `Resolution` [990,990].
  - **Rendu** : `RenderQuakeOverlay` appelé **AVANT** les shaders globaux (comme Scanner/Lumifly), puis shaders attack, puis HUD, puis shaders biome.

- **Cave Shader** (`renderer/shader/cave.kage`) : Effet d'ambiance oppressive pour le biome grotte. Couplé à la **Santé Mentale** du joueur via le paramètre `Intensity` (0.0 = sane, 1.0 = folie totale).
  - **Obscurité de fond** : Assombrit uniformément l'écran (55% à sanity满 → 98% à sanity 0). La torche centrale crée un cercle de lumière proportionnel à la folie.
  - **Torche centre** : Rayon de 175px (sanity满) → 13px (sanity 0). Force de 35% (sanity满) → 95% (sanity 0). Vacillement procédural (`hash12`) avec amplitude croissante.
  - **Lights HUD** : Cercles de lumière fixes sur les panneaux HUD (Portrait, Inventaire, Jauges, Minimap). Le rayon est contraint aux dimensions du panneau (`smoothstep(r*edge, r, dist)`). L'ombre interne rétrécit avec l'intensité (0.85 → 0.15), étendant la zone éclairée du centre vers les bords.
  - **Uniforms** : `Time`, `Intensity`, `Resolution`, `HudLights [16]float` (4 × `[cx, cy, radius, _]`).
  - **Application** : Appliqué dans `ProcessGlobalEffects` via `applyCaveShader()`, après les biomes (wave/heat/rain) et avant les effets créatures.

- **DebugWindow** (`ui/debug/window.go`) : Console de débogage interactive (F12) permettant de modifier les statistiques, la difficulté, et de filtrer les entités spawnables. Contrôle la vitesse de défilement des messages HUD (`MessageSpeed` via `+`/`-`).

- **TextUtil** (`ui/textutil/textutil.go`) : Package helper centralisant le rendu texte via `text/v2` avec police Go Mono intégrée (via `//go:embed`). Fournit `Draw()`, `MeasureWidth()` et `GoTextFace`. `MeasureWidth` optimisé avec `len(str) * charWidth` (police monospace). Le `Draw()` soustrait `HAscent` pour maintenir la compatibilité baseline avec text v1.
- **ActionButtons** (`ui/actionbuttons/manager.go`) : Manager purement réactif qui recalcule à chaque frame l'état des 4 boutons d'action (Match, Skip, Turn, Menu) en fonction du nombre de tuiles retournées et des troubles cognitifs actifs du joueur. 
  - **Aphasia** : Glitch périodique des labels (toutes les 0.5s) et pulsation de taille.
  - **Ataxia** : Mode **Whack-a-mole** dynamique. Les boutons sautent hors de l'écran avant de réapparaître dans un autre coin. Thème Désert appliqué.
  - **Agnosia** : Standardisation totale en Noir & Blanc. Labels forcés sur "BOUTON".
  - **Remplissage Effaceur** : Le timer se remplit de Noir vers Blanc, effaçant visuellement le contenu blanc du bouton à mesure qu'il progresse.
  - **Interaction** : Gère les transformations de coordonnées (scrambling) et l'interpolation pour des mouvements fluides. Coordonne le **Feedback de Coût** vers les jauges du HUD.

### 5. App (Wiring)

Connecte tout ensemble :
```go
app.NewApplication() // Crée world, assets, renderer, input...
```

La couche `App` gère également le choix du dépôt de persistance en fonction de l'environnement (WASM vs Desktop).

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

**Piège `ProcessQueue`** : `ProcessQueue()` itère sur un snapshot du slice. Les événements publiés via `Publish()` pendant le traitement d'un handler sont perdus. **Tout handler qui publie un événement chaîné doit utiliser `PublishImmediate()`**.

**Nouveau type d'événement** : `CreatureAttacked` — Publié par `AggressionSystem` quand l'agressivité ≥ 100 à la fin du flip. Payload : `hit_target` (*entity.Position, nil si joueur hors zone). Utilisé par le renderer pour déclencher l'animation de lunge. `AmnesiaStarted` (payload: `turns int`) et `AmnesiaEnded` — gèrent les messages HUD et l'inventaire. `ComboTriggered` — Publié par `ComboSystem` via `PublishImmediate` (sinon perdu dans `ProcessQueue`). Payload : `text`, `count`, `score`, `juiciness`.

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

| Action | Touche / Geste |
|--------|--------|
| Révéler tuile | Relâchement Clic gauche / Doigt (Plateau) |
| Sélectionner tuile révélée | Relâchement Clic gauche / Doigt (Tuile révélée) |
| Désélectionner / Annuler | Clic droit (Plateau) / Échap / Toggle (Mobile) |
| Matcher (valider paire) | M ou Bouton MATCH |
| Skip (si 2 tuiles révélées) | Espace ou Bouton SKIP |
| Fin de tour forcée | Espace (sans match en cours) ou Bouton TURN |
| Naviguer entre les zones | Flèches ou ZQSD / WASD / Clic Sortie |
| Rotation plateau (Visuel) | + (Horaire) / - (Anti-horaire) |
| Reset rotation | R |
| Basculer Plein Écran | F11 ou Bouton F/W (Portrait) |

### Gestion et Inventaire

| Action | Touche / Geste |
|--------|--------|
| Sélection Butin (Usage) | Relâchement Clic gauche / Tap (Inventaire) |
| Utiliser Butin sélectionné | Re-Relâchement / Re-Tap (Inventaire) |
| Sélection Suppression | Clic droit / Appui Long (0.5s) (Inventaire) |
| Désélectionner Butin | Clic droit / Tap hors inventaire |
| Défilement Inventaire | Molette / Glissement (Drag) vertical |
| Portail Portatif (Raccourci) | P |
| Statistiques des zones | I |
| Détails Inventaire | L |
| Atlas des Assets (Toggle) | T |
| Pagination Atlas | Boutons [PRECEDENT] / [SUIVANT] |
| Fermer Atlas | Bouton [X] ou T |

### Paramètres et Debug

| Action | Touche |
|--------|--------|
| Difficulté (E, N, H, I) | F1 à F4 |
| Augmenter Combo (Cheat) | F10 |
| Fenêtre de Debug (Console) | F12 |
| Spawn entités (Debug ouvert) | S |
| Spawn toutes créatures | Shift + S |
| Spawn créature aléatoire | F9 |
| Nettoyer plateau (Cheat) | C |
| Révéler tout (Cheat) | F5 |
| Cacher tout (Cheat) | F6 |
| Débloquer Navigation (Cheat) | F7 |
| Retirer état Bloqué (Cheat) | F8 |
| Changer de grille active | 1 à 9 |
| Retour menu / Abandon | \ ou Échap |
| Remplir Inventaire (Debug) | B |

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

| Créature | Déclencheur      | Navigation             | Perception | Mode | AggressionBase |
|----------|------------------|------------------------|------------|------|----------------|
| **Lumifly** | Auto             | Attraction (baie)      | Manifest | Over | 0 | Régresse les plantes (décrémente leur stade) |
| **Shadowstalker** | Proximité (4)    | Attraction (Player)    | Cloaked | Swap | 80 | échange de place avec la cible (swap validé) |
| **Echo Hound** | Auto             | Relatif                | Manifest | Swap | 50 | échange de place avec la cible et rebondit 180° quand bloqué |
| **Burrower** | Auto             | Relatif                | Manifest | Under | 20 |
| **Specter** | Echo             | Errance                | Cloaked | Under | 60 |
| **Stonewarden** | OnReveal         | Orientation            | Manifest | Normal | 40 | attaque = rotation grille 90° + shader quake |
| **Moss Monkey** | Proximité (4)    | Attraction (Empty)     | Manifest | Normal | 0 (dynamique) |
| **Flutterwing** | Proximité (2)    | Répulsion (Player)     | Manifest | Over | 0 |
| **Fleeing Sprite** | Proximité (3)    | Répulsion (Player)     | Manifest | Normal | 0 | Fuit le joueur. Attaque : **Vertige** (distorsion + aberration chromatique). Butin : Vision des Intentions. |

#### Types de navigation

| Type | Description |
|------|-------------|
| `NavWander` | Errance directionnelle |
| `NavPatrol` | Suit un itinéraire prédéfini |
| `NavRelative` | Par rapport à son orientation absolue (saute si bloqué) |
| `NavOrientation` | Selon la direction du regard |
| `NavAttraction` | Vise une cible spécifique (Ressource par nom, Joueur, etc.). **Fallback** : errance aléatoire si pas de cible. |
| `NavRepulsion` | S'éloigne de la cible. **Fallback** : errance aléatoire si pas de cible. |

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

#### Validation du Swap (ModeSwap)

Le mode `ModeSwap` est soumis à une **validation bidirectionnelle** avant exécution pour respecter les règles de cohabitation :

1. **Retrait temporaire** : Les deux entités (créature et cible) sont retirées de leur tuile respective.
2. **Vérification créature → tuile cible** : `IsWalkable` vérifie les règles de cohabitation (même espèce interdite, taille/poids, max 3 créatures).
3. **Vérification cible → tuile origine** :
   - Si la cible est une créature : `IsWalkable` vérifie les mêmes règles.
   - Si la cible est une ressource : `HasResourceAt` vérifie qu'aucune ressource n'existe déjà sur la tuile (interdiction de doublon).
   - Si la cible est un piège : pas de restriction.
4. **Échec** : Si une validation échoue, les entités sont restaurées et le mouvement est annulé.
5. **Succès** : Les entités sont placées sur leurs nouvelles tuiles via manipulation directe des slices `EntitiesID`.

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

---

### Système d'Agressivité (AggressionSystem)

**Fichier** : `internal/domain/system/aggression_system.go`  
**Tests** : `internal/domain/system/aggression_test.go`

#### Architecture
L'`AggressionSystem` remplace l'ancienne logique de confrontation statique par un système modulaire et dynamique. Il s'exécute en **Priority 1** (avant le mouvement) dans l'Engine.

#### Composantes de l'Agressivité
Chaque créature a un composant `Behavior` enrichi :
- `AggressionBase` (int) : Valeur de base par espèce (0-80).
- `AggressionFactors` (map[string]int) : Facteurs dynamiques recalculés chaque tour.
- `Aggression` (int) : Total = Base + ΣFacteurs, plafonné à 100.
- `RevealCount` (int) : Compteur de révélations manuelles joueur.

#### Facteurs Dynamiques (mis à jour dans `Update()`)
| Facteur | Source | Description |
|---------|--------|-------------|
| `reveals` | `TileRevealed` (player_action) | `RevealCount * (100 / (MaxSafeReveals + 1))` |
| `inventory` | Inventaire joueur | +50 par objet taggé `{species}_trophy` |
| `species_anger` | Grille | +20 par congénère révélé |
| `empty_plots` | Singe Mousse | % cases vides * 2 (max 100) |
| `toxic_dreamberry` | Lumifly | 100 si Dreamberry stade 4 adjacent |

#### Déclencheur d'Attaque
L'attaque ne se produit **pas** à la révélation, mais à la **fin de l'animation de flip** (`AnimationEnded`, type="flip", state=Revealed). Si `Aggression >= 100` :
1. Vérifie si le joueur est dans la `ThreatZone` (même logique périphérique qu'avant).
2. Publie `CreatureAttacked` (payload: `hit_target` = position joueur si menacé, nil sinon) → Renderer lance l'animation de lunge.
3. Si joueur menacé et pas de *Grâce* : `TakeDamage(10, "physical")` + `PlayerDamaged` event + effets visuels (Blur Shadowstalker, Bulle Lumifly, Vertige Fleeing Sprite).

#### Difficulté
`MaxSafeReveals` et `AggressionMult` dans `DifficultySettings` contrôlent la tolérance et l'intensité globale.

#### Feedback Visuel
Le `BoardRenderer` affiche une **barre d'agressivité** (orange→rouge) sous les créatures révélées quand `Aggression > 0`.

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

### Système de Combo (ComboSystem)

**Fichier** : `internal/domain/system/combo_system.go`  
**Tests** : `internal/domain/system/combo_system_test.go`

#### Architecture
Le `ComboSystem` (Priority 10) récompense les associations consécutives. Il s'exécute après tous les autres systèmes pour traiter les événements `TileMatched` et `TileMerged`.

#### Messages et Progression

| Combo Count | Message | Score Bonus |
|-------------|---------|-------------|
| 1 | GOOD! | 5 XP |
| 2 | NICE! | 10 XP |
| 3 | GREAT! | 15 XP |
| 4 | SUPER! | 20 XP |
| 5 | AWESOME! | 25 XP |
| 6 | EXCELLENT! | 30 XP |
| 7 | MARVELOUS! | 35 XP |
| 8 | INCREDIBLE! | 40 XP |
| 9 | UNSTOPPABLE! | 45 XP |
| 10+ | GODLIKE!!! | 50 XP |

- **Synergie** : Si le match implique plusieurs types d'association (`assoc_types` > 1), le message est toujours `"SYNERGY!"` avec +50 XP bonus.
- **Merge** : La fusion publie `"MERGE!"` sans incrémenter le combo (score fixe : 10 XP).

#### Juiciness (1-5)

La juiciness détermine l'intensité visuelle du message :

| Condition | Juiciness |
|-----------|-----------|
| comboCount >= 1 | 1 |
| comboCount > 2 | 2 |
| comboCount > 4 | 3 |
| comboCount > 7 | 4 |
| comboCount > 10 | 5 |

Bonus synergie : +1 (plafonné à 5).

#### Effets Visuels

| Juiciness | Fond | Texte | Animations |
|-----------|------|-------|------------|
| 1 | Bleu-gris | Blanc | — |
| 2 | Olive | Jaune | — |
| 3 | Orange foncé | Or | Tremblement 2px |
| 4 | Rouge | Rouge corail | Tremblement 4px + particules |
| 5 | Violet | Arc-en-ciel | Tremblement 6px + particules + rainbow |

- **Slide-in** : Offset initial 100px, décroissance ×0.85/frame.
- **Outline** : Texte avec contour noir 8-directions (1px).
- **Durée** : Persiste jusqu'à la fin du tour (`TurnCreated`).

#### Conditions de Réinitialisation

- Tour avancé sans match invalide.
- `PlayerDamaged` avec reason `"invalid_match"` ou `"skipped_valid_match"`.

#### Événement ComboTriggered

Publié via `PublishImmediate` (sinon perdu dans `ProcessQueue`) :

```go
ComboTriggered  // payload: text, count, score, juiciness
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

Pour générer le fichier WebAssembly compatible avec Itch.io :

```powershell
# 1. On renomme temporairement le fichier de ressources Windows pour éviter les conflits
Rename-Item ./cmd/game/rsrc.syso rsrc.syso.bak

# 2. On compile pour le WebAssembly
$env:GOOS="js"; $env:GOARCH="wasm"; go build -o hgcv0.2_basic-incursion.wasm ./cmd/game

# 3. On restaure le fichier de ressources
Rename-Item ./cmd/game/rsrc.syso.bak rsrc.syso
```

