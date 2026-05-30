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
}

// System - Logique métier qui opère sur les composants
type LifecycleSystem struct{}
func (s *LifecycleSystem) Update(world *World) {
    // Met à jour tous les composants Lifecycle
}
```

### Implémentation

- **`board/`** : Gestion de la géométrie et de la structure du monde
  - `Grid` : Plateau individuel (Tuiles, Biomes, Pentes)
  - `DreamPlane` : Réseau de grilles connectées. Gère les `DiscoveryStates` (Hidden, Adjacent, Visited) pour la minimap.
  - `LayoutGenerator` : Algorithmes de génération de la structure du plan onirique.
- **`entity/`** : Gestion des identités (`ID`, `Type`), des états (`TileState`), et du manager
  - `TileState` : Hidden, Revealed, Matched, Blocked
  - `Type` : Resource, Creature, Structure, Artefact, Trap, Loot
  - `Manager` : Stockage et accès rapide aux entités
  - `AddTag(string)`, `HasTag(string)`, `RemoveTag(string)` : Méthodes permettant de gérer les propriétés dynamiques ou visuelles des entités (ex: "moss_lure", "flying").
  - `ThreatZone` : (Creature) Liste de directions attaquées localement.
- **`component/`** : Stockage et définition des composants (`Store`)
- **`system.go`** : Systèmes qui traitent les données
  - `CreatureAISystem` : Gère les comportements de base des créatures
  - `CreatureMovementSystem` : Implémente le mouvement avancé avec triggers, navigation, modes
  - `LifecycleSystem` : Gère la maturation des ressources
  - `PropagationSystem` : Gère l'expansion organique des ressources
  - `TriggerSystem` : Gère les structures interactives (terriers, etc.) et les dégâts de révélation (ex: Singe Mousse)
  - `PreviewSystem` : Gère la révélation temporaire des tuiles à l'entrée d'une zone
  - `LootSystem` : Gère la transformation des associations réussies en butin d'inventaire
  - `ActionSystem` : Gère les actions spécifiques des créatures (ex: `spawn_trap` du Singe Mousse)
  - `TrackSystem` : Gère la durée de vie et la disparition progressive des traces au sol

**Note architecture importante** : À partir de la fusion du #18, l'état visuel (`TileState`) appartient à l'entité, pas à la tuile. Cela permet :
- Une gestion cohérente des états (l'entité contrôle sa visibilité)
- Une séparation claire : le plateau fournit la géométrie, les entités portent la logique
- Un système plus flexible pour les entités spéciales (ex: les portails de commencement qui se bloquent après un délai)
- **Unification de l'interface `Hoverable`** : Toutes les entités interactives (tuiles, butin) et les sorties implémentent `Hoverable`, permettant un effet d'inclinaison (tilt) unifié au survol.

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

Le cœur du jeu est le mécanisme d'association de tuiles (Memory). Différents types d'associations existent :
- **Identical** : Paire identique
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
eventBus.ProcessQueue()
```

### Types d'événements

```go
CreatureMoved      // Déplacement
CreatureFled       // Fuite (ex: Singe Mousse)
ResourceMatured    // Changement de stade
ResourcePropagated // Expansion (directions cardinales uniquement)
AssociationMade    // Paire trouvée
PlayerDamaged      // Dégâts subis
TurnEnded          // Fin de tour
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
    State string // "idle", "hunting", "fleeing", "pollinating"
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

---

## 9. Rendu Stratifié et Illusion de Profondeur

### Objectif

Bien que le jeu soit en vue aérienne 2D, il simule une profondeur via un système de trois calques conceptuels (Under, Normal, Over).

### Fonctionnement

Le domaine communique l'intention de profondeur au moteur de rendu via les événements de mouvement (`CreatureMoved`) :

- **Calque Under** : Pour les entités fouisseuses (Burrower) ou les traces profondes (boue, herbe brisée). Une entité en mode `Under` est animée et visible si la pile de tuiles à sa position est vide, créant l'illusion qu'elle rampe sous les parcelles.
- **Calque Normal** : Pour les tuiles physiques (Memory), les ressources et les déplacements standards.
- **Calque Over** : Pour les entités volantes ou les effets de surface (griffures).

### Effets des Objets de Butin (Loot)

Certains butins peuvent être utilisés directement depuis l'inventaire pour octroyer des bonus :

- **Shadowstalker** : Octroie l'état **Évanescent** pendant 1 tour. Le joueur ne subit aucun dégât (physique ou d'erreur de match). Le cadre de sélection du plateau devient **gris** pour signaler cet état.
- **Echo Hound** : Déclenche un scanner révélant les positions des entités cachées.
- **Fleeing Sprite** : Révèle visuellement les zones de menace de toutes les créatures sur la grille actuelle pendant 1 tour via des arcs blancs.
- **Flutterwing** : Restaure 10 Sanité et accorde l'état **Grâce** (3 tours), permettant d'éviter les attaques lors des révélations.
- **Burrower** : Force une créature à laisser des traces de boue lors de son prochain déplacement.
- **Spectre** : Fait disparaître une paire de créatures du plateau.
- **Ressources** : Restaurent de la santé, de la mana ou de la santé mentale.

### Animation de fermeture et relief

Pour renforcer l'immersion, les tuiles ne se referment pas aléatoirement. Elles utilisent la propriété `Tilt` (pente) de la parcelle. Cela simule une tuile qui "retombe" selon la gravité du terrain.

### Orientation Persistante et Mathématiques D4

Le moteur gère une accumulation réelle des transformations géométriques :
- **Composition SUR l'état** : Chaque nouveau mouvement (flip) est appliqué sur l'état actuel de l'entité (`apply * current`). Cela respecte la logique physique où le joueur manipule une tuile déjà orientée.
- **Fermeture Physique** : La fermeture d'une tuile (via la pente du terrain) est une transformation réelle qui modifie l'orientation logique face cachée.
- **Réversibilité** : Grâce aux propriétés du groupe $D_4$, deux flips identiques s'annulent ($T^2 = I$), permettant de retrouver l'état d'origine si le joueur et le terrain agissent sur le même axe.
- **Nomenclature Relative** : Les créatures utilisent des directions relatives (`Forward`, `Backward`, `Left`, `Right`) pour définir leurs zones de menace. Ces directions sont transformées en coordonnées absolues du plateau via la matrice D4 de l'entité.

### Distinction Invisibilité vs Profondeur

- `hidden: true` (Furtivité) : L'entité est réellement invisible (ex: Shadowstalker). Le rendu saute l'animation de déplacement.
- `mode: "under"` (Profondeur) : L'entité est physiquement sous les autres, mais le joueur peut la voir si rien ne la recouvre. L'animation de déplacement est maintenue pour guider l'œil du joueur.

---

## Flux de données

```
1. Joueur révèle une tuile
   ↓
2. World.RevealTile() → Événement TileRevealed (avec `grid_id` et `flip_direction`)
   ↓
3. L'UI / le renderer démarre l'animation de flip et met à jour l'affichage
   ↓
4. Engine.Update() progresse d'un tour
   ↓
5. LifecycleSystem : les ressources mûrissent
   CreatureAISystem : les créatures se déplacent
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
