package game

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

type skillUnitModel struct {
	unitID           uint16
	x                int
	y                int
	modelPath        string
	modelFallbacks   []string
	triggerEffectID  int
	scale            float64
	fixedFrame       int
	hasFixedFrame    bool
	animationStarted time.Time
}

func skillUnitModelSpec(unitID uint16) (db.SkillUnitModelSpec, bool) {
	spec, ok := db.SkillUnitModels[unitID]
	return spec, ok
}

func skillUnitModelFromEntry(entry network.SkillUnitEntry, spec db.SkillUnitModelSpec) skillUnitModel {
	return skillUnitModel{
		unitID:          entry.UnitID,
		x:               int(entry.X),
		y:               int(entry.Y),
		modelPath:       spec.ModelPath,
		modelFallbacks:  append([]string(nil), spec.FallbackModelPaths...),
		triggerEffectID: spec.TriggerEffectID,
		scale:           spec.Scale,
		fixedFrame:      spec.FixedFrame,
		hasFixedFrame:   spec.HasFixedFrame,
	}
}

func (m *WorldMode) applySkillUnitModelEntry(entry network.SkillUnitEntry) bool {
	spec, ok := skillUnitModelSpec(entry.UnitID)
	if !ok {
		return false
	}
	unit := skillUnitModelFromEntry(entry, spec)
	if entry.Visible {
		if m.skillUnitModels == nil {
			m.skillUnitModels = make(map[uint32]skillUnitModel)
		}
		m.skillUnitModels[entry.ID] = unit
		if m.hiddenSkillUnits != nil {
			delete(m.hiddenSkillUnits, entry.ID)
		}
		glog.Debugf("skill unit model unit=%d id=%d creator=%d cell=%d,%d model=%s", entry.UnitID, entry.ID, entry.CreatorID, entry.X, entry.Y, unit.modelPath)
		return true
	}
	if m.hiddenSkillUnits == nil {
		m.hiddenSkillUnits = make(map[uint32]skillUnitModel)
	}
	m.hiddenSkillUnits[entry.ID] = unit
	if m.skillUnitModels != nil {
		delete(m.skillUnitModels, entry.ID)
	}
	glog.Debugf("hidden skill unit model unit=%d id=%d creator=%d cell=%d,%d model=%s", entry.UnitID, entry.ID, entry.CreatorID, entry.X, entry.Y, unit.modelPath)
	return true
}

func (m *WorldMode) applySkillUnitUpdate(update network.SkillUnitUpdate) {
	if update.ID == 0 || m.hiddenSkillUnits == nil {
		return
	}
	unit, ok := m.hiddenSkillUnits[update.ID]
	if !ok {
		return
	}
	delete(m.hiddenSkillUnits, update.ID)
	if m.skillUnitModels == nil {
		m.skillUnitModels = make(map[uint32]skillUnitModel)
	}
	m.skillUnitModels[update.ID] = unit
	glog.Debugf("hidden skill unit model revealed id=%d unit=%d cell=%d,%d model=%s", update.ID, unit.unitID, unit.x, unit.y, unit.modelPath)
}

func (m *WorldMode) applyUsedTrapLookChange(ctx client.Context, look network.ActorLookChange) bool {
	if look.Type != 0 || look.ID == 0 || (look.Value != uint32(db.SkillUnitUsedTraps) && look.Value != 141) {
		return false
	}
	if look.Value == 141 {
		unit, ok := m.skillUnitModels[look.ID]
		if !ok {
			unit, ok = m.hiddenSkillUnits[look.ID]
		}
		if !ok || unit.unitID != 147 { // Land Mine becomes an active Fire Pillar.
			return false
		}
	}
	unit, ok := m.removeSkillUnitModel(look.ID)
	if !ok {
		return false
	}
	m.removeSkillUnitEffects(look.ID)
	if look.Value == 141 {
		unit.triggerEffectID = effectFirePillarBomb
	}
	if unit.triggerEffectID > 0 {
		now := time.Now()
		if m.addWorldEffectAtCellLifetime(ctx, unit.triggerEffectID, 0, unit.x, unit.y, now, 0, false) {
			glog.Debugf("skill unit trap triggered id=%d unit=%d cell=%d,%d effect=%d", look.ID, unit.unitID, unit.x, unit.y, unit.triggerEffectID)
		}
	}
	return true
}

func (m *WorldMode) removeSkillUnitModel(id uint32) (skillUnitModel, bool) {
	if m.skillUnitModels != nil {
		if unit, ok := m.skillUnitModels[id]; ok {
			delete(m.skillUnitModels, id)
			return unit, true
		}
	}
	if m.hiddenSkillUnits != nil {
		if unit, ok := m.hiddenSkillUnits[id]; ok {
			delete(m.hiddenSkillUnits, id)
			return unit, true
		}
	}
	return skillUnitModel{}, false
}

func (m *WorldMode) removeSkillUnitModelOnly(id uint32) bool {
	removed := false
	if m.skillUnitModels != nil {
		if _, ok := m.skillUnitModels[id]; ok {
			delete(m.skillUnitModels, id)
			removed = true
		}
	}
	if m.hiddenSkillUnits != nil {
		if _, ok := m.hiddenSkillUnits[id]; ok {
			delete(m.hiddenSkillUnits, id)
			removed = true
		}
	}
	return removed
}

