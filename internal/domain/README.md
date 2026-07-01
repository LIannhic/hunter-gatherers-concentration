# Domain Layer - Architecture et Design Patterns

Ce document décrit l'architecture et les patterns de conception utilisés dans la couche domain du jeu "Hunter Gatherers Concentration".

## Vue d'ensemble

Le domaine est organisé selon une architecture **Clean Architecture** avec une séparation claire des responsabilités. Le code est réparti dans des sous-packages thématiques pour faciliter la maintenance et les tests.

```
┌─────────────────────────────────────────────────────────────┐
│                        Domain Layer                          │
├─────────────┬─────────────┬─────────────┬───────────────────┤
│   Entity    │   Board     │ Component   │    Systems        │
│  (Identity) │  (Grid)     │   (Data)    │  (Logic/Update)   │
├─────────────┼─────────────┼─────────────┼───────────────────┤
│  Creature   │  Resource   │   Player    │      Meta         │
│  (Behavior) │  (Items)    │  (Stats)    │ (Progression)     │
├─────────────┴─────────────┴─────────────┴───────────────────┤
│                      Event Bus                               │
│                 (Communication)                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 1. Entity-Component-System (ECS)

### Pourquoi ECS ?

Le jeu nécessite une grande flexibilité : des entités (ressources, créatures) avec des comportements variés qui évoluent au fil du temps. L'ECS permet :
- **Composition over inheritance** : pas de hiérarchie de classes complexe
- **Flexibilité** : ajouter/retirer des comportements dynamiquement
- **Performance** : les systèmes traitent les données de manière cache-friendly

### Structure ECS

```go
// Entity - Identité unique + position
type Entity interface {
    GetID() ID
    GetType() Type
    GetPosition() Position
    SetPosition(Position)
    GetGridID() string
    SetGridID(string)
    IsActive() bool
    Deactivate()
    GetState() TileState
    SetState(TileState)
    AddTag(string)
    HasTag(string) bool
    RemoveTag(string)
}

// Component - Données pures, pas de logique
type Lifecycle struct {
    CurrentStage int
    MaxStages    int
    StageNames   []string
    Cyclic       bool // Si vrai, revient au stade 0 après le stade max
}

