# TODO

This file tracks intentionally ugly or temporary implementation choices, plus
large gameplay/rendering work that should not become invisible project
assumptions.

## Technical Debt

### Runtime And Portability

- **Lua 5.1 `.lub` execution**
  - Current state: RO `.lub` bytecode is parsed in `res/lub.go`, translated into
    GopherLua bytecode, then executed by GopherLua.
  - Ugly part: GopherLua requires a private `FunctionProto.stringConstants`
    field for `GETGLOBAL` and `SETGLOBAL`, so we currently set that field with
    `reflect` and `unsafe`.
  - Better fix: upstream or fork a small public API in GopherLua for externally
    built prototypes, or own a minimal Lua 5.1 bytecode runner for RO data files.
  - Also revisit whether the old fallback evaluator is still needed after enough
    real `.lub` files are covered by tests.

- **`nofakecgo` build tag requirement**
  - Current state: builds are expected to use `CGO_ENABLED=0 -tags nofakecgo`.
  - Ugly part: the tag exists to avoid duplicate fake-cgo symbol providers in
    the pure-Go GoGPU dependency stack.
  - Better fix: remove the tag requirement once the dependency stack no longer
    needs competing fake-cgo shims.

### Rendering

- **Dynamic world geometry**
  - Current state: static GND surfaces, lightmapped GND subdivisions, and RSM
    placements are retained `render.WorldMesh` objects. They are built once for a
    loaded map/model placement, uploaded into persistent GPU buffers, and reused.
  - Current state: actor/NPC/mob/item/effect sprite billboards use the dedicated
    shared-quad billboard pipeline with per-instance data instead of per-frame
    quad vertex slices.
  - Current state: shader-side camera projection and fog are used for retained
    meshes and billboards; render stats expose `world_mesh_commands`,
    `retained_world_meshes`, `world_billboards`, and remaining dynamic
    `world_vertices`.
  - Remaining dynamic geometry is intentional: animated water waves, warp/effect
    cylinders/rings, STR effects with animated per-corner coordinates, tile
    cursor geometry, and debug/fallback paths.
  - Better fix: move water and common effect primitives to specialized
    instanced/procedural GPU pipelines so `world_vertices` mostly measures only
    rare debug/fallback geometry.

- **Billboard depth and clipping**
  - Current state: sprites use 3D-ish billboard placement with custom depth
    handling to avoid being wrongly hidden by terrain/buildings.
  - Ugly part: RO sprites are not physically normal 3D quads; correct occlusion
    still needs policy knobs and reference-client comparison.
  - Better fix: document and implement the exact reference-client/OpenMidgard approach
    for sprite depth bias, top/head clipping, and object interaction.

- **Lighting model**
  - Current state: fog uses the reference client's camera defaults and near/far
    table values, applies the ramp at full strength, derives depth from the
    camera projection, and uses a non-additive color mix. Additive GND lightmap
    layers are explicitly exempt from fog so fog is applied once to the composed
    world color.
  - Ugly part: parts of the RSW/GND lighting pipeline are still approximations,
    especially the split base/lightmap rendering and per-map visual parity.
  - Better fix: collapse GND base, lightmap alpha, posterized light color, and
    fog into one shader path like the reference client, then add screenshot-style regression
    fixtures for representative outdoor, indoor, and dungeon maps.

### Gameplay State

- **Local death as held animation**
  - Current state: local player death uses the transient actor animation path,
    with the final death frame held until positive HP or map change.
  - Ugly part: reference client models death as persistent entity action/state, not as
    an expiring combat-style animation.
  - Better fix: split persistent actor state such as idle, walk, sit, and dead
    from transient overlays such as attack, hurt, and pickup, then clear dead
    state only on resurrection, respawn, or map transition.

### Data And Protocol

- **Fallback resource tables**
  - Current state: some NPC/monster/job/resource names still have hardcoded
    fallback maps.
  - Ugly part: these can hide `.lub` or DB parsing bugs and can become stale.
  - Better fix: rely on client data tables first, keep fallbacks only as explicit
    diagnostics or emergency compatibility.

