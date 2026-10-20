package game

import (
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/world"
)

func swordmanContext() client.Context {
	w := world.New()
	w.Player = world.Actor{ID: 100, X: 10, Y: 20, Job: db.JobSwordman}
	w.Actors[300] = world.Actor{ID: 300, X: 16, Y: 20, Job: 1002}
	return client.Context{World: w, Resources: &res.Manager{}, Session: &session.Session{AccountID: 2000000, CharID: 100}}
}

func TestMagnumBreakOncePerCast(t *testing.T) {
	for _, targets := range []int{0, 1, 5} {
		ctx := swordmanContext()
		m := &WorldMode{}
		now := time.Now()
		for i := 0; i < targets; i++ {
			action := network.ActorActionNotify{SkillID: db.SkillSMMagnum, SourceID: 2000000, TargetID: uint32(300 + i), Damage: int32(i * 50)}
			m.addSkillBeginEffect(ctx, action, now)
			m.addSkillEffect(ctx, action, now, now)
			m.addSkillHitEffect(ctx, action, now)
		}
		if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatal("Magnum Break victim notifications replayed the cast effect")
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillSMMagnum, SourceID: 2000000, TargetID: 2000000, Result: 1})
		if len(m.worldEffects) != 2 || len(m.scheduledSounds) != 1 {
			t.Fatalf("%d victims: effects=%+v sounds=%+v", targets, m.worldEffects, m.scheduledSounds)
		}
		burst := m.worldEffects[0]
		if burst.effectID != effectMagnumBreak || burst.actorID != 2000000 || burst.actorOrigin.x != 10.5 || !burst.hasActorTransform {
			t.Fatalf("burst must capture the caster's position: %+v", burst)
		}
		if m.worldEffects[1].effectID != effectQuakeMagnum || m.cameraShakeEnd.Sub(m.cameraShakeStart) != 50*time.Millisecond {
			t.Fatal("missing native Magnum Break camera shake")
		}
	}
}

func TestSwordmanSuccessEffects(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		effect int
		actor  uint32
		delays []time.Duration
	}{
		{db.SkillSMProvoke, effectProvoke, 300, []time.Duration{5 * db.EffectFrameDuration, 30 * db.EffectFrameDuration}},
		{db.SkillSMEndure, effectEndure, 2000000, []time.Duration{0}},
	} {
		ctx := swordmanContext()
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatalf("failed skill %d emitted effects", tc.skill)
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 1 || m.worldEffects[0].actorID != tc.actor || m.worldEffects[0].effectID != tc.effect {
			t.Fatalf("skill %d has wrong effect or owner: %+v", tc.skill, m.worldEffects)
		}
		if len(m.scheduledSounds) != len(tc.delays) {
			t.Fatalf("skill %d sounds: %+v", tc.skill, m.scheduledSounds)
		}
		for i, delay := range tc.delays {
			if !m.scheduledSounds[i].at.Equal(m.worldEffects[0].starts.Add(delay)) {
				t.Fatalf("skill %d sound %d has wrong delay", tc.skill, i)
			}
		}
	}
	// Auto Berserk only changes server status; it must not play Provoke on cast.
	m := &WorldMode{}
	m.applySkillNoDamageNotify(swordmanContext(), network.SkillNoDamageNotify{SkillID: db.SkillSMAutoberserk, SourceID: 2000000, TargetID: 2000000, Result: 1})
	if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
		t.Fatal("Auto Berserk invented a cast effect")
	}
}

func TestBashImpactOnlyOnHit(t *testing.T) {
	ctx := swordmanContext()
	starts := time.Unix(100, 0)
	for _, damage := range []int32{0, 50} {
		m := &WorldMode{}
		action := network.ActorActionNotify{SkillID: db.SkillSMBash, SourceID: 2000000, TargetID: 300, Damage: damage}
		m.addSkillBeginEffect(ctx, action, starts)
		m.addSkillHitEffect(ctx, action, starts.Add(200*time.Millisecond))
		want := 1
		if damage > 0 {
			want = 2
		}
		if len(m.worldEffects) != want || m.worldEffects[0].effectID != effectBashBegin || m.worldEffects[0].actorID != 2000000 {
			t.Fatalf("damage %d: effects=%+v", damage, m.worldEffects)
		}
		if damage > 0 && (m.worldEffects[1].actorID != 300 || m.worldEffects[1].effectID != effectBashHit || !m.worldEffects[1].starts.Equal(starts.Add(200*time.Millisecond))) {
			t.Fatal("Bash impact did not follow the victim/hit time")
		}
	}
}