// System - Logique métier qui opère sur les composants
type LifecycleSystem struct{}
func (s *LifecycleSystem) Update(world *World) {
    // Met à jour tous les composants Lifecycle
}
```

### Implémentation

- **`system/`** : Logique centrale du domaine
  - `World` : Structure World (Cœur de l'état global).
  - `Engine` : Orchestrateur des systèmes ECS.
  - `ECS Systems` : Implémentations (IA, Mouvement, Lifecycle, Loot...).
  - `AggressionSystem` : Gère le calcul modulaire de l'agressivité (Base + Facteurs : révélations, patience, inventaire, colère d'espèce). Déclenche les attaques à 100%.
  - `CreatureAttackEffectSystem` : Centralise les effets mondiaux déclenchés par les attaques réussies (ex: rotation de grille, troubles cognitifs Aphasia/Ataxia/Agnosia/Amnesia/Vertigo).
  - `Mechanics` : Flip, Match, Merge.
  - `Navigation` : Gestion des grids et navigation.
  - `Entities` : Logique de spawn.
  - `Portal` : Système de portail portable (`world_portal.go`) — déploiement 3x3, effet séisme, pénalités (Rê, vortex shader, prévisualisation curseur.
  - `Query` : Fonctions de recherche.
  - `TriggerLumiflyEffect` : Publie l'événement `lumifly_effect_triggered` avec les positions des Lumifly, le rayon et la durée restante du tour (`TurnTimer.Remaining`).
- **`board/`** : Gestion de la géométrie et de la structure du monde
- **`entity/`** : Gestion des identités (`ID`, `Type`), des états (`TileState`), et du manager
  - `TileState` : Hidden, Revealed, Matched, Blocked
  - `Type` : Resource, Creature, Structure, Artefact, Trap, Loot
  - `CumulationLevel` : Niveau de cumul (0 à 2) utilisé pour la fusion et le scaling visuel.
  - `Manager` : Stockage et accès rapide aux entités
  - `DebugState` : État global de débogage permettant d'outrepasser les règles de difficulté, de filtrer les entités et de forcer les shaders. Inclut `MessageSpeed float64` (défaut 1.0) contrôlable via F12 `+`/`-`.
  - `AddTag(string)`, `HasTag(string)`, `RemoveTag(string)` : Méthodes permettant de gérer les propriétés dynamiques ou visuelles des entités (ex: "moss_lure", "flying").
  - `ThreatZone` : (Creature) Liste de directions attaquées localement.
  - `Behavior` : Composant IA enrichi — `AggressionBase` (statique), `Aggression` (total calculé), `AggressionFactors` (map[string]int : "reveals", "inventory", "species_anger", "empty_plots", "toxic_dreamberry"), `RevealCount` (compteur révélations manuelles).
- **`component/`** : Stockage et définition des composants (`Store`)
- **`world.go`** : Agrégateur de l'état global (Grids, Entities, Player).
- **`engine.go`** : Orchestrateur des systèmes. Sépare la logique en trois cycles :
  - `Reset()` : Réinitialise l'état interne de tous les systèmes (Previews, Mouvements, Combos) lors du démarrage d'une nouvelle session. Garantit un état propre (ex: effacement des "révélations récentes" qui pourraient déclencher les Stonewardens).
  - `Update()` : Cycle par tour.
  - `UpdateFrame(dt)` : Cycle temps réel à 60 FPS.
- **`system.go`** : Implémentation des systèmes ECS qui traitent les données.
  - `CreatureAISystem` : Gère les comportements de base des créatures
  - `CreatureMovementSystem` : Implémente le mouvement avancé avec triggers, navigation, modes. **Filtre les déclencheurs par grille** : `TriggerOnReveal`, `TriggerOnEcho`, `TriggerProximity` ne réagissent qu'aux événements sur la grille de la créature. Stocke les révélations avec `RevealedTile{Position, GridID}`. **Validation du swap** : Le mode `ModeSwap` est soumis à une validation bidirectionnelle avant exécution — retrait temporaire des deux entités, vérification `IsWalkable` (règles de cohabitation créature) et `HasResourceAt` (interdiction de doublon ressource). Si une validation échoue, le mouvement est annulé.
  - `LifecycleSystem` : Gère la maturation des ressources. Supporte les **cycles cycliques** : les ressources avec `Cyclic: true` reviennent au stade 0 après avoir atteint le stade maximum. Déclenche des événements `ResourceMatured` et met à jour les valeurs des ressources lors du bouclage du cycle pour garantir une durée visuelle cohérente.
  - `PropagationSystem` : Gère l'expansion organique des ressources. La propagation se déclenche au stade défini par `PropagationStage` dans le composant `Lifecycle` (par défaut le dernier stade).
  - `ToxicitySystem` : Gère les dégâts de poison cumulés et dégressifs infligés par les ressources toxiques (ex: Dreamberry stade 4). Vérifie les entités au sommet des piles **sur la grille actuelle uniquement** avec hazard actif + `IsConstant`.
  - `TriggerSystem` : Gère les structures interactives (terriers, etc.) et les dégâts de révélation (ex: Singe Mousse)
  - `PreviewSystem` : Gère la révélation temporaire des tuiles à l'entrée d'une zone
  - `LootSystem` : Gère la transformation des associations réussies en butin d'inventaire
  - `ActionSystem` : Gère les actions spécifiques des créatures (ex: `spawn_trap` du Singe Mousse)
  - `TrackSystem` : Gère la durée de vie et la disparition progressive des traces au sol
  - `AggressionSystem` : **Nouveau** — Calcule l'agressivité totale des créatures (Base + Facteurs). S'abonne à `TileRevealed` (reason: "player_action") pour le facteur "reveals" et à `CreatureMoved` pour le facteur "patience" (+2% par pas, +10% par rebond). Déclenche l'attaque si agressivité ≥ 100 à la fin de l'animation de flip.
  - `CreatureAttackEffectSystem` : Centralise les effets mondiaux des attaques réussies. Linke les impacts physiques aux altérations mentales (Aphasia, Ataxia, Agnosia, Amnesia, Vertigo) et transformations de terrain (Quake).
  - `ComboSystem` : Gère le système de combo (Priority 10). S'abonne à `TileMatched` pour incrémenter le compteur et calculer la juiciness (1-5). S'abonne à `TileMerged` pour publier un message sans incrémenter. S'abonne à `PlayerDamaged` pour réinitialiser le combo en cas d'erreur. Publie `ComboTriggered` via `PublishImmediate` (sinon perdu dans `ProcessQueue`).

**Note architecture importante** : À partir de la fusion du #18, l'état visuel (`TileState`) appartient à l'entité, pas à la tuile. Cela permet :
- Une gestion cohérente des états (l'entité contrôle sa visibilité)
- Une séparation claire : le plateau fournit la géométrie, les entités portent la logique
- Un système plus flexible pour les entités spéciales (ex: les portails de commencement qui se bloquent après un délai)
- **Unification de l'interface `Hoverable`** : Toutes les entités interactives (tuiles, butin) et les sorties implémentent `Hoverable`, permettant un effet d'inclinaison (tilt) unifié au survol.
- **Système de Cumul (Merge)** : Une mécanique permettant de fusionner des paires identiques pour augmenter leur valeur visuelle et stratégique avant la capture.
- **Système de Cadres d'Action** : Feedback hybride (permanent sur sélection + au survol) indiquant l'état d'interaction via `TileActionState` (None/Interactive/Impossible/Unavailable). Logique centralisée dans `handler.go:computeTileActionState()`, rendu dans `board_renderer.go:RenderTileActionFrame()`.

---

## 2. Factory Pattern

### Objectif

Créer des entités préconfigurées avec des valeurs par défaut cohérentes sans polluer le code métier avec des constructeurs complexes.

### Exemple : Création de créatures

```go
type Factory struct{}