- **Packet-version assumptions**
  - Current state: packet client date/profile defaults target the current rAthena
    setup.
  - Ugly part: several protocol choices are still date/profile-sensitive.
  - Better fix: centralize packetver negotiation/configuration and document which
    server builds are supported.

- **Real-data tests**
  - Current state: real RO data tests are gated by `GORO_DATA_DIR`.
  - Ugly part: CI cannot validate the most important compatibility paths without
    proprietary data.
  - Better fix: keep small synthetic fixtures for parsers and maintain a local
    real-data test checklist for releases.

## Effects

### Original-client job audit

Active follow-up to the first-job, Knight/Lord Knight, Crusader/Paladin,
Priest/High Priest, Monk, Wizard, Sage, Assassin/Rogue, and Soul Linker passes.
For each class: inspect every skill against dhxj and the 2008 HighPriest client,
cross-check server packets, implement discrepancies, review and fix twice,
validate, then commit and push. Existing support is not proof of visual parity.

Reference caveat from the Assassin Cross pass: HighPriest and the actual
`2008-09-10aSakexe.exe` are not interchangeable. HighPriest uses different
Soul Breaker/Meteor Assault poses and adds an Asura sound/quake to Soul Breaker.
Where they disagree, verify the actual Sakexe binary before choosing behaviour.

- [x] Hunter — traps, Detecting, Blitz Beat, Phantasmic Arrow, and utility/passive skills audited; reviewed twice and validated with OldRO assets.
- [x] Sniper — all four skills audited; procedural visuals, casting poses, hit routing, and sound timing reviewed twice and validated with OldRO assets.
- [x] Bard — songs/ensembles, Musical Strike, Frost Joke, Pang Voice, and utility/passive skills audited; reviewed twice and validated with OldRO assets.
- [x] Clown — all six 2008 skills audited; rays, sphere, link/body effects, Hermode cubes, and Tarot timing reviewed twice and validated with OldRO assets.
- [x] Dancer — all nine skills and shared ensembles audited; native dances, Slinging Arrow, Scream, and Wink reviewed twice and validated with OldRO assets.
- [x] Gypsy — all six shared skills and inherited dances audited; both reviews and actual female ACT checks passed, using the Clown/Dancer implementation.
- [x] Blacksmith — all 24 skills audited, including Full Adrenaline Rush; native buff visuals, packet ownership, Hammer Fall, and sound timing reviewed twice and validated with client assets.
- [x] Whitesmith — all five active skills audited; Cart Termination, buff poses/audio, and shared Overthrust afterimages reviewed twice and validated with client assets.
- [x] Alchemist — all 20 skills audited; bottle flights, Chemical Protection, homunculus casts, Demonstration, Berserk Pitcher, and Twilight Alchemy reviewed twice and validated with client assets.
- [x] Creator — all four active skills audited; potion bursts, Acid Demonstration, protection, and Cultivation reviewed twice and validated with client assets.
- [x] High Wizard — all six skills audited; Soul Drain, Amplification, Magic Crasher, Napalm Vulcan, Ganbantein, and Gravitation reviewed twice and validated with client assets.
- [x] Professor — all eight skills audited; particle trails, linking/body effects, ground units, casting, and sound timing reviewed twice and validated with client assets.
- [x] Assassin Cross — all five skills audited; slashes, poison particles, packet ownership, casting, poses, and sounds reviewed twice and validated with client assets.
- [ ] Stalker
- [ ] Champion
- [ ] Taekwon
- [ ] Star Gladiator
- [ ] Ninja
- [ ] Gunslinger

Hunter reference checks: `dhxj` and 2008 `HighPriest` skill action/effect
routing, trap state changes, particle primitives, and rAthena trap packets.
Trap bursts are owned by look changes, Shock Wave loops on its unit, and
Remove Trap/Spring Trap resolve the trap position before removal. Added missing
procedural effects, original sound delays, Ankle Snare's one-shot model
animation, trap picking/chasing, and Talkie Box's bubble-only packet. Passives
(Beast Bane, Falconry Mastery, Steel Crow) have no invented activation effect;
the existing falcon option owns the bird. Full tests with client assets and a
normal build passed. Live visual comparison remains useful for fine tuning.