func TestSwordmanNativeTimings(t *testing.T) {
	for _, tc := range []struct {
		frame, life, rise, fade int
		peak                    float64
		want                    uint8
	}{
		{0, 40, 6, 30, 170, 28}, {5, 40, 6, 30, 170, 170}, {30, 40, 6, 30, 170, 153}, {39, 40, 6, 30, 170, 0},
		{0, 80, 20, 54, 254, 12}, {19, 80, 20, 54, 254, 254}, {79, 80, 20, 54, 254, 0},
		{0, 30, 15, 15, 254, 16}, {14, 30, 15, 15, 254, 254}, {29, 30, 15, 15, 254, 0},
	} {
		if got := swordmanAlpha(tc.frame, tc.life, tc.rise, tc.fade, tc.peak); got != tc.want {
			t.Fatalf("phase %+v: alpha %d", tc, got)
		}
	}
	effect := worldEffect{starts: time.Unix(100, 0), actorID: 300}
	for i := 0; i < 8; i++ {
		first, later := bashHitParticle(effect, i, 0), bashHitParticle(effect, i, 7)
		if first.alpha != 31 || later.alpha != 254 || later.halfWidth >= first.halfWidth || later.halfHeight <= first.halfHeight {
			t.Fatalf("Bash impact must brighten, narrow and stretch: %+v -> %+v", first, later)
		}
		if p := bashHitParticle(effect, i, 29); p.alpha != 0 {
			t.Fatal("Bash impact outlived its native maximum")
		}
	}
	for birth := 0; birth < 50; birth++ {
		first, later := endureStreak(effect, birth, birth), endureStreak(effect, birth, birth+19)
		if birth <= 40 {
			if first.alpha != 12 || later.alpha != 254 || math.Hypot(later.x, later.y+20) >= math.Hypot(first.x, first.y+20) {
				t.Fatalf("Endure streak must brighten and travel inward: %+v -> %+v", first, later)
			}
		} else if first.alpha != 0 || later.alpha != 0 {
			t.Fatal("Endure emitted a streak after the native parent fade cutoff")
		}
		if endureStreak(effect, birth, birth-1).alpha != 0 || endureStreak(effect, birth, birth+39).alpha != 0 {
			t.Fatal("Endure streak visible outside its lifetime")
		}
	}
}

