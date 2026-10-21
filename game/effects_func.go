package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
)

type effectFuncAdapter int

const (
	effectFuncUnknown effectFuncAdapter = iota
	effectFuncGroundSample
	effectFuncCastRing
	effectFuncLockOnTarget
	effectFuncLevel99Aura
	effectFuncLevel99Ground
	effectFuncLevel99Bubble
	effectFuncPropertyGround
	effectFuncGroundSigil
	effectFuncSpiritSphere
	effectFuncGumgang
	effectFuncFlatColorTile
	effectFuncGroundTexture
	effectFuncBodyColor
	effectFuncMapPillar
	effectFuncHitRing
	effectFuncHitParticles
	effectFuncBlessingCircle
	effectFuncEstin
	effectFuncRisingBuffParticles
	effectFuncHitLineBurst
	effectFuncEsma
	effectFuncKaizel
	effectFuncEsmaHitTexture
	effectFuncSoulLink
	effectFuncKaupeDodge
	effectFuncBlueCasting
	effectFuncSonicBlow
	effectFuncThrowItem
	effectFuncSlimPitcher
	effectFuncAcidDemon
	effectFuncGrimtooth
	effectFuncRisingParticles
	effectFuncSteal
	effectFuncMammoniteCoins
	effectFuncArrowShower
	effectFuncHunter
	effectFuncSniper
	effectFuncClownGypsy
	effectFuncBlacksmith
	effectFuncCartTermination
	effectFuncChemicalProtection
	effectFuncTwilight
	effectFuncBerserkPotion
	effectFuncSongMusic
	effectFuncSongVertical
	effectFuncSongLight
	effectFuncSongOut
	effectFuncDirectedSprite
	effectFuncCartRevolution
	effectFuncSprinkleSand
	effectFuncRogueBurst
	effectFuncImpactParticles
	effectFuncBash
	effectFuncBashHit
	effectFuncEndure
	effectFuncMagnumBreak
	effectFuncBowlingBash
	effectFuncSpearBoomerang
	effectFuncShieldBoomerang
	effectFuncShieldChain
	effectFuncPressure
	effectFuncBashRays
	effectFuncGrandCross
	effectFuncCrusaderDefense
	effectFuncProvidence
	effectFuncDevotionLink
	effectFuncGuard
	effectFuncEquipmentBurst
	effectFuncGospelCross
	effectFuncSpiralPierce
	effectFuncAgility
	effectFuncMonkRays
	effectFuncTripleAttack
	effectFuncChainCrush
	effectFuncGlyphs
	effectFuncTaekwonStorm
	effectFuncStarRings
	effectFuncStarBlind
	effectFuncTaekwonStop
	effectFuncChemicalDash
	effectFuncTaekwonJump
	effectFuncTaekwonFoot
	effectFuncHitLine
	effectFuncTurnUndead
	effectFuncRevive
	effectFuncGroundWalls
	effectFuncMageProjectile
	effectFuncFrostDiverHit
	effectFuncMageBolt
	effectFuncSight
	effectFuncGroundSpikes
	effectFuncWaterBall
	effectFuncJupitel
	effectFuncSightTrasher
	effectFuncFirePillar
	effectFuncGroundCast
	effectFuncSoulDrain
	effectFuncGravitation
	effectFuncHPConversion
	effectFuncSoulChangeLink
	effectFuncSoulBreaker
	effectFuncNinjaWind
	effectFuncNinjaPetals
	effectFuncNativeSprite
)