Sniper reference checks: both original clients agree on True Sight's layered
rays, Wind Walk's four strips and pale-green body flash, Sharp Shooting's
delayed rays and ACT 88 casting pose, and Falcon Assault's impact/quake timing.
The 2008 executable confirms the Korean sound filenames. Buff effects belong
to the packet source; secondary splash packets do not replay attacks. Falcon
Assault rotates only the rendered sprite during its burst, preserving the
server's facing afterwards. Full asset-backed tests and a normal build passed;
live comparison of these procedural visuals remains useful.

Bard reference checks: dhxj and 2008 HighPriest agree on the floating song
symbols, vertical ribbons, rotating bands, and Roki particles. Ground units own
these visuals; activation owns the single sound (Whistle and Dissonance are
silent). Musical Strike uses ACT 88 and the native arrow flight timing. Frost
Joke and Pang Voice use the finite shout pose; Pang Voice captures its target
position and camera-relative direction. Song/dance loops now use their correct
frame ranges and stop on Adaptation/status end. Musical Lesson has no activation
visual, and Encore uses the server's subsequent song notification. Full tests
with client assets and a normal build passed; live comparison remains useful.

Clown reference checks: dhxj and 2008 HighPriest agree on Arrow Vulcan's
36 delayed pink rays, white casting aura, ACT 88 release, and additional attack
replays. Moonlit uses a status-owned sphere on the caster, not ground tiles;
Marionette links two actors and applies the pink body glow until status end.
Hermode ground units own their blue cubes. Longing flashes the caster; Tarot
flashes its target and displays only the card selected by the server, at its
native height and fade timing, without duplicate sound. The later renewal
skill CG_SPECIALSINGER is outside this 2008 audit. Both review passes, full
tests with client assets, and a normal build passed; live comparison remains
useful for these procedural effects.

Dancer reference checks: dhxj and 2008 HighPriest agree on the five ground
visuals, including Slow Grace's green ribbons, Service's pink cross, and
Fortune's enlarged offset kiss. Ground units own the visuals; the skill packet
owns the sound. Slinging Arrow uses ACT 88 and native arrow timing, Scream
holds frame zero for 1200ms and sends only its caster's chat line, and Wink
uses a kiss emote plus the directional sprite shared with Pang Voice. Their
shared sprite primitive now follows the caster's lifetime while retaining the
captured target position. Dancing Lesson is passive; shared ensembles and
Adaptation/Encore reuse the Bard audit. Both reviews, full asset-backed tests,
and a normal build passed; live visual comparison remains useful.

Gypsy reference checks: dhxj and 2008 HighPriest use the same six CG skill
paths as Clown, without gender-specific effect variants; rAthena's pre-renewal
tree confirms that set and the inherited Dancer skills. The shared effects
and lifecycle checks pass. Actual Gypsy body assets were checked in all eight
directions: Arrow Vulcan releases with ACT 88, replays the normal whip attack
for subsequent hits, and the ensemble dance has its full 1,2,3,2 sequence.
No additional runtime code was needed. Both reviews and focused tests with
client assets passed; live comparison remains useful as for Clown/Dancer.

Blacksmith reference checks: dhxj and 2008 HighPriest agree on Adrenaline's
20 screen-space rays, four rising orbiting particles, and sounds at ticks 0/30.
Overthrust starts a fading ring and two sounds; its status tint and afterimages
were completed in the Whitesmith pass. Weapon Repair and Maximize Power use their complete STR duration and
native audio ticks; Weapon Perfection stays on the recipient. Hammer Fall uses
the attack pose while casting, releases in the skill pose, and places a fresh
STR on the ground with its sound at tick 30 and quake at tick 35. Greed uses
the pickup pose and caster sound. Forging/passive skills introduce no activation
visual; existing crafting-result packets own success/failure effects. rAthena's
party-buff notifications and roBrowser's explicitly simplified Adrenaline entry
were cross-checked. Both reviews, full asset-backed tests, and a normal build
passed; live comparison remains useful for the new procedural visuals.