func TestSwordmanRenderingAndMovement(t *testing.T) {
	ctx := swordmanContext()
	starts := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 10, 20, 0)
	for _, id := range []int{effectBashBegin, effectBashHit, effectEndure, effectMagnumBreak} {
		spec, _ := worldEffectSpecForID(id)
		component := spec.components[0]
		m := &WorldMode{textures: map[string]*render.Image{}}
		for _, path := range append([]string{component.textureFile}, component.textureFiles...) {
			m.textures["__effectfile_"+path+"|blackkey=true"] = render.NewImage(8, 8)
		}
		ctx.World.Player.X = 10
		m.addWorldEffectAt(ctx, id, 2000000, starts)
		ctx.World.Player.X = 12
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, starts.Add(5*db.EffectFrameDuration), false)
		value := reflect.ValueOf(frame).Elem()
		commands, overlays := value.FieldByName("worldCommands"), value.FieldByName("commands")
		if commands.Len()+overlays.Len() == 0 {
			t.Fatalf("effect %d rendered nothing", id)
		}
		if value.FieldByName("worldBillboards").Len() != 0 {
			t.Fatalf("effect %d unexpectedly queued a world billboard", id)
		}
		if (id == effectMagnumBreak && overlays.Len() != 0) || (id != effectMagnumBreak && commands.Len() != 0) {
			t.Fatalf("effect %d uses the wrong render pass", id)
		}
		for _, draws := range []reflect.Value{commands, overlays} {
			for i := 0; i < draws.Len(); i++ {
				options := draws.Index(i).FieldByName("Options")
				if options.FieldByName("Blend").Int() != int64(render.BlendSourceOver) || options.FieldByName("DepthWrite").Bool() {
					t.Fatalf("effect %d changed native blending or writes depth", id)
				}
				if options.FieldByName("DepthTest").Bool() != (id == effectMagnumBreak && i == 0) {
					t.Fatalf("effect %d wrong depth behavior", id)
				}
			}
		}
		if id == effectMagnumBreak {
			if commands.Len() != 3 {
				t.Fatal("Magnum Break must draw a sphere and two rings")
			}
			verts := commands.Index(1).FieldByName("Vertices")
			minX, maxX := math.Inf(1), math.Inf(-1)
			for i := 0; i < verts.Len(); i++ {
				x := verts.Index(i).FieldByName("X").Float()
				minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			}
			if math.Abs((minX+maxX)/2-10.5) > .001 {
				t.Fatal("Magnum Break followed the caster instead of staying at the cast position")
			}
		} else if id == effectBashBegin {
			for _, zoom := range []float64{75, 125, 200} {
				projection := newSceneProjectionForTargetYawPitchZoom(800, 600, 10, 20, 0, 45, 50, zoom)
				frame := render.NewFrame(800, 600)
				m.drawWorldEffects(frame, ctx, projection, starts.Add(5*db.EffectFrameDuration), false)
				verts := reflect.ValueOf(frame).Elem().FieldByName("commands").Index(0).FieldByName("Vertices")
				center := verts.Index(0)
				cx, cy := center.FieldByName("DstX").Float(), center.FieldByName("DstY").Float()
				want := projection.Project(12.5, 20.5, .07)
				if math.Abs(cx-float64(want.x)) > .001 || math.Abs(cy-float64(want.y-40)) > .001 {
					t.Fatal("Bash must follow the caster, forty pixels above its feet")
				}
				for i := 0; i < verts.Len(); i++ {
					vertex := verts.Index(i)
					radius := math.Hypot(vertex.FieldByName("DstX").Float()-cx, vertex.FieldByName("DstY").Float()-cy)
					if i%3 != 0 && math.Abs(radius-100) > .001 {
						t.Fatal("Bash's screen-space radius changed with camera zoom")
					}
				}
			}
		}
		expired := render.NewFrame(800, 600)
		m.drawWorldEffects(expired, ctx, projection, starts.Add(spec.duration+time.Second), false)
		value = reflect.ValueOf(expired).Elem()
		if value.FieldByName("commands").Len()+value.FieldByName("worldCommands").Len()+value.FieldByName("worldBillboards").Len() != 0 {
			t.Fatalf("effect %d rendered after its lifetime", id)
		}
	}
}

func TestMagnumSphereGeometry(t *testing.T) {
	texture := render.NewImage(16, 16)
	origin := modelPoint3{x: 10, y: 2, z: 20}
	vertices, indices := magnumSphereMesh(texture, origin, 14, 254)
	if len(vertices) != 200 || len(indices) != 300 {
		t.Fatal("Magnum Break must use the native ten by five sphere mesh")
	}
	// Fifteen updates: v += -1.15/60, r += v; five native units per cell.
	const radius = 2.99
	below, above := false, false
	for _, vertex := range vertices {
		p := modelPoint3{x: float64(vertex.X), y: float64(vertex.Y), z: float64(vertex.Z)}
		offset := sub3(p, origin)
		if math.Abs(math.Sqrt(dot3(offset, offset))-radius) > .00001 {
			t.Fatalf("sphere radius at peak opacity = %+v", offset)
		}
		below, above = below || p.y < origin.y, above || p.y > origin.y
	}
	if !below || !above {
		t.Fatal("Magnum Break must be a sphere intersecting the floor, not a cylinder")
	}
	for _, index := range indices {
		if int(index) >= len(vertices) {
			t.Fatal("out-of-bounds sphere index")
		}
	}
}

func TestSwordmanRealAssets(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to check real effect assets")
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, archive := range manager.Archives {
		t.Cleanup(func() { _ = archive.Close() })
	}
	m := &WorldMode{}
	for _, id := range []int{effectBashBegin, effectBashHit, effectEndure, effectMagnumBreak, effectProvoke} {
		spec, _ := worldEffectSpecForID(id)
		for _, c := range spec.components {
			for _, path := range append([]string{c.textureFile}, c.textureFiles...) {
				if path != "" && m.effectFileTexture(manager, path) == nil {
					t.Errorf("effect %d missing texture %s", id, path)
				}
			}
			if c.strFile != "" && m.loadWorldEffectSTR(manager, c.strFile, c.texturePath) == nil {
				t.Errorf("effect %d missing STR %s", id, c.strFile)
			}
		}
		for _, sound := range spec.sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Error(err)
			}
		}
	}
}