func (m *WorldMode) skillUnitModelCell(id uint32) (int, int, bool) {
	if m.skillUnitModels != nil {
		if unit, ok := m.skillUnitModels[id]; ok {
			return unit.x, unit.y, true
		}
	}
	if m.hiddenSkillUnits != nil {
		if unit, ok := m.hiddenSkillUnits[id]; ok {
			return unit.x, unit.y, true
		}
	}
	return 0, 0, false
}

func (m *WorldMode) clickedSkillTarget(ctx client.Context, projection sceneProjection, skill session.Skill, mouseX, mouseY int, now time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	if skill.Type&skillTargetTrap == 0 {
		return clickedSkillTarget(ctx, projection, skill, mouseX, mouseY, now, m.actorDeaths)
	}
	var best worldstate.Actor
	bestDistance := math.Inf(1)
	for id, unit := range m.skillUnitModels {
		x, y := float64(unit.x), float64(unit.y)
		point, ok := projection.projectPoint(cellCenter(x), cellCenter(y), terrainHeightAt(ctx.World, x, y))
		if !ok {
			continue
		}
		dx, dy := float64(point.x)-float64(mouseX), float64(point.y)-float64(mouseY)
		distance := dx*dx + dy*dy
		if distance <= 20*20 && (distance < bestDistance || distance == bestDistance && id < best.ID) {
			best, bestDistance = worldstate.Actor{ID: id, X: unit.x, Y: unit.y}, distance
		}
	}
	return best, best.ID != 0
}

func (m *WorldMode) applyTrapModelLookChange(look network.ActorLookChange) bool {
	if look.Type != 0 || look.Value != 145 {
		return false
	}
	unit, ok := m.skillUnitModels[look.ID]
	if !ok || unit.unitID != 145 {
		return false
	}
	// ZC_SKILL_UPDATE reveals Ankle Snare; the following look change closes it.
	unit.animationStarted = time.Now()
	m.skillUnitModels[look.ID] = unit
	return true
}

func (m *WorldMode) drawSkillUnitRSMModels(screen *render.Frame, ctx client.Context, projection sceneProjection, now time.Time) {
	if screen == nil || ctx.World == nil || ctx.World.GND == nil || ctx.Resources == nil || len(m.skillUnitModels) == 0 {
		return
	}
	radius := rsmRenderRadius() * 2
	for id, unit := range m.skillUnitModels {
		worldX := cellCenter(float64(unit.x))
		worldY := cellCenter(float64(unit.y))
		if math.Abs(worldX-projection.playerX) > radius || math.Abs(worldY-projection.playerY) > radius {
			continue
		}
		rsm := m.runtimeRSMModel(ctx.Resources, unit.modelPath)
		if rsm == nil {
			rsm, unit.modelPath = m.runtimeSkillUnitFallbackRSMModel(ctx.Resources, unit)
		}
		if rsm == nil {
			continue
		}
		placement := skillUnitRSMPlacement(ctx.World, id, unit)
		frame, _ := skillUnitRSMFrame(rsm, placement.model, unit, now)
		m.drawAnimatedRSMPlacement(screen, ctx.Resources, ctx.World.RSW, rsm, placement, frame)
	}
}

func (m *WorldMode) runtimeRSMModel(manager *res.Manager, modelPath string) *res.RSM {
	if manager == nil || modelPath == "" {
		return nil
	}
	if m.runtimeRSMModels == nil {
		m.runtimeRSMModels = make(map[string]*res.RSM)
	}
	if rsm, ok := m.runtimeRSMModels[modelPath]; ok {
		return rsm
	}
	rsm, err := loadRSMModel(manager, modelPath)
	if err != nil {
		glog.Warnf("runtime rsm model unavailable model=%s: %v", modelPath, err)
		m.runtimeRSMModels[modelPath] = nil
		return nil
	}
	m.runtimeRSMModels[modelPath] = rsm
	return rsm
}

func (m *WorldMode) runtimeSkillUnitFallbackRSMModel(manager *res.Manager, unit skillUnitModel) (*res.RSM, string) {
	for _, modelPath := range unit.modelFallbacks {
		rsm := m.runtimeRSMModel(manager, modelPath)
		if rsm != nil {
			return rsm, modelPath
		}
	}
	return nil, ""
}

func skillUnitRSMPlacement(world *worldstate.World, id uint32, unit skillUnitModel) visibleRSMPlacement {
	baseX := cellCenter(float64(unit.x))
	baseY := cellCenter(float64(unit.y))
	scale := unit.scale
	if scale == 0 {
		scale = 1
	}
	return visibleRSMPlacement{
		index: int(id),
		model: res.RSWModel{
			Name:     "skill_unit",
			Filename: unit.modelPath,
			Position: res.RSWVector3{
				Y: float32(terrainHeightAt(world, float64(unit.x), float64(unit.y))),
			},
			Scale: res.RSWVector3{X: float32(scale), Y: float32(scale), Z: float32(scale)},
		},
		baseX: baseX,
		baseY: baseY,
	}
}

func skillUnitRSMFrame(rsm *res.RSM, placement res.RSWModel, unit skillUnitModel, now time.Time) (int, bool) {
	if !unit.animationStarted.IsZero() {
		last := max(3, 3+(rsmAnimationLength(rsm)-4)/100*100)
		frame := 3 + max(0, int(now.Sub(unit.animationStarted)/db.EffectFrameDuration))*100
		return min(frame, last), frame < last
	}
	if unit.hasFixedFrame {
		return unit.fixedFrame, false
	}
	return rsmAnimationFrame(rsm, placement, now)
}
