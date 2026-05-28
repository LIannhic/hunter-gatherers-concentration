# Hunter Gatherers Concentration

> Ce puzzle-aventure réinvente le "Memory" en outil de survie stratégique. Associez des tuiles pour capturer des entités, récolter des ressources et créer des artefacts. Un défi cognitif au tour par tour où chaque erreur de mémoire fragilise votre progression.

---

## Concept

Le projet propose une expérience de puzzle-aventure en tour par tour, où la mécanique du "Memory" est réinventée pour devenir un outil d'exploration et de survie. Dans ce jeu, l'association de paires ne se limite pas à la simple identité visuelle ; elle devient un langage stratégique permettant d'interagir avec un environnement complexe. Le joueur doit faire preuve de mémoire, mais aussi d'intuition logique pour capturer des entités, récolter des composants et manufacturer des artefacts. En transformant un exercice cognitif en une boucle de gameplay profonde, le jeu invite à une réflexion calme mais tendue, où chaque erreur peut altérer la stabilité de l'expérience.

## Contexte

L’univers du jeu prend racine dans un monde marqué par la coexistence de la réalité matérielle et de plans de réalités alternatifs, des dimensions oniriques et ésotériques où les souvenirs et les symboles acquièrent une substance physique. Dans ces contrées instables, les ressources ne sont pas inertes : elles naissent, mûrissent et réagissent selon des cycles mystérieux que seuls les initiés savent décrypter. Le récit suit le parcours d'un jeune protagoniste soudainement propulsé au rang de "récolteur" suite à une rupture brutale de son apprentissage. Sans avoir achevé sa formation, il hérite de la lourde responsabilité de subvenir aux besoins d'un foyer fragilisé par l'absence et les dettes.
L’enjeu narratif réside dans la tension entre la grisaille du monde réel et l'étrangeté fascinante des plans parcourus. Chaque incursion est une nécessité vitale : le héros doit apprendre à déchiffrer les indices d'un environnement qui cherche autant à se dissimuler qu'à l'entraver. La survie ne dépend pas seulement de sa force, mais de sa capacité à comprendre la faune éthérée qui marque son territoire et transforme le décor. Le contexte impose ainsi un poids émotionnel à chaque tour de jeu : le joueur n'explore pas par simple curiosité, mais pour allouer les gains de ses découvertes à une famille dont l'avenir dépend de sa maîtrise croissante des lois invisibles de ces dimensions.

## Fonctionnalités principales et Contenus

Le projet se définit par un système de jeu hybride alliant le puzzle mental, la survie et la gestion. Le contenu principal repose sur une mécanique d'association étendue : le joueur ne se contente pas de mémoriser des visuels, mais doit comprendre les liens logiques, élémentaires et narratifs qui unissent les tuiles pour synthétiser des ressources complexes ou capturer des entités. Ce système est soutenu par un écosystème dynamique où les ressources ne sont pas de simples objets inertes, mais des entités vivantes capables de maturer, de se propager ou de réagir aux interactions environnementales. Le plateau de jeu devient ainsi un monde en constante mutation, influencé par des structures interactives telles que des terriers ou des zones de dissimulation qui modifient la visibilité et la sécurité du plan.
L'expérience propose une profondeur stratégique s'appuyant sur une double boucle de progression. En mission, le joueur doit composer avec des créatures aux comportements riches, capables de transformer le terrain ou de laisser des traces persistantes de leur passage, offrant un défi de lecture environnementale permanent. Hors mission, le contenu se déplace vers une gestion de méta-progression exigeante : le foyer familial devient un enjeu central où chaque ressource collectée doit être judicieusement allouée pour assurer la survie du groupe et l'évolution des capacités du protagoniste. Cette richesse de contenu vise à récompenser l'expertise du joueur, transformant chaque incursion en une opportunité de déchiffrer les secrets d'un univers où l'indice visuel et le cycle de vie des éléments sont aussi cruciaux que la mémoire pure.

## Mécanisme

Le gameplay repose sur une gestion rigoureuse des ressources et du temps, dictée par un système en tour par tour. Chaque interaction, qu’il s’agisse de retourner une tuile ou d’utiliser un objet, consomme une unité de ressource (temps ou mana), forçant le joueur à planifier ses mouvements au sein d’une limite de tours impartis. La survie dépend de la gestion d'une barre de santé, physique ou mentale, qui s’érode au fil des erreurs ou des confrontations.
L’aspect central de l’association étendue enrichit la mécanique de mémoire classique : le joueur doit identifier des paires dont la corrélation peut être identique, logique (clé et serrure), élémentaire ou narrative. La difficulté est modulable via des variables structurelles, comme l'éparpillement des paires ou la visibilité de l'inventaire. Enfin, l'environnement est rendu vivant et menaçant par la présence de créatures aux comportements déterminés : celles-ci occupent des placements précis et effectuent des déplacements prévisibles mais contraignants, obligeant le joueur à adapter sa stratégie de mémorisation en fonction de leurs mouvements sur le plateau.