func (f *Factory) Create(species string, pos entity.Position) (*Creature, error) {
    c := New(species, pos)
    
    switch species {
    case "lumifly":
        c.SetBehavior(component.Behavior{State: "pollinating"})
        c.SetMobility(component.Mobility{CanMove: true})
        c.AddTag("flying")
        
    case "shadowstalker":
        c.SetBehavior(component.Behavior{State: "hunting", Aggression: 80})
        c.AddTag("dangerous")
    }
    
    return c, nil
}
```

### Avantages

- **Centralisation** : La logique de création est au même endroit
- **Extensibilité** : Ajouter une nouvelle espèce = ajouter un case
- **Testabilité** : Facile de mocker les factories

---

## 3. Strategy Pattern (Associations)

### Objectif

Le cœur du jeu est le mécanisme d'association de tuiles (Memory). Pour le moment, seul le type d'association suivant est actif :
- **Identical** : Paire identique (même ID)

D'autres types d'associations sont prévus pour le futur et ne sont pas encore actifs :
- **Logical** : Clé/Serrure, Marteau/Enclume
- **Elemental** : Feu + Bois, Eau + Plante
- **Narrative** : Fragments d'histoire
- **Orientation** : Selon l'orientation des tuiles 

Le pattern Strategy permet de traiter ces différents types uniformément.

### Implémentation

```go
type Strategy interface {
    Type() Type
    CanAssociate(a, b Matchable) bool
    Resolve(a, b Matchable) (Result, error)
}

// Implémentations concrètes
type IdenticalStrategy struct{}
type LogicalStrategy struct{}
type ElementalStrategy struct{}