Whitesmith reference checks: dhxj and 2008 HighPriest agree on Meltdown and
Cart Boost's standing release pose, caster ownership, STR assets, and sounds.
Maximum Power Thrust reuses the Overthrust ring; both statuses now produce the
native red tint and frozen, fading sprite copies (five-tick spacing, alpha 180
decreasing by eight per tick, no copies on dyed bodies). These reuse cached
billboards and are cleared on actor removal/map entry. Cart Termination starts
on its victim at release, including misses: at tick 30 it plays Fire Pillar's
impact sound and emits four groups of twelve pink-white light particles, with
native movement, slowdown, fade, and ground clipping. It adds no hit spark or
secondary-splash replay. Weapon Refinement already uses the item-selection UI
and server-owned refinement success/failure effects. Create Coin, Create
Nugget, and System Create are unused draft skills, disabled in rAthena's tree;
no behavior was invented for them. Both reviews, the full test suite with
client assets, and the normal build passed. Live visual comparison is still
useful, particularly for the new actor afterimages.

Alchemist reference checks: dhxj and 2008 HighPriest agree on the bottle
assets, 25-tick flight speed, arc, spin, afterimages, and potion-level colors.
Acid Terror throws at release; Potion Pitcher uses its success packet, while
separate server notifications own the healing visuals. Berserk Pitcher retains
its casting throw pose and applies the recipient's Berserk Potion effect,
including its red tint, growing pillar, rising streaks, and tick-18 sound/quake.
Chemical Protection now draws its eight blue-white rays on the recipient.
Homunculus utility skills use the pink Couple Casting rings. Demonstration's
unit owns its looping, ground-anchored sprite; ground notifications animate
the caster without duplicating the field. Twilight I–III use casting effects
573–575, scattering 320 ingredients around the local viewer as both originals
do, with incremental updates and at most three draw calls. Completion uses
only the native release flare. ACT assets determine throw animation speed;
the old SetAction speed argument is unused in the 2008 executable. Pharmacy's
existing crafting UI/results and summon/homunculus entity lifecycles remain
server-owned. Passives and unused draft homunculus skills have no invented
activation effects. Both review passes, full asset-backed tests, and a normal
build passed. No live playtest was performed; visual comparison remains useful.

Creator reference checks: dhxj and 2008 HighPriest agree on the five-frame
ACT 40 casting sequence for Aid Condensed Potion and Acid Demonstration;
release preserves that pose. Potion bursts select red/yellow/white by level,
fall with three afterimages, and emit their expanding ground wave; the red
variant draws it twice. The activation sound uses the native fixed attenuation,
and separate server heal packets own healing. Acid Demonstration launches its
two bottles at cast ticks 5/10, then places four dark rings at the victim's
release position, with immediate sound, tick-5 quake, and per-hit flashes.
Textures preserve their original alpha and black outlines. Full Chemical
Protection reuses the recipient's chemical rays and blue-white body flash;
Cultivation adds only its white caster flash and skill pose, leaving plants to
the server. rAthena's tree and skill handlers confirm the four active skills;
Alchemy/Synthesis Potion are unused drafts, and inherited skills reuse the
Merchant/Alchemist audits. Both reviews, the full suite with client assets, and
a normal build passed. No live playtest was performed; visual comparison remains
useful, particularly for the new procedural bursts and rings.

High Wizard reference checks: dhxj and 2008 HighPriest agree on Soul Drain's
three curved particle trails, blue body flashes, and forced ACT 88 pose, which
preserves an existing walk. Mystical Amplification uses the server's buff state
to retain its alternating additive weapon pass; activation refreshes the effect
and plays the original Korean sound. The weapon alternation uses 60 Hz effect
ticks rather than render frames. Magic Crasher starts at release, shakes at tick
30 for 650 ms, flashes only the body additively, and retains per-hit flashes.
Napalm Vulcan plays its five delayed sounds; both originals call Hit2 after its
particle creation condition has passed, so no extra bursts were invented.
Ganbantein uses the native white ground rings and release sound. Gravitation's
ground unit owns its four stone/ice spikes and lifetime; ground notification
only animates/sounds the caster. Casting poses, aura colors, splash suppression,
and miss handling were checked against the original packet handlers and
rAthena. Professor HP Conversion's shared routing was corrected to effect 383
alone; its renderer remains for the Professor audit. Both reviews, the full
suite with client assets, and a normal build passed. No live playtest was
performed; visual comparison remains useful for the procedural effects.