### Minimap et Exploration (Fog of War)

Le plan onirique est un réseau de zones interconnectées. Pour s'orienter sans briser le mystère, la minimap utilise un système de **brouillard de guerre dynamique** :

- **Rendu Centré** : La zone actuelle est toujours fixée au centre de la minimap (grille 9x9). Le reste du monde glisse autour lors des transitions.
- **États de Découverte** :
  - **Inconnu** : Zone cachée.
  - **Adjacent** : Silhouette grise (route possible identifiée).
  - **Visité** : Affiche l'icône du biome (Forest, Cave, Swamp, etc.).
  - **Actuel** : Indicateur doré pulsant.

### Matrice de Dégâts (MATCH vs SKIP)

Le jeu punit l'inattention et la précipitation via une matrice de décision stricte lors de l'interaction avec des paires de créatures :

| État de la paire | Action: **MATCH** | Action: **SKIP** |
| :--- | :--- | :--- |
| **VALIDE** | 0 Dégât (Succès) | Dégâts de groupe (Erreur) |
| **INVALIDE** | Dégâts de groupe (Échec) | 0 Dégât (Prudence) |

- **Dégâts de groupe** : `nombre de créatures révélées * 10`.
- **Match Valide** : Les créatures sont capturées (Loot).
- **Match Invalide** : Les tuiles se referment violemment.
- **Skip** : Permet de refermer les tuiles sagement si aucune paire n'est identifiée.

### Confrontation et Zones de Menace

