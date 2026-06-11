# Assets - Hunter Gatherers Concentration

⚠️ **ATTENTION : ASSETS TEMPORAIRES / PLACEHOLDERS**

Tous les assets visuels actuels sont **générés procéduralement par le code** et servent de 
**placeholders temporaires** pour le développement. Ils seront remplacés par des assets
finaux (sprites, pixel art, illustrations) avant la release.

## Statut des Assets

 Type | Statut | Priorité | Notes |
------|--------|----------|-------|
 Tuiles avec thèmes | 🟡 Temporaire | Moyenne | Thèmes : Default, Forest, Cave, Desert, Swamp |
 Ressources | 🟡 Temporaire | Haute | Génération procédurale avec stades de maturation |
 Créatures | 🟡 Temporaire | Haute | Génération procédurale avec comportements |
 Traces (Tracks) | 🟡 Temporaire | Moyenne | Boue, griffures, herbe brisée, empreintes, rayons |
 Audio | 🟡 Temporaire | Moyenne | Non implémenté |

## Avantages des Assets Temporaires

- ✅ **Développement rapide** - Pas besoin d'attendre les assets finaux
- ✅ **Pas de dépendances** - Aucun fichier externe à gérer
- ✅ **Taille réduite** - Génération à la volée
- ✅ **Modifiables** - Ajustement facile via code (Paramètres et Palettes)
- ✅ **Libres de droit** - CC0 / Domaine public

## Remplacement par des Assets Finaux

Pour remplacer un asset temporaire :

1. Ajouter le fichier image dans `assets/images/`
2. Modifier `internal/infrastructure/assets/manager.go` pour charger le fichier via `ebitenutil.NewImageFromFile`
3. Conserver la génération procédurale comme fallback

### Exemple de chargement d'asset externe :

```go
// Charger une image externe
img, _, err := ebitenutil.NewImageFromFile("assets/images/dreamberry.png")
if err == nil {
    m.images["resource_dreamberry"] = img
} else {
    // Fallback sur la génération procédurale
    m.images["resource_dreamberry"] = generateDreamberry(size, DreamberryPalette, "fruit")
}
```

## 🎨 Nature des Assets

Contrairement à de nombreux jeux qui utilisent des images externes, tous les graphiques de ce jeu sont créés dynamiquement via des algorithmes de génération procédurale. Cela signifie :

- ✅ **Aucune dépendance externe** - Pas de fichiers image à gérer
- ✅ **Taille réduite** - Le code génère les images à la volée
- ✅ **Modifiable à l'infini** - Changez les paramètres pour obtenir des variantes
- ✅ **Libre de droit garanti** - Vous êtes propriétaire du code générateur

## 🗂️ Structure des Assets

Le gestionnaire d'assets (`internal/infrastructure/assets/manager.go`) centralise la génération et le cache.

### 1. Tuiles de Jeu (`tiles.go`)

Les tuiles utilisent un système de thèmes pour s'adapter à différents environnements :

 Thème | Description | Utilisation |
-------|-------------|-------------|
 `default` | Bleu-violet classique | Grille par défaut |
 `forest` | Vert naturel | Forêts, bois |
 `cave` | Sombre gris-violet | Cavernes, grottes |
 `desert` | Jaune/Orange sable | Déserts, dunes |
 `swamp` | Vert mystique / Ocre | Marais, zones humides |

Chaque thème définit des couleurs pour les états : `Hidden`, `Revealed`, `Matched`, `Blocked`, `Sealed`, `Trap`.

### 2. Ressources (`resources.go`)

Les icônes de ressources supportent des variantes selon leur stade de maturation.

 Ressource | Description | Stades supportés |
-----------|-------------|------------------|
 `dreamberry` | Baie onirique violette | bourgeon, fleur, fruit, gâté |
 `moonstone` | Pierre de lune bleutée | - |
 `whispering_herb` | Herbe murmurante | graine, pousse, mature |
 `crystal_shard` | Éclat de cristal | - |
 `moss_truffle` | Truffe de mousse | - |
 `void_bloom` | Fleur du vide | - |
 `echo_crystal` | Cristal d'écho | - |
 `sand_core` | Noyau de sable | - |

### 3. Créatures (`creatures.go`)

 Icône | Description | Palette |
-------|-------------|---------|
 `lumifly` | Luciole lumineuse | Jaune doré |
 `shadowstalker` | Rôdeur des ombres | Violet sombre |
 `burrower` | Fouisseur terrestre | Brun terreux |
 `flutterwing` | Papillon aérien | Bleu ciel |
 `specter` | Spectre spectral | Cyan pâle |
 `echo_hound` | Chien résonnant | Bleu profond |
 `moss_monkey` | Singe de mousse | Vert mousse |
 `stonewarden` | Gardien de pierre | Gris roc |
 `fleeing_sprite` | Esprit fuyant | Blanc étincelant |

### 4. Traces et Indices (`tracks.go`)

Les créatures marquent le terrain via des entités de traces :
- `mud` : Boue (Calque Under)
- `claws` : Griffures (Calque Over)
- `broken_grass` : Herbe brisée (Calque Under)
- `footprints` : Empreintes (Calque Normal)
- `intent_beam` : Rayon d'intention d'attaque (Calque Over)

### 5. Structures (`structures.go`)

- `dolmen` : Bloc de pierre rituel (Bloquant)
- `obelisk` : Monolithe ancien (Bloquant)
- `portal` : Portail de zone (Départ/Fin)
- `exit` : Indicateur de sortie de navigation

## 🎮 Utilisation dans le Code

```go
// Obtenir le manager d'assets
manager := assets.NewManager()

// Obtenir une tuile avec thème (ex: forêt)
tileImg := manager.GetTileImage("hidden", "forest")

// Obtenir une icône de ressource avec stade précis
berryImg := manager.GetResourceIcon("dreamberry", "fruit")

// Obtenir une icône de créature
flyImg := manager.GetCreatureIcon("lumifly")
```

## 📄 Licence

Tous les assets générés par ce code sont dans le domaine public.
Vous êtes libres de les utiliser, modifier et distribuer à des fins commerciales ou personnelles sans attribution.

## 🔧 Fichiers Source

- `manager.go` - Gestionnaire principal et cache
- `tiles.go` - Génération des tuiles et thèmes
- `resources.go` - Icônes de ressources (Palettes et Stades)
- `creatures.go` - Icônes de créatures
- `structures.go` - Génération des décors fixes
- `tracks.go` - Traces environnementales