Professor reference checks: dhxj and 2008 HighPriest agree on HP Conversion's
five groups of three green particle trails and delayed blue expansion of the
body. Soul Change now connects the actors with its two-color beam, flashes both
bodies, and plays the STR on both participants; the Korean filename was
incorrectly written as `사랑효과` instead of the original `사람효과`. Soul Burn
adds Magic Crasher's flash, sound, and quake on the server-selected recipient,
including the caster on failure. Its separate level-5 damage packet adds only
the generic hit flash. Mind Breaker follows the 2008 transient random tint and
tick-25 Provoke sound; later dhxj instead keeps the tint through the status.
Memorize flashes yellow and plays Suffragium; Double Casting flashes red with
an expanded body, the native release flare, and centered amplification sound.
Wall of Fog uses the original translucent grey square walls and diagonals;
Spider Web uses the pulsing ground texture. Both are owned by server units,
without duplicate fields at cast notification. rAthena confirms the eight
active skills, packet outcomes, and unit lifetimes. Mage/Sage skills inherit
their earlier audits. Both reviews, full tests with client assets, and a normal
build passed. No live playtest was performed; visual comparison remains useful,
especially for the procedural trails and ground walls.

Assassin Cross reference checks: dhxj and HighPriest describe four delayed
Soul Breaker slashes and eight groups of four Meteor Assault slashes, with
acceleration, fade, fixed launch position, and a brief purple caster flash.
The actual 2008 Sakexe confirms normal-alpha pale-blue slashes (renderer at
`0x62a6e0`), weapon-dependent attack poses for both skills (`0x6847c0`), and
no extra Asura sound/quake in Soul Breaker's release (`0x686972`). These last
two behaviours differ from HighPriest. Soul Breaker's spatial sound starts
at tick 20; Meteor Assault sounds immediately and only its successful
no-damage packet starts the burst. Damage packets add the ordinary hit flash;
secondary splash packets do not replay effects. Enchant Deadly Poison restores
the small rising particles, alternating purple body flash, and six Enchant
Poison sounds at ticks 0/5/11/18/26/40. Create Deadly Poison reuses crafting
success/failure effects; rAthena owns production, self-damage on failure, and
the poison status. Advanced Katar Mastery is passive. Both reviews, full tests
with client assets, and a normal build passed. No live visual comparison was
performed; the procedural geometry still benefits from playtesting.

Previous catalog import source: roBrowser's `src/DB/Effects/EffectConst.js`,
`src/DB/Effects/EffectTable.js`, `src/DB/Skills/SkillEffect.js`,
`src/DB/Skills/SkillUnit.js`, and `src/DB/Items/ItemEffect.js`.

### Status

- Reference active numeric `EffectTable.js` entries: `607`.
- Reference numeric effect constants: `1147`.
- Goro effect catalog entries: `712`.
- Catalog size relative to all numeric constants: about `62.1%`.
- The catalog is not a strict subset of the active table: it also contains
  inactive, later-client, and synthetic routed effects. An exact active-table
  intersection report is still part of the backlog below.
- Coverage only counts world/effect IDs handled through `worldEffectSpecForID`
  and direct effect trigger mappings. It does not count UI rendering, damage
  number rendering, cursor drawing, actor sprite animation, or map logic that is
  not modeled as an effect ID.

### Current Shape

- Effects are primarily data-table driven through `skillEffectSpecs`,
  `worldEffectSpecs`, and parser-backed tests against roBrowser's effect table.
- Renderer support exists for STR, CYLINDER, 2D billboards, 3D billboards,
  SPR/ACT attachments, sprite-backed 3D effects, duplicate timing, delays,
  basic alpha/fade, color tint, size interpolation, and source-to-target motion.
- Special Go behavior is still allowed for effects that are not purely
  declarative in roBrowser data, such as camera shake, map transitions, or UI
  choices produced by a skill.