Chaque créature possède une **Zone de Menace** (directions qu'elle attaque).
- **Placement Périphérique** : Le joueur agit depuis le bord du plateau. Sa position (`entity.Position`) et son ancrage (`BorderPosition`) sont déterminés par l'endroit où il clique pour révéler une tuile.
- **Calcul de Confrontation** : Lors du dévoilement, si le joueur se trouve dans la zone de menace de la créature (après application de la transformation D4/Flip), il subit **10 points de dégâts physiques**.
- **Esquive** : En choisissant de "tirer" la tuile depuis un angle mort de la créature, le joueur peut éviter l'attaque lors de la révélation.

### Compte à rebours temps réel (Turn Timer)

Pour rompre le rythme classique du Memory et simuler l'urgence de la survie en plan onirique, chaque tour est soumis à une **pression temporelle dynamique** :

- **Durée** : dépend de la difficulté (10s en Easy, 8s en Normal, 5s en Hard, 4s en Insane).
- **Reset** : le timer se réinitialise à chaque action volontaire (retourner une tuile, Match, Skip, Fin de tour).
- **Auto-skip** : si le timer atteint 0, le système simule automatiquement un **Skip** — les tuiles retournées se referment et le tour est consommé, entraînant la pénalité de Santé Mentale associée.
- **Feedback visuel** :
  - Le bouton **Skip** se remplit progressivement d'une couleur violette, puis passe au rouge brique en phase d'alerte.
  - La jauge de **Santé Mentale** tremble de plus en plus fort lorsque le timer descend sous les 3 secondes (phase de panique).

### Boutons d'action réactifs du Playmat

Quatre boutons fixes occupent les coins du Playmat. Leur accessibilité est purement réactive :

| Bouton | Position | Comportement |
|--------|----------|--------------|
| **MATCH** | Haut-gauche | Actif uniquement quand 2 tuiles sont retournées. Valide la paire si les entités sont identiques (coûte du Mana). |
| **SKIP** | Haut-droite | Actif uniquement quand 2 tuiles sont retournées. Referme les tuiles et consomme le tour (coûte de la Santé Mentale). Affiche le remplissage temporel. |
| **TURN** | Bas-gauche | Toujours actif. Force la fin du tour. |
| **MENU** | Bas-droite | Toujours actif. Retour au menu principal. |

### Troubles cognitifs (Status Effects)

Le joueur peut subir des altérations mentales qui déforment l'interface :

- **Aphasia** – brouille les labels des boutons (ex: "MATCH" devient "???" ou est permuté).
- **Agnosia** – rend les boutons visuellement indifférenciés (couleurs altérées).
- **Ataxia** – scramble les positions des boutons (ex: le bouton Match peut basculer au coin bas-droit).
- **Amnesia** – désactive aléatoirement des boutons (30% de chance d'"oublier" un bouton).

Ces effets sont interceptés par le système de rendu **avant** l'affichage, forçant le joueur à lutter contre sa propre interface.

## Verbes

Pour le joueur :Pendant les parties (Actions directes)

* Dévoiler : Révéler le contenu d’une tuile face cachée (consomme du temps ou du mana).
* Prendre : Collecter des éléments ou capturer des créatures simples via l’identification d’une paire identique.
* Associer : Créer un lien logique entre deux tuiles complémentaires (ex: Clé + Serrure) pour synthétiser des ressources ou capturer des entités complexes.
* Utiliser : Activer un outil, une ressource consommable ou une capacité de créature (ex: cri de l'Echo Hound) pour modifier le plateau ou protéger ses statistiques. Nécessite une confirmation par un second clic sur l'objet sélectionné.
* Déployer portail : Activer un portail portable pour créer une zone de dégagement 3x3 et initier l’extraction du plan.
* S’extraire : Initier la fin de l’incursion pour sécuriser le butin avant l’épuisement des ressources de survie.

Entre les parties (Méta-progression)

* Commencer : Choisir une destination onirique et lancer une nouvelle session d'exploration.
* Échanger : Troc ou vente des ressources collectées contre des devises ou des objets rares.
* Allouer : Distribuer les gains au foyer pour répondre aux besoins de la famille (nourriture, dettes, santé).
* Apprendre : Investir dans des compétences pour débloquer de nouveaux types d'associations ou améliorer les résistances du héros.
* Équiper : Préparer son inventaire et ses outils en fonction des dangers prévus pour la prochaine mission.

Savoir du joueur (Apprentissage et Déduction)

* Identifier : Reconnaître le stade de maturation d'une ressource (du bourgeon au fruit gâté) pour optimiser le moment de la récolte.
* Déchiffrer : Interpréter les indices visuels dissimulés sur le recto/verso des tuiles ou dans l'arrière-plan pour deviner le contenu caché.
* Prédire : Anticiper le comportement d'une créature ou l'évolution d'un cycle environnemental en fonction de l'expérience acquise.
* Distinguer : Différencier une opportunité d'un danger (piège) grâce à l'observation de détails subtils sur le plateau de jeu.

Pour les créatures :

* Se placer : se positionner au départ.
* Se déplacer : changer position au cours de la partie.
* Fuir: quitter la zone de jeu.
* Attaquer: infliger des dégâts au joueur.
* Altérer : modifie le statut du joueur, transforme l'environnement.
* Transformer : Agir sur les ressources pour les faire évoluer, les dégrader ou modifier leur accessibilité (ex: polliniser, briser, fertiliser).
* Marquer : Laisser des traces persistantes de son passage sur le plateau (empreintes, griffures), offrant au joueur des indices sur sa position ou son trajet.

### Système de Déplacement Avancé

Les créatures disposent d'un système de mouvement configurable avec les paramètres suivants :

#### Déclencheurs (quand se déplacer)
- **Passif** : Aucun mouvement
- **Auto** : À la fin de chaque tour
- **Vue** : Dès que révélée
- **Echo** : Quand une autre tuile est révélée
- **Proximité** : Si action dans un rayon de N cases

#### Navigation (où aller)
- **Errance** : Direction aléatoire
- **Patrouille** : Suit un itinéraire défini
- **Orientation** : Selon la direction du regard
- **Attraction** : Vise une cible spécifique
- **Répulsion** : S'éloigne d'une cible

#### Modes de déplacement
- **Bento** : Déplacement visible (le joueur voit le mouvement)
- **Shadow** : Déplacement invisible (face cachée, le joueur doit deviner)
- **Swap** : Échange de place avec la cible
- **Over** : Calque supérieur (vol, effets de surface)
- **Under** : Calque inférieur (fouissage, traces profondes)

### Traces et Indices Environnementaux

Les créatures laissent derrière elles des traces qui respectent des règles de placement précises sur les calques de rendu :

- **Boue (Mud)** : Rendu sur le calque **Under**, positionné exactement entre deux cases (interstice) et orienté selon la direction du mouvement.
- **Herbe Brisée (Broken Grass)** : Rendu sur le calque **Under**, marquant la case d'origine du déplacement.
- **Griffures (Claws)** : Rendu sur le calque **Over**, marquant la case de destination (impact).
- **Empreintes (Footprints)** : Rendu sur le calque **Normal**, apparaissant sous les tuiles physiques pour simuler le passage au sol.

Toutes les traces s'adaptent dynamiquement à la distance entre les cases, qu'il s'agisse d'un plateau 3x3 ou 6x6.

### Bestiaire

| Créature | Déclencheur | Navigation | Mode | Collision | Description |
|----------|-------------|------------|------|-----------|-------------|
| **Lumifly** | Auto | Errance (nord) | Over | Glisse | Insecte lumineux qui vole au-dessus du plateau |
| **Shadowstalker** | Proximité (4) | Attraction joueur | Shadow | Rebond | Prédateur qui chasse discrètement le joueur |
| **Burrower** | Vue | Errance | Under | Phase (terre) | Créature fouisseuse qui se cache sous terre |
| **Specter** | Echo | Errance | Shadow | Phase (murs) | Fantôme qui traverse les murs |
| **Stonewarden** | Passif | Patrouille | Bento | Stop | Gardien immobile qui patrouille si révélé |
| **Echo Hound** | Echo | Attraction curseur | Bento | Glisse | Chien rapide qui réagit aux révélations. |
| **Moss Monkey** | Proximité (4) | Target Empty | Bento | Glisse | Saboteur qui rebouche les cases vides avec des leurres. Fuit si le plateau est saturé de pièges. |

Pour les ressources :

* Mûrir / Maturer : Évoluer d'un stade initial (ex: bourgeon) vers un stade optimal (ex: fruit mûr) puis vers un stade dégradé (ex: fruit gâté), changeant ainsi sa valeur et ses propriétés d'association.
* Se propager : S'étendre aux tuiles adjacentes (uniquement directions cardinales) si certaines conditions sont remplies. Une faible probabilité (5%) de stérilité peut freiner cette expansion.
* Réagir : Modifier son état au contact d'un autre élément ou d'une créature (ex: une ressource qui "éclot" si une créature pollinisatrice passe dessus).
* Se dégrader : Perdre en qualité ou disparaître si elle n'est pas récoltée à temps ou si le plan devient instable.
* Rayonner : Laisser filtrer des indices visuels sur l'arrière-plan ou le verso de la tuile (ex: une ressource luminescente qui "brille" légèrement à travers la carte face cachée).
* S'altérer : Changer de nature sous l'influence d'un événement climatique ou psychique (ex: un minerai qui devient instable ou explosif).


Verbes de l'Environnement (Décors et Structures)

* Dissimuler : Masquer la présence d'une créature ou d'une ressource (ex: de hautes herbes ou un brouillard psychique sur la tuile).
* Abriter / Révéler : Servir de refuge à une entité (ex: le terrier). Si le refuge est dévoilé en même temps que l'entité, celle-ci est débusquée.
* Déclencher (Trigger) : Activer un événement si une condition est remplie (ex: si le terrier ET la créature sont visibles, la créature fuit).
* Entraver : Restreindre les mouvements des créatures ou l'accès du joueur à certaines tuiles (ex: ronces, rochers).

### Note aux collaborateurs

Ce document pose des piliers mécaniques et narratifs. Tout le reste, l'esthétique précise, le bestiaire, les collectables, les ressources, l’écosystème, l’environnement, la structure des plans parallèles, l’histoire, les personnages, les lieux, etc...,  est un espace ouvert à vos idées et à votre expertise.

---

## Design et travaux préparatoires

[Consulter le tableau blanc sur Figma.](https://www.figma.com/design/Pzh46PCN6sdWgwqBhf8bvn/Jeux-Vid%C3%A9o?node-id=325-2&t=69lWglExoEjbC8O1-1)

---

## Installion & lancement

### Prérequis

* go

### Récupérer le projet

* cloner le dépôt
* Installer les dépendances

```bash
go mod download
```

### Lancer le projet

```bash
# Mode normal
go run ./cmd/game

# Mode debug (avec logs détaillés)
go run ./cmd/game -debug

# Build production
go build -o game ./cmd/game
./game
```

### Contrôles

| Action | Touche |
|--------|--------|
| Révéler tuile | Click gauche |
| Matcher (ou valider paire) | M |
| Skip (ou passer quand 2 tuiles) | Espace |
| Sélection Butin (Usage) | Click gauche (Inv) |
| Utiliser Butin sélectionné | Re-click gauche (Inv) |
| Sélection Suppression | Click droit (Inv) |
| Désélectionner tout | Click droit (vide) |
| Naviguer zones | ZQSD / Flèches |
| Statistiques zones | I |
| Liste Inventaire | L |
| Fin de tour | Espace (hors match en cours) |
| Menu / Abandon | Échap |
| Changer de grille | 1-9 |
| Difficulté | F1 à F4 |
| Révéler tout (Cheat) | F5 |
| Cacher tout (Cheat) | F6 |
| Rotation plateau | + / - |
| Reset rotation | R |
| Remplir Inv (Debug) | B |
| Spawn entités (debug) | S |
| Spawn toutes créatures (debug) | Shift+S |
| Nettoyer plateau (debug) | C |
| Retour menu | \ |
Annulation de la PR #39
