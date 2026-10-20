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

- [x] Hunter — traps, Detecting, Blitz Beat, Phantasmic Arrow, and utility/passive skills audited; reviewed twice and validated with OldRO assets.
- [x] Sniper — all four skills audited; procedural visuals, casting poses, hit routing, and sound timing reviewed twice and validated with OldRO assets.
- [x] Bard — songs/ensembles, Musical Strike, Frost Joke, Pang Voice, and utility/passive skills audited; reviewed twice and validated with OldRO assets.
- [x] Clown — all six 2008 skills audited; rays, sphere, link/body effects, Hermode cubes, and Tarot timing reviewed twice and validated with OldRO assets.
- [ ] Dancer — analysis in progress
- [ ] Gypsy
- [ ] Blacksmith
- [ ] Whitesmith
- [ ] Alchemist
- [ ] Creator
- [ ] High Wizard
- [ ] Professor
- [ ] Assassin Cross
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

Previous catalog import source: roBrowser's `src/DB/Effects/EffectConst.js`,
`src/DB/Effects/EffectTable.js`, `src/DB/Skills/SkillEffect.js`,
`src/DB/Skills/SkillUnit.js`, and `src/DB/Items/ItemEffect.js`.

### Status

- Reference active numeric `EffectTable.js` entries: `607`.
- Reference numeric effect constants: `1147`.
- Goro effect catalog entries: `663`.
- Catalog size relative to all numeric constants: about `57.8%`.
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