func (m *WorldMode) drawFuncEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, effect worldEffect, component worldEffectComponent, componentIndex int, worldX, worldY, worldZ, progress float64, now time.Time) {
	switch component.funcAdapter {
	case effectFuncHPConversion:
		m.drawHPConversion(screen, ctx, projection, component, effect, now)
	case effectFuncSoulBreaker:
		m.drawSoulBreaker(screen, ctx, component, effect, now)
	case effectFuncSoulChangeLink:
		m.drawSoulChangeLink(screen, ctx, component, effect, now)
	case effectFuncSoulDrain:
		m.drawSoulDrain(screen, ctx, projection, component, effect, now)
	case effectFuncGravitation:
		m.drawGravitation(screen, ctx, component, effect, modelPoint3{x: worldX, y: worldZ, z: worldY}, now)
	case effectFuncGroundSpikes:
		m.drawGroundSpikes(screen, ctx, component, effect, modelPoint3{x: worldX, y: worldZ, z: worldY}, now)
	case effectFuncWaterBall:
		m.drawWizardWaterBall(screen, ctx, projection, component, effect, now)
	case effectFuncJupitel:
		m.drawJupitel(screen, ctx, projection, component, effect, modelPoint3{x: worldX, y: worldZ, z: worldY}, now)
	case effectFuncSightTrasher:
		m.drawSightTrasher(screen, ctx, projection, component, effect, modelPoint3{x: worldX, y: worldZ, z: worldY}, now)
	case effectFuncFirePillar:
		m.drawFirePillar(screen, ctx, component, effect, modelPoint3{x: worldX, y: worldZ, z: worldY}, now)
	case effectFuncMageProjectile:
		m.drawMageProjectile(screen, ctx, projection, component, effect, now)
	case effectFuncFrostDiverHit:
		m.drawFrostDiverSpikes(screen, ctx, component, effect, modelPoint3{x: worldX, y: worldZ, z: worldY}, now)
	case effectFuncMageBolt:
		m.drawMageBolt(screen, ctx, projection, component, effect, modelPoint3{x: worldX, y: worldZ, z: worldY}, now)
	case effectFuncSight:
		m.drawSightEffect(screen, ctx, projection, component, effect, modelPoint3{x: worldX, y: worldZ, z: worldY}, now)
	case effectFuncTurnUndead:
		m.drawTurnUndeadEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncRevive:
		m.drawReviveEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncGroundWalls:
		m.drawGroundWallsEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncMonkRays:
		m.drawMonkRays(screen, ctx, component, effect, now)
	case effectFuncSongMusic:
		m.drawSongMusic(screen, ctx, projection, component, effect, modelPoint3{x: worldX, y: worldZ, z: worldY}, now)
	case effectFuncSongVertical, effectFuncSongLight, effectFuncSongOut:
		m.drawSongGround(screen, ctx, projection, component, effect, modelPoint3{x: worldX, y: worldZ, z: worldY}, now)
	case effectFuncDirectedSprite:
		m.drawDirectedSpriteEffect(screen, ctx, projection, component, effect, now)
	case effectFuncClownGypsy:
		m.drawClownGypsyEffect(screen, ctx, projection, component, effect, now)
	case effectFuncBlacksmith:
		m.drawBlacksmithEffect(screen, ctx, projection, component, effect, now)
	case effectFuncChemicalProtection:
		m.drawChemicalProtection(screen, ctx, projection, component, effect, now)
	case effectFuncBerserkPotion:
		m.drawBerserkPotion(screen, ctx, component, effect, now)
	case effectFuncTwilight:
		m.drawTwilight(screen, ctx, projection, component, effect, now)
	case effectFuncCartTermination:
		m.drawCartTermination(screen, ctx, projection, component, effect, now)
	case effectFuncSniper:
		m.drawSniperEffect(screen, ctx, component, effect, now)
	case effectFuncGlyphs:
		m.drawEffectGlyphs(screen, ctx, projection, component, effect, now)
	case effectFuncStarRings:
		m.drawStarRings(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncStarBlind:
		m.drawStarBlind(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncTaekwonStorm:
		m.drawTaekwonStorm(screen, ctx, component, effect, now)
	case effectFuncTaekwonStop:
		m.drawTaekwonStop(screen, ctx, component, effect, now)
	case effectFuncChemicalDash:
		m.drawChemicalDash(screen, ctx, component, effect, now)
	case effectFuncTaekwonFoot:
		m.drawTaekwonFoot(screen, ctx, component, effect, now)
	case effectFuncTaekwonJump:
		m.drawTaekwonJump(screen, ctx, projection, component, effect, now)
	case effectFuncChainCrush:
		m.drawChainCrush(screen, ctx, component, effect, now)
	case effectFuncTripleAttack:
		m.drawTripleAttack(screen, ctx, component, effect, now)
	case effectFuncHitLine:
		m.drawHitLine(screen, ctx, component, effect, now)
	case effectFuncAgility:
		m.drawAgilityEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncSpiralPierce:
		m.drawSpiralPierceEffect(screen, ctx, component, effect, now)
	case effectFuncEquipmentBurst:
		m.drawEquipmentBurst(screen, ctx, projection, component, effect, now)
	case effectFuncGuard:
		m.drawGuardEffect(screen, ctx, projection, component, effect, now)
	case effectFuncGospelCross:
		m.drawGospelCrossEffect(screen, ctx, projection, component, effect, now)
	case effectFuncShieldBoomerang:
		m.drawShieldBoomerangEffect(screen, ctx, projection, component, effect, now)
	case effectFuncShieldChain:
		m.drawShieldChainEffect(screen, ctx, projection, component, effect, now)
	case effectFuncPressure:
		m.drawPressureEffect(screen, ctx, projection, component, effect, now)
	case effectFuncBashRays:
		m.drawBashRays(screen, ctx, projection, component, effect, now)
	case effectFuncGrandCross:
		m.drawGrandCrossEffect(screen, ctx, projection, component, effect, now)
	case effectFuncCrusaderDefense:
		m.drawCrusaderDefenseEffect(screen, ctx, projection, component, effect, now)
	case effectFuncProvidence:
		m.drawProvidenceEffect(screen, ctx, projection, component, effect, now)
	case effectFuncDevotionLink:
		m.drawDevotionLinkEffect(screen, ctx, projection, component, effect, now)

	case effectFuncBowlingBash:
		m.drawBowlingBashEffect(screen, ctx, component, effect, now)
	case effectFuncSpearBoomerang:
		m.drawSpearBoomerangEffect(screen, ctx, projection, component, effect, now)
	case effectFuncBashHit:
		m.drawBashHitEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncBash:
		m.drawBashEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncEndure:
		m.drawEndureEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncMagnumBreak:
		m.drawMagnumBreakEffect(screen, ctx, component, effect, now)
	case effectFuncHunter:
		m.drawHunterEffect(screen, ctx, projection, component, effect, modelPoint3{x: worldX, y: worldZ, z: worldY}, now)
	case effectFuncArrowShower:
		m.drawArrowShower(screen, ctx, projection, component, effect, now)
	case effectFuncMammoniteCoins:
		m.drawMammoniteCoins(screen, ctx, projection, component, effect, now)
	case effectFuncCartRevolution:
		m.drawCartRevolutionEffect(screen, ctx, component, effect, now)
	case effectFuncRogueBurst:
		m.drawRogueBurst(screen, ctx, component, effect, now)
	case effectFuncImpactParticles:
		m.drawImpactParticles(screen, ctx, projection, component, effect, now)
	case effectFuncSlimPitcher:
		m.drawSlimPitcher(screen, ctx, projection, component, effect, now)
	case effectFuncAcidDemon:
		m.drawAcidDemon(screen, ctx, component, effect, now)
	case effectFuncThrowItem:
		m.drawThrowItemEffect(screen, ctx, projection, component, effect, now)
	case effectFuncSonicBlow:
		m.drawSonicBlowEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncGrimtooth:
		m.drawGrimtoothEffect(screen, ctx, component, effect, now)
	case effectFuncRisingParticles:
		m.drawRisingParticlesEffect(screen, ctx, projection, component, effect, now)
	case effectFuncSteal:
		m.drawStealEffect(screen, ctx, projection, component, effect, now)
	case effectFuncSprinkleSand:
		m.drawSprinkleSandEffect(screen, ctx, projection, component, effect, now)
	case effectFuncGroundSample:
		m.drawGroundPlaneEffect(screen, ctx, component, effect, worldX, worldY, progress, now)
	case effectFuncCastRing:
		m.drawCastRingEffect(screen, ctx, component, effect, componentIndex, worldX, worldY, worldZ, progress)
	case effectFuncLockOnTarget:
		m.drawLockOnTargetEffect(screen, ctx, component, effect, worldX, worldY, worldZ, progress, now)
	case effectFuncLevel99Aura:
		m.drawLevel99AuraEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncLevel99Ground:
		m.drawLevel99GroundEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncLevel99Bubble:
		m.drawLevel99BubbleEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncPropertyGround:
		m.drawPropertyGroundEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncGroundCast:
		m.drawGroundCastEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncGroundSigil:
		m.drawGroundSigilEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncSpiritSphere:
		m.drawSpiritSphereEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncGumgang:
		m.drawGumgangEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncFlatColorTile:
		m.drawFlatColorTileEffect(screen, component, worldX, worldY, worldZ)
	case effectFuncGroundTexture:
		m.drawGroundTextureEffect(screen, ctx, component, effect, componentIndex, worldX, worldY, worldZ, progress, now)
	case effectFuncMapPillar:
		m.drawMapPillarEffect(screen, ctx, component, worldX, worldY, worldZ, progress)
	case effectFuncHitRing:
		m.drawHitRingEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncHitParticles:
		m.drawHitParticlesEffect(screen, ctx, projection, component, effect, componentIndex, worldX, worldY, worldZ, now)
	case effectFuncBlessingCircle:
		m.drawBlessingCircleEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncSoulLink:
		m.drawSoulLinkEffect(screen, ctx, projection, component, effect, now)
	case effectFuncKaupeDodge:
		m.drawSoulLights(screen, ctx, projection, component, effect, now)
	case effectFuncBlueCasting:
		m.drawBlueCastingEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncEstin:
		m.drawEstinEffect(screen, ctx, component, effect, now)
	case effectFuncRisingBuffParticles:
		m.drawRisingBuffParticles(screen, ctx, projection, component, effect, now)
	case effectFuncNinjaWind:
		m.drawNinjaWind(screen, ctx, component, effect, now)
	case effectFuncNinjaPetals:
		m.drawNinjaPetals(screen, ctx, projection, component, effect, now)
	case effectFuncNativeSprite:
		m.drawNativeSpriteEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncEsma:
		m.drawEsmaEffect(screen, ctx, component, effect, now)
	case effectFuncKaizel:
		m.drawKaizelEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncEsmaHitTexture:
		m.drawEsmaHitTexture(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncHitLineBurst:
		m.drawHitLineBurst(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	default:
	}
}

func effectFuncAdapterForName(name string) effectFuncAdapter {
	switch name {
	case "SoulDrain":
		return effectFuncSoulDrain
	case "HPConversion":
		return effectFuncHPConversion
	case "SoulChangeLink":
		return effectFuncSoulChangeLink
	case "SoulBreaker":
		return effectFuncSoulBreaker
	case "Gravitation":
		return effectFuncGravitation
	case "ChemicalProtection":
		return effectFuncChemicalProtection
	case "BerserkPotion":
		return effectFuncBerserkPotion
	case "Twilight":
		return effectFuncTwilight
	case "CartTermination":
		return effectFuncCartTermination
	case "HasteUp", "Overthrust":
		return effectFuncBlacksmith
	case "SageGroundCast":
		return effectFuncGroundCast
	case "WizardSpikes":
		return effectFuncGroundSpikes
	case "WaterBall":
		return effectFuncWaterBall
	case "Jupitel":
		return effectFuncJupitel
	case "SightTrasher":
		return effectFuncSightTrasher
	case "FirePillar":
		return effectFuncFirePillar
	case "MageProjectile":
		return effectFuncMageProjectile
	case "FrostDiverHit":
		return effectFuncFrostDiverHit
	case "MageBolt":
		return effectFuncMageBolt
	case "Sight":
		return effectFuncSight
	case "TurnUndead":
		return effectFuncTurnUndead
	case "Revive":
		return effectFuncRevive
	case "PriestGround":
		return effectFuncGroundWalls
	case "Agility":
		return effectFuncAgility
	case "FullStrip":
		return effectFuncEquipmentBurst
	case "Guard":
		return effectFuncGuard
	case "GospelCross":
		return effectFuncGospelCross
	case "DevotionLink":
		return effectFuncDevotionLink
	case "Providence":
		return effectFuncProvidence
	case "CrusaderDefense":
		return effectFuncCrusaderDefense
	case "GrandCross":
		return effectFuncGrandCross
	case "Sacrifice":
		return effectFuncBashRays
	case "Pressure":
		return effectFuncPressure
	case "ShieldChain":
		return effectFuncShieldChain
	case "ShieldBoomerang":
		return effectFuncShieldBoomerang
	case "BowlingBash":
		return effectFuncBowlingBash
	case "SpearBoomerang":
		return effectFuncSpearBoomerang
	case "BashHit":
		return effectFuncBashHit
	case "Bash":
		return effectFuncBash
	case "Endure":
		return effectFuncEndure
	case "MagnumBreak":
		return effectFuncMagnumBreak
	case "MagicTarget":
		return effectFuncGroundSample
	case "CastRing":
		return effectFuncCastRing
	case "LockOnTarget":
		return effectFuncLockOnTarget
	case "Level99Aura":
		return effectFuncLevel99Aura
	case "GroundAura":
		return effectFuncLevel99Ground
	case "Level99Bubble":
		return effectFuncLevel99Bubble
	case "NinjaWind":
		return effectFuncNinjaWind
	case "NinjaPetals":
		return effectFuncNinjaPetals
	case "NativeSprite":
		return effectFuncNativeSprite
	case "PropertyGround":
		return effectFuncPropertyGround
	case "LandProtectorGround":
		return effectFuncGroundSigil
	case "SpiritSphere":
		return effectFuncSpiritSphere
	case "Gumgang":
		return effectFuncGumgang
	case "FlatColorTile":
		return effectFuncFlatColorTile
	case "GroundTexture", "SpiderWeb":
		return effectFuncGroundTexture
	case "EffectBodyColor", "MagicCrasherBodyColor", "TransBlueBody", "LightBlade":
		return effectFuncBodyColor
	case "MapPillar":
		return effectFuncMapPillar
	case "HitRing":
		return effectFuncHitRing
	case "HitParticles":
		return effectFuncHitParticles
	case "BlessingCircle":
		return effectFuncBlessingCircle
	case "SoulLink":
		return effectFuncSoulLink
	case "KaupeDodge":
		return effectFuncKaupeDodge
	case "RogueBurst":
		return effectFuncRogueBurst
	case "ImpactParticles":
		return effectFuncImpactParticles
	case "MonkRays":
		return effectFuncMonkRays
	case "SongMusic":
		return effectFuncSongMusic
	case "SongVertical":
		return effectFuncSongVertical
	case "SongLight":
		return effectFuncSongLight
	case "SongOut":
		return effectFuncSongOut
	case "DirectedSprite":
		return effectFuncDirectedSprite
	case "ArrowVulcan", "Moonlit", "Marionette", "Hermode", "TarotCard":
		return effectFuncClownGypsy
	case "TrueSight", "WindWalk", "SharpShooting":
		return effectFuncSniper
	case "AsuraGlyphs":
		return effectFuncGlyphs
	case "StarRings":
		return effectFuncStarRings
	case "StarBlind":
		return effectFuncStarBlind
	case "TaekwonStorm":
		return effectFuncTaekwonStorm
	case "TaekwonStop":
		return effectFuncTaekwonStop
	case "TaekwonDash":
		return effectFuncChemicalDash
	case "TaekwonFoot":
		return effectFuncTaekwonFoot
	case "TaekwonJump":
		return effectFuncTaekwonJump
	case "ChainCrush":
		return effectFuncChainCrush
	case "TripleAttack":
		return effectFuncTripleAttack
	case "HitLine":
		return effectFuncHitLine
	case "Tanji":
		return effectFuncShieldBoomerang
	case "SonicBlow":
		return effectFuncSonicBlow
	case "SlimPitcher":
		return effectFuncSlimPitcher
	case "AcidDemon":
		return effectFuncAcidDemon
	case "ThrowItem":
		return effectFuncThrowItem
	case "Grimtooth":
		return effectFuncGrimtooth
	case "RisingParticles":
		return effectFuncRisingParticles
	case "Steal":
		return effectFuncSteal
	case "Detecting", "RemoveTrap", "Flasher", "BlitzBeat":
		return effectFuncHunter
	case "ArrowShower":
		return effectFuncArrowShower
	case "MammoniteCoins":
		return effectFuncMammoniteCoins
	case "CartRevolution":
		return effectFuncCartRevolution
	case "SprinkleSand":
		return effectFuncSprinkleSand
	case "BlueCasting":
		return effectFuncBlueCasting
	case "SpiralPierce":
		return effectFuncSpiralPierce
	case "Estin":
		return effectFuncEstin
	case "Estun", "SoulLinkerParticles":
		return effectFuncRisingBuffParticles
	case "Esma":
		return effectFuncEsma
	case "Kaizel":
		return effectFuncKaizel
	case "EsmaHitTexture":
		return effectFuncEsmaHitTexture
	case "SoulLinkerHitLines":
		return effectFuncHitLineBurst
	default:
		return effectFuncUnknown
	}
}