// Engine qui orchestre
func (e *Engine) TryAssociate(a, b Matchable) (Result, error) {
    for _, strategy := range e.strategies {
        if strategy.CanAssociate(a, b) {
            return strategy.Resolve(a, b)
        }
    }
    return Result{Success: false}, errors.New("aucune association")
}
```

### Extensibilité

Ajouter un nouveau type d'association :
```go
type TemporalStrategy struct{} // Associations basées sur le temps
engine.RegisterStrategy(&TemporalStrategy{})
```

---

## 4. Observer Pattern (Event Bus)

### Objectif

Découpler les systèmes qui produisent des événements de ceux qui les consomment. Éviter les dépendances cycliques.

### Cas d'usage

- Une créature se déplace → Le système de déplacement publie un événement
- Le système de score peut écouter et attribuer des points
- Le système d'affichage peut mettre à jour l'UI

### Implémentation

```go
// Publication
eventBus.Publish(event.NewCreatureMovedEvent(creatureID, from, to))

// Souscription
eventBus.SubscribeFunc(CreatureMoved, func(e Event) {
    // Réagir au mouvement
})

// Traitement batch (éviter les effets de bord en cascade)
// Appelé à chaque frame par l'Engine pour garantir la fluidité UI
eventBus.ProcessQueue()
```

### Types d'événements et Priorité UI

Certains événements critiques pour le rendu (comme `TileRevealed`) utilisent `PublishImmediate` pour forcer la mise à jour du Renderer avant la fin de la frame logique. Les autres utilisent `Publish` et sont consommés lors du `ProcessQueue`.

**Piège `ProcessQueue`** : `ProcessQueue()` itère sur un snapshot du slice (`for _, e := range b.queue`). Les événements publiés via `Publish()` pendant le traitement d'un handler sont perdus quand `b.queue[:0]` vide la file après l'itération. **Tout handler qui publie un événement chaîné doit utiliser `PublishImmediate()`**.

Le payload de `TileRevealed` inclut désormais un champ `reason` pour distinguer l'origine de l'action :
- `"player_action"` : Révélation explicite par un clic ou une capacité du joueur. Déclenche les triggers d'IA (`OnReveal`).
- `"system_hide"` : Fermeture automatique (fin de tour, échec de match). Ignoré par les triggers d'IA.
- `"system_action"` : Autres actions automatiques (prévisualisation, scellage de zone). Ignoré par les triggers d'IA.

### Types d'événements

```go
CreatureMoved      // Déplacement
CreatureFled       // Fuite (ex: Singe Mousse)
ResourceMatured    // Changement de stade
ResourcePropagated // Expansion (directions cardinales uniquement)
AssociationMade    // Paire trouvée
PlayerDamaged      // Dégâts subis (reason: "physical", "toxicity", "match_error", etc.)
TurnEnded          // Fin de tour
CreatureAttacked   // Attaque de créature (agressivité ≥ 100) — payload: hit_target (*Position)
AmnesiaStarted     // Début d'amnésie (payload: turns int) — message droite "AMNÉSIE ! X tours."
AmnesiaEnded       // Fin d'amnésie — message gauche "La mémoire revient..."
ComboTriggered     // Combo actif — payload: text, count, score, juiciness (publié via PublishImmediate)
```

---

## 5. Adapter Pattern

### Objectif

Adapter l'interface `World` pour l'IA des créatures sans exposer tout l'état du monde.

### Implémentation

```go
// Interface minimale pour l'IA
type WorldState interface {
    GetPlayerPosition() entity.Position
    GetNearbyCreatures(pos entity.Position, radius int) []*Creature
    IsValidMove(pos entity.Position) bool
}

// Adaptateur
type worldAdapter struct {
    world *World
}

func (wa *worldAdapter) GetPlayerPosition() entity.Position {
    return wa.world.playerPosition
}
// ... implémentations limitées
```

### Avantages

- **Principe de moindre privilège** : L'IA n'a accès qu'à ce dont elle a besoin
- **Testabilité** : Facile de créer un mock WorldState pour tester l'IA

---

## 6. Repository Pattern (Entity Manager)

### Objectif

Abstraire le stockage et la récupération des entités.

### Implémentation

```go
type Manager struct {
    entities map[ID]Entity      // Accès par ID
    byType   map[Type]map[ID]Entity  // Index par type
    byPos    map[Position]ID         // Index spatial
}