- RSW-placed map effects and map-wide weather are tracked separately in
  [docs/map-effects-weather-todo.md](docs/map-effects-weather-todo.md).

### Recently Covered Families

- Swordman first-job effects: Bash, Provoke, Magnum Break, Endure.
- Mage first-job effects: Napalm Beat, Soul Strike, Cold Bolt, Frost Diver,
  Stone Curse, Fire Ball, Fire Wall, Fire Bolt, Lightning Bolt, Thunder Storm,
  Sight, Safety Wall, and Energy Coat routing/effects where currently supported.
- Acolyte first-job effects: Heal, Increase/Decrease Agility, Aqua Benedicta,
  Signum Crucis, Angelus, Blessing, Cure, Ruwach, Pneuma, Teleport, Warp Portal,
  and Holy Light.
- Thief first-job effects: Steal, Hiding, Envenom, Detoxify, Sprinkle Sand,
  Throw Stone.
- Archer first-job effects: Improve Concentration, Double Strafe, Arrow Shower,
  arrow projectile and ammo handling.
- Merchant first-job effects: Mammonite, Cart Revolution, Crazy Uproar,
  cart state/appearance, vending bubbles and shop interaction.
- Ninja expanded-class support: normal/baby jobs, full skill tree and level
  metadata, roBrowser skill actions/effects, and Ninja status routing.
- Gunslinger expanded-class support: normal/baby jobs, full skill tree and
  level metadata, weapon actions, status icons, and roBrowser skill effects.
- Taekwon-family support: normal/baby Taekwon, Star Gladiator, and Soul Linker
  jobs, skill trees, stance/status behavior, mission packets, and ranking UI.
- Common effects: hit feedback, potion/food families, Heal/recovery feedback,
  base/job level-up, teleport/portal, refine/pharmacy success/failure.

### Priority Backlog

- [ ] Improve STR parity: blend modes, alpha edge behavior, texture frame
  selection, UV animation, layer timing, rotation/scale interpolation, source
  offsets, and subtractive/additive modes.
- [ ] Improve CYLINDER parity: partial circles, repeat texture axes, fixed
  perspective versus world-space orientation, rotation deltas, duplicate
  instances, randomized fields, and exact alpha behavior.
- [ ] Complete reference `wav` behavior: delayed sounds, randomized sounds,
  positional sounds, and no inferred sounds when roBrowser declares none.
- [ ] Finish generic routing for `SkillEffect.js`, `SkillUnit.js`, and
  `ItemEffect.js` so adding new effects is mostly data import instead of custom
  Go code.
- [ ] Add persistent attached lifecycle handling for buffs/status loops:
  follow actors while active, detach only when the reference client does, and
  clear on vanish/death/status removal.
- [ ] Add a debug command or CLI flag that spawns an effect ID at the player for
  fast visual comparison.
- [ ] Generate an `effects_coverage.json` style report with support levels:
  `none`, `sound-only`, `partial`, `close`, and `parity`.

### Gameplay-Visible Gaps

- [ ] Finish live visual/timing parity for hit and combat feedback. Catalog
  routing now covers `EF_HIT1` through `EF_HIT6`, elemental/poison variants,
  critical-related effects, and blocked/zero-damage feedback, but source-level
  coverage is not the same as verified visual parity.
- [ ] Finish ground-skill lifecycle and visual parity. Routing exists for
  Sanctuary, Magnus, Quagmire, traps, Bard/Dancer songs, Volcano, Deluge,
  Violent Gale, Land Protector, Spider Web, Basilica, Suiton, and Epiclesis;
  remaining work is exact geometry, timing, server update/removal behavior, and
  live comparison.
- [ ] Status loops: poison, curse, blind, silence, stun, sleep, confusion,
  frozen, stone curse, hiding/cloaking/invisible states, spirit spheres, aura
  loops, and actor-attached buff/debuff decorations.
- [ ] Second-job pre-renewal sweep: Knight/Crusader, Wizard/Sage,
  Priest/Monk, Assassin/Rogue, Hunter/Bard/Dancer, Blacksmith/Alchemist.
- [ ] Remaining expanded and later content: Super Novice, transcendent classes,
  renewal third jobs, and modern fourth-job effect IDs.