func (m *Manager) Get(id ID) (Entity, bool)
func (m *Manager) GetByPosition(pos Position) (Entity, bool)
func (m *Manager) GetByType(t Type) []Entity
func (m *Manager) QueryByTag(tag string) []Entity
```

### Indexation spatiale

Le `byPos` permet des requêtes rapides : "Quelle entité est à la position (3,4) ?"

---

## 7. State Pattern (Créatures)

### Objectif

Les créatures changent de comportement selon leur état (chasse, fuite, pollinisation).

### Implémentation

```go
// L'état est stocké dans le composant Behavior
type Behavior struct {
    State string // "idle", "hunting", "fleeing", "regressing"
}

// L'AI utilise l'état pour décider
func (ai *SimpleAI) Decide(c *Creature, world WorldState) Action {
    switch c.Behavior.State {
    case "fleeing":
        return ai.flee(c, world)
    case "hunting":
        return ai.hunt(c, world)
    default:
        return ai.idle(c, world)
    }
}
```

---

## 8. Inventaire et Acquisition de Butin (Loot)

### Objectif

L'inventaire agit comme un tampon entre la session de récolte onirique et le foyer familial (méta-progression).

### Fonctionnement du LootSystem

1. **Détection** : Le système écoute les événements `TileMatched`.
2. **Instanciation** : Il crée un `LootItem` à partir des métadonnées de l'entité matchée. `LootItem` est maintenant une entité de plein droit (`entity.Entity`) de type `TypeLoot`.
3. **Grille d'Inventaire** : L'inventaire est géré comme une `Grid` dédiée (`InventoryGridID = "inventory"`). Cela permet d'utiliser les mêmes systèmes de rendu et de survol que pour le plateau de jeu.
4. **Transfert** : L'objet est ajouté à la fois à la liste logique du joueur via `AddLootItem` et placé spatialement sur la grille d'inventaire pour la synchronisation.
5. **Overflow** : Si l'inventaire est plein (30 slots), le butin est détruit et un événement `InventoryFull` est émis pour déclencher un retour visuel.

### Caractéristiques de l'Inventaire

- **Multi-sélection** : Permet de sélectionner plusieurs objets pour une suppression groupée.
- **Sécurité** : Certains objets (ex: Portail Portatif) possèdent le tag `IsDeletable: false` et ne peuvent pas être supprimés par le joueur.
- **Affichage** : Utilise un système de clipping et de défilement fluide (pixel par pixel) pour suggérer la profondeur de la réserve.
- **Atlas Technique** : Une fenêtre d'atlas (T) permet de vérifier visuellement le rendu de chaque entité. Elle supporte la pagination et utilise un système de boutons pour une navigation simplifiée.
- **Support des Niveaux** : Les constructeurs d'objets (`NewXXXItem`) acceptent un paramètre de niveau pour instancier directement des butins puissants.
- **Valorisation du Butin** : Lors de la victoire, chaque item en inventaire est converti en score (XP) : 100 XP par ressource, 250 XP par créature.

---

## 9. Rendu Stratifié et Illusion de Profondeur

### Objectif

Bien que le jeu soit en vue aérienne 2D, il simule une profondeur via un système de trois calques conceptuels (Under, Normal, Over).

### Fonctionnement

Le domaine communique l'intention de profondeur au moteur de rendu via les événements de mouvement (`CreatureMoved`) :

- **Calque Under** : Pour les entités fouisseuses (Burrower) ou les traces profondes (boue, herbe brisée). Une entité en mode `Under` est animée et visible si la pile de tuiles à sa position est vide, créant l'illusion qu'elle rampe sous les parcelles.
- **Calque Normal** : Pour les tuiles physiques (Memory), les ressources et les déplacements standards.
- **Calque Over** : Pour les entités volantes ou les effets de surface (griffures).

### Fusion et Cumul (MERGE)

Une nouvelle étape de gameplay s'insère avant la capture :
- **Principe** : Fusionner 2 entités identiques de même niveau via le bouton **MERGE**.
- **Effet** : Une entité est retirée, l'autre augmente son `CumulationLevel` (Niveau max 2).
- **Match** : Le bouton **MATCH** valide des paires de même rang. Un match de haut niveau donne un butin plus puissant. Les associations sont désormais gérées de manière centralisée par l'**AssocEngine** dans la couche Domain, qui impose une égalité stricte des niveaux de cumul.
- **Mana** : La fusion a un coût progressif, et la révélation d'une tuile cumulée consomme du Mana.

Certains butins peuvent être utilisés directement depuis l'inventaire pour octroyer des bonus :

- **Shadowstalker** : Octroie l'état **Évanescent** pendant 1 tour. Le joueur ne subit aucun dégât (physique ou d'erreur de match). Le cadre de sélection du plateau devient **gris** pour signaler cet état.
- **Echo Hound** : Déclenche un scanner révélant les positions des entités cachées.
- **Fleeing Sprite** : Révèle visuellement les zones de menace de toutes les créatures sur la grille actuelle pendant 1 tour via des arcs blancs.
- **Flutterwing** : Restaure 10 Sanité et accorde l'état **Grâce** (3 tours), permettant d'éviter les attaques lors des révélations.
- **Burrower** : Force une créature à laisser des traces de boue lors de son prochain déplacement.
- **Spectre** : Fait disparaître une paire de créatures du plateau.
- **Ressources Globales** : Restaurent de la santé, de la mana ou de la santé mentale (+5).
- **Ressources Exclusives** : Restaurent des statistiques de manière plus puissante (+15) ou équilibrée.

### Alchimie et Pénalités de Ressources

Tout comme les créatures, l'interaction avec les ressources est soumise à une rigueur alchimique :
- **Match Invalide** : Tenter d'appairer des ressources incompatibles provoque une déstabilisation magique (-5 Mana par ressource révélée).
- **Skip de Match Valide** : Ignorer une paire de ressources compatible gaspille leur potentiel énergétique (-5 Mana par ressource révélée).

### Animation de fermeture et relief

Pour renforcer l'immersion, les tuiles ne se referment pas aléatoirement :
- **Pente (Tilt)** : Elles utilisent la propriété `Tilt` de la parcelle pour une retombée "naturelle".
- **Navigation** : Les entrées/sorties se scellent (Révélé -> Caché) de l'intérieur vers l'extérieur tant que la zone n'est pas sécurisée, et s'ouvrent à nouveau (Caché -> Révélé) une fois les objectifs atteints.

- **Toxicity (Poison)** : Les ressources peuvent posséder un composant `Hazard` définissant des dégâts de zone ou locaux. Le poison est cumulatif mais dégressif (diminishing returns). Les Dreamberries sont toxiques au stade 4.

- **Standardized Events** : Les dégâts au joueur sont centralisés via `NewPlayerDamagedEvent`, unifiant les retours visuels (HUD/Renderer) pour les attaques, les échecs de match et la toxicité.

### Orientation Persistante et Mathématiques D4

Le moteur gère une unification réelle des transformations géométriques :
- **Unification de l'Orientation** : `GetOrientation()` combine dynamiquement l'orientation "intrinsèque" de la créature (définie par son espèce) avec la transformation D4 actuelle de sa tuile. Cela garantit que les comportements d'IA (comme `NavRelative`) utilisent toujours le regard absolu sur la grille.
- **Composition SUR l'état** : Chaque nouveau mouvement (flip) est appliqué sur l'état actuel de l'entité (`apply * current`). Cela respecte la logique physique où le joueur manipule une tuile déjà orientée.
- **Fermeture Physique** : La fermeture d'une tuile (via la pente du terrain) est une transformation réelle qui modifie l'orientation logique face cachée.
- **Réversibilité** : Grâce aux propriétés du groupe $D_4$, deux flips identiques s'annulent ($T^2 = I$), permettant de retrouver l'état d'origine si le joueur et le terrain agissent sur le même axe.
- **Nomenclature Relative** : Les créatures utilisent des directions relatives (`Forward`, `Backward`, `Left`, `Right`) pour définir leurs zones de menace. Ces directions sont transformées en coordonnées absolues du plateau via l'orientation unifiée de l'entité.

### Niveaux de Difficulté

Le domaine définit quatre niveaux de difficulté influençant la génération, le rythme et l'agressivité :
- **Easy** : Timer 15s, prévisualisation 1.3s, `MaxSafeReveals: 3`, `AggressionMult: 0.5`.
- **Normal** : Timer 10s, prévisualisation 0.8s, `MaxSafeReveals: 2`, `AggressionMult: 1.0`.
- **Hard** : Timer 5s, prévisualisation 0.3s, `MaxSafeReveals: 1`, `AggressionMult: 1.5`.
- **Insane** : Timer 5s, prévisualisation 0.1s, `MaxSafeReveals: 0`, `AggressionMult: 2.0`.

**MaxSafeReveals** : Nombre de révélations "sûres" par créature avant que l'agressivité n'atteigne 100%. Formule : `increment = 100 / (MaxSafeReveals + 1)` par clic.
**AggressionMult** : Multiplicateur global appliqué à l'agressivité totale (implémentation future).

### Time Scaling (Gestion Dynamique du Temps)

Le `TurnTimer` adapte sa vitesse selon le contexte dans `Engine.UpdateFrame` :

| Condition | Time Scale | Comportement |
|-----------|------------|--------------|
| **Grille vide** (aucune Resource/Creature) | 0.0 | Timer **arrêté** (`Stop()`) |
| **Prévisualisation active** (`PreviewSystem.IsPreviewActive`) | 0.5 | Timer **ralenti 50%** |
| **Animation active** (`World.ActiveAnimationCount > 0`) | 0.5 | Timer **ralenti 50%** |
| **Normal** | 1.0 | Vitesse normale |

Priorité : Grille vide > Animation/Preview > Normal.

Le `World.ActiveAnimationCount` est incrémenté par `AnimationStarted` et décrémenté par `AnimationEnded` (géré par `BoardRenderer.SubscribeToEvents`).

### Distinction Invisibilité vs Profondeur

- `hidden: true` (Furtivité) : L'entité est réellement invisible (ex: Shadowstalker). Le rendu saute l'animation de déplacement.
- `mode: "under"` (Profondeur) : L'entité est physiquement sous les autres, mais le joueur peut la voir si rien ne la recouvre. L'animation de déplacement est maintenue pour guider l'œil du joueur.

### Correspondance Inter-Zones (Cross-Zone Matching)

Le système d'appariement supporte maintenant les tuiles sur **grilles différentes** :
- `revealedGridIDs []string` dans Handler : grille d'origine par tuile révélée (parallèle à `revealedTiles`).
- `MatchTilesCommand.GridID2` : grille de la seconde tuile (optionnel, défaut = `GridID`).
- `processMatchAttempt()` résout chaque tuile sur sa grille respective via `revealedGridIDs[0]` / `[1]`.
- `processSkip()` utilise aussi `revealedGridIDs` pour pénalités inter-zones.

### Traces de Pas (Footsteps)

Système de traces visuelles pour les déplacements de créatures :
- **Entité `Track`** : Champs `OffsetX`, `OffsetY`, `Angle` (positionnement bord de case, rotation vers centre).
- **Types** : Boue (Under, interstice), Herbe Brisée (Under, origine), Griffures (Over, destination), Empreintes (Normal, sous tuiles), Intent Beam (Over, créature→cible).
- **FIFO max 2** : `footstepTrackIDs` dans Handler, nettoyage dans `OnTurnEnd` callback.
- **Indicateur de position** : `DrawPlayerPosition()` — affiche la position réelle du joueur (grid pos + ancrage) comme indicateur persistant au bord du plateau.

---

## 10. Persistance et Stockage

Le domaine définit une interface de persistance (`Repository`) pour découpler la logique de sauvegarde des détails techniques.

### Interface Repository

```go
type Repository interface {
    Save(slotID int, data *SaveData) error
    Load(slotID int) (*SaveData, error)
    Delete(slotID int) error
    GetAllMetadata() ([]Metadata, error)
    GetLatestSlotID() (int, error)
    Exists(slotID int) bool
}
```

### Implémentations

- **`JsonRepository`** : Utilisé sur Desktop, stocke les fichiers dans un dossier local (`./saves`).
- **`WebRepository`** : Utilisé dans les builds WebAssembly (Itch.io), utilise le `LocalStorage` du navigateur via `syscall/js`.

---

## Flux de données

```
1. Joueur révèle une tuile (Reason: "player_action")
   - Sur Desktop : Clic gauche (Press)
   - Sur Mobile : Tap ou Relâchement (Release)
   ↓
2. World.RevealTile() → Événement TileRevealed (avec `grid_id`, `flip_direction` et `reason`)
   ↓
3. L'UI / le renderer démarre l'animation de flip et met à jour l'affichage
   ↓
4. Engine.Update() progresse d'un tour
   ↓
5. LifecycleSystem : les ressources mûrissent
   CreatureAISystem : les créatures se déplacent (Vérifient `reason` pour TriggerOnReveal)
   TriggerSystem : vérifie les conditions de déclenchement
   ↓
6. EventBus.ProcessQueue() traite les événements
   ↓
7. Mise à jour de l'affichage
```

---

## Testing

L'architecture facilite les tests unitaires :

```go
// Test d'une stratégie d'association
func TestIdenticalStrategy(t *testing.T) {
    strategy := &IdenticalStrategy{}
    a := &MockMatchable{matchID: "card1"}
    b := &MockMatchable{matchID: "card1"}
    
    if !strategy.CanAssociate(a, b) {
        t.Error("Should associate identical items")
    }
}

// Test de l'AI avec un WorldState mocké
func TestCreatureAI(t *testing.T) {
    mockWorld := &MockWorldState{playerPos: Position{X: 5, Y: 5}}
    creature := NewCreature("test", Position{X: 0, Y: 0})
    
    action := ai.Decide(creature, mockWorld)
    // Vérifier l'action
}
```

---

## Bonnes pratiques

1. **Données séparées de la logique** : Les composants sont des structs simples, la logique est dans les systèmes
2. **Immutabilité préférée** : Les événements sont immutables, les composants sont modifiés par les systèmes
3. **Pas de dépendances circulaires** : Les packages dépendent de `entity` et `component`, jamais l'inverse
4. **Interfaces minimales** : `WorldState` pour l'IA, `Matchable` pour les associations

---

## Ajouter une fonctionnalité

### Exemple : Ajouter un nouveau système

```go
// 1. Créer le système
 type WeatherSystem struct{}
 
 func (s *WeatherSystem) Priority() int { return 5 }
 
 func (s *WeatherSystem) Update(world *World) {
     // Modifier les ressources selon la météo
 }
 
// 2. L'enregistrer dans l'Engine
 engine := NewEngine(world)
 engine.AddSystem(&WeatherSystem{})
```

### Exemple : Ajouter un nouveau composant

```go
// 1. Définir le composant
 type WeatherSensitive struct {
     PreferredWeather string
 }
 func (w WeatherSensitive) Type() string { return "weather_sensitive" }

// 2. L'ajouter aux entités concernées
 resource.Components.Add(entityID, &WeatherSensitive{PreferredWeather: "rain"})

// 3. Le système Weather peut le lire
 if comp, ok := components.Get(id, "weather_sensitive"); ok {
     ws := comp.(*WeatherSensitive)
     // Appliquer les effets
 }
```
