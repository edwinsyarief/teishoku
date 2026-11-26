package teishoku

import (
	"fmt"
	"testing"
)

// Test constants
const (
	TestCap = 1024
)

// Test components
type Position struct {
	X, Y float32
}

type Velocity struct {
	DX, DY float32
}

type Health struct {
	HP int
}

type WithPointer struct {
	Data *int
}

type RichPointer struct {
	Ptr   *int
	Label string
	Bytes []byte
}

type Dummy1 struct {
	Val float32
}

type Dummy2 struct {
	Val float32
}

type Dummy3 struct {
	Val int64
}

type Dummy4 struct {
	Val uint32
}

type Dummy5 struct {
	Val int16
}

type Tag struct{}

func assertPanics(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("%s: expected panic, got none", name)
		}
	}()
	fn()
}

// World Creation and Basic Lifecycle Tests

func TestNewWorld(t *testing.T) {
	t.Run("DefaultCapacity", func(t *testing.T) {
		w := NewWorld()
		if w == nil {
			t.Fatal("NewWorld returned nil")
		}
		if w.Len() != 0 {
			t.Errorf("expected Len 0, got %d", w.Len())
		}
		if cap(w.metas) < defaultWorldCapacity+1 {
			t.Errorf("expected metas cap >= %d, got %d", defaultWorldCapacity+1, cap(w.metas))
		}
		if len(w.metas) != 1 {
			t.Errorf("expected reserved sentinel slot len 1, got %d", len(w.metas))
		}
		if len(w.archetypes) != 1 || w.emptyArch == nil {
			t.Errorf("expected 1 empty archetype, got %d", len(w.archetypes))
		}
		if w.Resources() == nil {
			t.Error("expected non-nil Resources")
		}
	})

	t.Run("CustomCapacity", func(t *testing.T) {
		w := NewWorld(TestCap)
		if w.Len() != 0 {
			t.Errorf("expected Len 0, got %d", w.Len())
		}
		if cap(w.metas) < TestCap+1 {
			t.Errorf("expected metas cap >= %d, got %d", TestCap+1, cap(w.metas))
		}
		if len(w.archetypes) != 1 {
			t.Errorf("expected 1 archetype, got %d", len(w.archetypes))
		}
	})

	t.Run("CustomEntityAndArchCapacity", func(t *testing.T) {
		const entityCap = 128
		const archCap = 16
		w := NewWorld(entityCap, archCap)
		if w.emptyArch.capacity != archCap {
			t.Errorf("expected emptyArch.capacity %d, got %d", archCap, w.emptyArch.capacity)
		}
		if cap(w.metas) < entityCap+1 {
			t.Errorf("expected cap(metas) >= %d, got %d", entityCap+1, cap(w.metas))
		}
		if cap(w.freeIDs) < entityCap {
			t.Errorf("expected cap(freeIDs) >= %d, got %d", entityCap, cap(w.freeIDs))
		}
		b := NewBuilder[Position](w)
		if b.arch.capacity != archCap {
			t.Errorf("expected new archetype capacity %d, got %d", archCap, b.arch.capacity)
		}

		wZeroArch := NewWorld(64, 0)
		if wZeroArch.emptyArch.capacity != 0 {
			t.Errorf("expected emptyArch.capacity 0, got %d", wZeroArch.emptyArch.capacity)
		}
		if cap(wZeroArch.metas) < 65 {
			t.Errorf("expected cap(metas) >= 65, got %d", cap(wZeroArch.metas))
		}
		if cap(wZeroArch.freeIDs) < 64 {
			t.Errorf("expected cap(freeIDs) >= 64, got %d", cap(wZeroArch.freeIDs))
		}
	})
}

func TestWorldCreateRemoveClearIsValid(t *testing.T) {
	w := NewWorld(16)

	// Zero Entity{} is always invalid.
	if w.IsValid(Entity{}) {
		t.Error("zero Entity{} should never be valid")
	}
	w.RemoveEntity(Entity{}) // safe no-op

	// Out-of-bounds ID is invalid.
	oob := Entity{ID: 9999, Version: 1}
	if w.IsValid(oob) {
		t.Error("out-of-bounds entity should be invalid")
	}
	w.RemoveEntity(oob) // safe no-op

	// CreateEntity and RemoveEntity
	e1 := w.CreateEntity()
	if !w.IsValid(e1) {
		t.Error("newly created entity should be valid")
	}
	if w.Len() != 1 {
		t.Errorf("expected Len 1, got %d", w.Len())
	}

	w.RemoveEntity(e1)
	if w.IsValid(e1) {
		t.Error("removed entity should be invalid")
	}
	if w.Len() != 0 {
		t.Errorf("expected Len 0, got %d", w.Len())
	}
	w.RemoveEntity(e1) // double remove is safe no-op

	// Recycle ID checks stale Version rejection.
	e2 := w.CreateEntity()
	if e2.ID != e1.ID {
		t.Errorf("expected recycled ID %d, got %d", e1.ID, e2.ID)
	}
	if e2.Version == e1.Version {
		t.Errorf("expected incremented version, got %d", e2.Version)
	}
	if !w.IsValid(e2) {
		t.Error("recycled entity should be valid")
	}
	if w.IsValid(e1) {
		t.Error("stale entity version must remain invalid after ID recycle")
	}

	// CreateEntities count <= 0 no-op
	w.CreateEntities(0)
	w.CreateEntities(-5)
	if w.Len() != 1 {
		t.Errorf("expected Len 1 after non-positive CreateEntities, got %d", w.Len())
	}

	// CreateEntities positive count
	w.CreateEntities(5)
	if w.Len() != 6 {
		t.Errorf("expected Len 6, got %d", w.Len())
	}

	// RemoveEntities batch including invalid, stale, duplicate, and valid entities.
	allEnts := make([]Entity, len(w.emptyArch.entityIDs[:w.emptyArch.size]))
	copy(allEnts, w.emptyArch.entityIDs[:w.emptyArch.size])
	toRemove := []Entity{
		Entity{},
		oob,
		e1, // stale
		allEnts[0],
		allEnts[0], // duplicate
		allEnts[2],
		allEnts[4],
	}
	w.RemoveEntities(toRemove)
	if w.Len() != 3 {
		t.Errorf("expected Len 3 after RemoveEntities, got %d", w.Len())
	}
	if w.IsValid(allEnts[0]) || w.IsValid(allEnts[2]) || w.IsValid(allEnts[4]) {
		t.Error("removed entities should be invalid")
	}
	if !w.IsValid(allEnts[1]) || !w.IsValid(allEnts[3]) || !w.IsValid(allEnts[5]) {
		t.Error("remaining entities should still be valid")
	}

	// ClearEntities across multiple archetypes keeps archetypes allocated and clears all entities.
	bPos := NewBuilder[Position](w)
	pEnt := bPos.NewEntityWith(Position{X: 3, Y: 4})
	if w.Len() != 4 {
		t.Errorf("expected Len 4, got %d", w.Len())
	}
	archCountBefore := len(w.archetypes)
	w.ClearEntities()
	if w.Len() != 0 {
		t.Errorf("expected Len 0 after ClearEntities, got %d", w.Len())
	}
	if w.IsValid(pEnt) || w.IsValid(allEnts[1]) {
		t.Error("all entities should be invalid after ClearEntities")
	}
	if len(w.archetypes) != archCountBefore {
		t.Errorf("expected archetype count preserved (%d), got %d", archCountBefore, len(w.archetypes))
	}
	for _, a := range w.archetypes {
		if a.size != 0 {
			t.Errorf("expected archetype size 0 after ClearEntities, got %d", a.size)
		}
	}

	// Creating entities after ClearEntities works and bumps version.
	e3 := bPos.NewEntityWith(Position{X: 9, Y: 10})
	if !w.IsValid(e3) || w.Len() != 1 {
		t.Error("failed to create entity after ClearEntities")
	}
	if w.IsValid(pEnt) {
		t.Error("previous pEnt handle must remain invalid")
	}
	if got := bPos.Get(e3); got == nil || *got != (Position{X: 9, Y: 10}) {
		t.Errorf("unexpected position after ClearEntities: %+v", got)
	}
}

func TestComponentID(t *testing.T) {
	w := NewWorld()
	idPos1 := ComponentID[Position](w)
	idVel := ComponentID[Velocity](w)
	idPos2 := ComponentID[Position](w)
	idHealth := ComponentID[Health](w)

	if idPos1 != idPos2 {
		t.Errorf("expected reused ComponentID for Position: %d != %d", idPos1, idPos2)
	}
	if idPos1 == idVel || idPos1 == idHealth || idVel == idHealth {
		t.Errorf("expected distinct ComponentIDs, got %d, %d, %d", idPos1, idVel, idHealth)
	}
	if w.nextCompID != 3 {
		t.Errorf("expected nextCompID 3, got %d", w.nextCompID)
	}

	// Independent across worlds
	w2 := NewWorld()
	if got := ComponentID[Health](w2); got != 0 {
		t.Errorf("expected independent ComponentID 0 in fresh world, got %d", got)
	}

	// MaxComponentTypes limit panic
	assertPanics(t, "ComponentID MaxComponentTypes", func() {
		w.nextCompID = MaxComponentTypes
		ComponentID[Dummy1](w)
	})
}

func TestGetOrCreateArchetype(t *testing.T) {
	w := NewWorld()
	idPos := ComponentID[Position](w)
	var mask bitmask256
	mask.set(idPos)
	a1 := w.getOrCreateArchetype(mask, []uint8{idPos})
	if a1 == nil {
		t.Fatal("archetype not created")
	}
	if len(w.archetypes) != 2 {
		t.Errorf("expected 2 archetypes, got %d", len(w.archetypes))
	}
	a2 := w.getOrCreateArchetype(mask, []uint8{idPos})
	if a1 != a2 {
		t.Error("expected same archetype pointer for identical mask")
	}
}

func TestAutoExpand(t *testing.T) {
	initialCap := 8
	w := NewWorld(initialCap)
	builder := NewBuilder[Position](w)

	// Create 50 via NewEntityWith (exceeds initialCap=8 and defaultArchCapacity=32)
	ents := make([]Entity, 0, 100)
	for i := 0; i < 50; i++ {
		e := builder.NewEntityWith(Position{X: float32(i), Y: float32(i * 2)})
		if !w.IsValid(e) {
			t.Fatalf("entity %d invalid immediately after creation", i)
		}
		ents = append(ents, e)
	}

	// Create 50 more via NewEntitiesWith
	builder.NewEntitiesWith(50, Position{X: 777, Y: 888})

	if w.Len() != 100 {
		t.Errorf("expected Len 100, got %d", w.Len())
	}
	if cap(w.metas) < 101 {
		t.Errorf("expected metas cap >= 101, got %d", cap(w.metas))
	}
	if builder.arch.capacity < 100 {
		t.Errorf("expected archetype capacity >= 100, got %d", builder.arch.capacity)
	}

	// Verify integrity of first 50 entities after multiple expansions
	for i, e := range ents {
		if !w.IsValid(e) {
			t.Errorf("entity %d became invalid after expand", i)
		}
		got := builder.Get(e)
		want := Position{X: float32(i), Y: float32(i * 2)}
		if got == nil || *got != want {
			t.Errorf("entity %d: expected %+v, got %+v", i, want, got)
		}
	}

	// Verify batch-created entities
	for i := 50; i < 100; i++ {
		e := builder.arch.entityIDs[i]
		got := builder.Get(e)
		want := Position{X: 777, Y: 888}
		if got == nil || *got != want {
			t.Errorf("batch entity %d: expected %+v, got %+v", i, want, got)
		}
	}
}

// Builder Tests (arities 1, 2, 3, 6, 8)

func TestBuilder1(t *testing.T) {
	w := NewWorld()
	b := NewBuilder[Position](w)
	bHealth := NewBuilder[Health](w)

	// NewEntity & NewEntityWith
	e1 := b.NewEntity()
	e2 := b.NewEntityWith(Position{X: 1, Y: 2})
	if !w.IsValid(e1) || !w.IsValid(e2) {
		t.Fatal("created entities should be valid")
	}
	if !b.Has(e1) || !b.Has(e2) {
		t.Error("builder.Has should be true")
	}
	if p := b.Get(e2); p == nil || *p != (Position{X: 1, Y: 2}) {
		t.Errorf("unexpected Get on e2: %+v", p)
	}

	// NewEntities & NewEntitiesWith including count <= 0
	b.NewEntities(0)
	b.NewEntities(-3)
	b.NewEntitiesWith(0, Position{X: 9, Y: 9})
	b.NewEntitiesWith(-2, Position{X: 9, Y: 9})
	if w.Len() != 2 {
		t.Errorf("expected Len 2 after non-positive batch calls, got %d", w.Len())
	}

	b.NewEntities(3)
	b.NewEntitiesWith(4, Position{X: 5, Y: 6})
	if w.Len() != 9 {
		t.Errorf("expected Len 9, got %d", w.Len())
	}

	// Invalid / missing checks
	if b.Has(Entity{}) || b.Get(Entity{}) != nil {
		t.Error("zero entity should fail Has/Get")
	}
	emptyEnt := w.CreateEntity()
	if b.Has(emptyEnt) || b.Get(emptyEnt) != nil {
		t.Error("empty entity should fail Has/Get for Position")
	}

	// Set: migration from empty + migration from Health + in-place update
	b.Set(emptyEnt, Position{X: 10, Y: 20})
	if p := b.Get(emptyEnt); p == nil || *p != (Position{X: 10, Y: 20}) {
		t.Errorf("Set migration from empty failed: %+v", p)
	}
	b.Set(emptyEnt, Position{X: 11, Y: 22})
	if p := b.Get(emptyEnt); p == nil || *p != (Position{X: 11, Y: 22}) {
		t.Errorf("Set in-place update failed: %+v", p)
	}

	hEnt := bHealth.NewEntityWith(Health{HP: 50})
	b.Set(hEnt, Position{X: 30, Y: 40})
	if !b.Has(hEnt) || !bHealth.Has(hEnt) {
		t.Error("Set migration should preserve Health")
	}
	if *bHealth.Get(hEnt) != (Health{HP: 50}) {
		t.Error("Health value altered during Position Set migration")
	}

	// SetBatch
	eEmpty2 := w.CreateEntity()
	b.SetBatch([]Entity{}, Position{X: 1, Y: 1})
	b.SetBatch([]Entity{e1, eEmpty2, Entity{}}, Position{X: 42, Y: 43})
	if *b.Get(e1) != (Position{X: 42, Y: 43}) || *b.Get(eEmpty2) != (Position{X: 42, Y: 43}) {
		t.Error("SetBatch failed to update/migrate entities")
	}

	// Remove & RemoveBatch
	b.Remove(hEnt)
	if b.Has(hEnt) || b.Get(hEnt) != nil {
		t.Error("Position should be removed from hEnt")
	}
	if !bHealth.Has(hEnt) || *bHealth.Get(hEnt) != (Health{HP: 50}) {
		t.Error("Health should remain on hEnt after Position Remove")
	}
	b.Remove(hEnt)     // no-op when absent
	b.Remove(Entity{}) // no-op when invalid

	b.RemoveBatch([]Entity{e1, eEmpty2, Entity{}})
	if b.Has(e1) || b.Has(eEmpty2) {
		t.Error("RemoveBatch failed to remove Position")
	}
}

func TestBuilder2(t *testing.T) {
	w := NewWorld()
	b2 := NewBuilder2[Position, Velocity](w)
	bPos := NewBuilder[Position](w)
	bHealth := NewBuilder[Health](w)

	e1 := b2.NewEntity()
	if !b2.Has(e1) {
		t.Error("NewEntity should have both components")
	}
	e2 := b2.NewEntityWith(Position{X: 1, Y: 2}, Velocity{DX: 3, DY: 4})
	p, v := b2.Get(e2)
	if p == nil || v == nil || *p != (Position{X: 1, Y: 2}) || *v != (Velocity{DX: 3, DY: 4}) {
		t.Errorf("unexpected Get on e2: %+v, %+v", p, v)
	}

	b2.NewEntities(0)
	b2.NewEntitiesWith(-1, Position{}, Velocity{})
	b2.NewEntities(2)
	b2.NewEntitiesWith(3, Position{X: 7, Y: 8}, Velocity{DX: 9, DY: 10})
	if w.Len() != 7 {
		t.Errorf("expected Len 7, got %d", w.Len())
	}

	// Partial Has/Get returns false/nil,nil
	posOnly := bPos.NewEntityWith(Position{X: 5, Y: 5})
	if b2.Has(posOnly) {
		t.Error("Has should be false when only 1 of 2 components present")
	}
	gp, gv := b2.Get(posOnly)
	if gp != nil || gv != nil {
		t.Error("Get should return nil, nil when missing a component")
	}

	// Set: in-place, from empty, from partial (preserving Health)
	b2.Set(e2, Position{X: 11, Y: 12}, Velocity{DX: 13, DY: 14})
	p, v = b2.Get(e2)
	if *p != (Position{X: 11, Y: 12}) || *v != (Velocity{DX: 13, DY: 14}) {
		t.Error("Builder2.Set in-place failed")
	}

	emptyEnt := w.CreateEntity()
	b2.Set(emptyEnt, Position{X: 21, Y: 22}, Velocity{DX: 23, DY: 24})
	if !b2.Has(emptyEnt) {
		t.Error("Builder2.Set migration from empty failed")
	}

	partialEnt := bPos.NewEntityWith(Position{X: 1, Y: 1})
	bHealth.Set(partialEnt, Health{HP: 99})
	b2.Set(partialEnt, Position{X: 31, Y: 32}, Velocity{DX: 33, DY: 34})
	p, v = b2.Get(partialEnt)
	if p == nil || v == nil || *p != (Position{X: 31, Y: 32}) || *v != (Velocity{DX: 33, DY: 34}) {
		t.Error("Builder2.Set migration from partial failed")
	}
	if *bHealth.Get(partialEnt) != (Health{HP: 99}) {
		t.Error("Health lost during Builder2.Set migration")
	}

	// SetBatch
	eEmpty2 := w.CreateEntity()
	b2.SetBatch([]Entity{e1, eEmpty2, Entity{}}, Position{X: 50, Y: 51}, Velocity{DX: 52, DY: 53})
	p, v = b2.Get(eEmpty2)
	if p == nil || v == nil || *p != (Position{X: 50, Y: 51}) || *v != (Velocity{DX: 52, DY: 53}) {
		t.Error("Builder2.SetBatch failed")
	}

	// Remove preserves Health and removes Position+Velocity; also works when only 1 of 2 present
	b2.Remove(partialEnt)
	if b2.Has(partialEnt) || bPos.Has(partialEnt) {
		t.Error("Builder2.Remove failed to remove both components")
	}
	if !bHealth.Has(partialEnt) || *bHealth.Get(partialEnt) != (Health{HP: 99}) {
		t.Error("Health should be preserved after Builder2.Remove")
	}
	b2.Remove(posOnly)
	if bPos.Has(posOnly) {
		t.Error("Builder2.Remove should remove present subset of its components")
	}

	b2.RemoveBatch([]Entity{e1, e2, Entity{}})
	if b2.Has(e1) || b2.Has(e2) {
		t.Error("Builder2.RemoveBatch failed")
	}
}

func TestBuilder3_6_8(t *testing.T) {
	t.Run("Builder3", func(t *testing.T) {
		w := NewWorld()
		b3 := NewBuilder3[Position, Velocity, Health](w)
		bD1 := NewBuilder[Dummy1](w)

		e := b3.NewEntityWith(Position{X: 1, Y: 2}, Velocity{DX: 3, DY: 4}, Health{HP: 100})
		bD1.Set(e, Dummy1{Val: 9.5})
		p, v, h := b3.Get(e)
		if p == nil || v == nil || h == nil || *p != (Position{X: 1, Y: 2}) || *v != (Velocity{DX: 3, DY: 4}) || *h != (Health{HP: 100}) {
			t.Fatal("Builder3.NewEntityWith / Get mismatch")
		}

		b3.NewEntities(0)
		b3.NewEntities(2)
		b3.NewEntitiesWith(0, Position{}, Velocity{}, Health{})
		b3.NewEntitiesWith(-1, Position{}, Velocity{}, Health{})
		b3.NewEntitiesWith(3, Position{X: 5, Y: 6}, Velocity{DX: 7, DY: 8}, Health{HP: 50})
		if w.Len() != 6 {
			t.Errorf("expected Len 6, got %d", w.Len())
		}

		eEmpty := w.CreateEntity()
		b3.Set(eEmpty, Position{X: 2, Y: 3}, Velocity{DX: 4, DY: 5}, Health{HP: 80})
		if !b3.Has(eEmpty) {
			t.Error("Builder3.Set migration failed")
		}
		b3.SetBatch([]Entity{e, eEmpty, Entity{}}, Position{X: 9, Y: 10}, Velocity{DX: 11, DY: 12}, Health{HP: 77})
		for _, target := range []Entity{e, eEmpty} {
			p, v, h := b3.Get(target)
			if p == nil || *p != (Position{X: 9, Y: 10}) || *v != (Velocity{DX: 11, DY: 12}) || *h != (Health{HP: 77}) {
				t.Errorf("Builder3.SetBatch mismatch on %+v", target)
			}
		}

		b3.RemoveBatch([]Entity{e, eEmpty})
		if b3.Has(e) || b3.Has(eEmpty) {
			t.Error("Builder3.RemoveBatch failed")
		}
		if !bD1.Has(e) || *bD1.Get(e) != (Dummy1{Val: 9.5}) {
			t.Error("Dummy1 should be preserved after Builder3.RemoveBatch")
		}
	})

	t.Run("Builder6", func(t *testing.T) {
		w := NewWorld()
		b6 := NewBuilder6[Position, Velocity, Health, Dummy1, Dummy2, Dummy3](w)
		bD4 := NewBuilder[Dummy4](w)

		e := b6.NewEntityWith(
			Position{X: 1, Y: 2},
			Velocity{DX: 3, DY: 4},
			Health{HP: 60},
			Dummy1{Val: 1.5},
			Dummy2{Val: 2.5},
			Dummy3{Val: 300},
		)
		bD4.Set(e, Dummy4{Val: 400})

		if !b6.Has(e) {
			t.Fatal("Builder6.Has failed")
		}
		c1, c2, c3, c4, c5, c6 := b6.Get(e)
		if *c1 != (Position{X: 1, Y: 2}) || *c2 != (Velocity{DX: 3, DY: 4}) || *c3 != (Health{HP: 60}) ||
			*c4 != (Dummy1{Val: 1.5}) || *c5 != (Dummy2{Val: 2.5}) || *c6 != (Dummy3{Val: 300}) {
			t.Error("Builder6.Get values mismatch")
		}

		b6.NewEntities(2)
		b6.NewEntitiesWith(3, Position{X: 2}, Velocity{DX: 3}, Health{HP: 4}, Dummy1{Val: 5}, Dummy2{Val: 6}, Dummy3{Val: 7})
		if w.Len() != 6 {
			t.Errorf("expected Len 6, got %d", w.Len())
		}

		eEmpty := w.CreateEntity()
		b6.Set(eEmpty, Position{X: 8}, Velocity{DX: 9}, Health{HP: 10}, Dummy1{Val: 11}, Dummy2{Val: 12}, Dummy3{Val: 13})
		if !b6.Has(eEmpty) {
			t.Error("Builder6.Set migration failed")
		}
		b6.SetBatch([]Entity{e, eEmpty}, Position{X: 14}, Velocity{DX: 15}, Health{HP: 16}, Dummy1{Val: 17}, Dummy2{Val: 18}, Dummy3{Val: 19})
		c1, _, _, _, _, c6 = b6.Get(e)
		if c1.X != 14 || c6.Val != 19 {
			t.Error("Builder6.SetBatch failed")
		}

		b6.Remove(e)
		if b6.Has(e) || HasComponent[Position](w, e) || HasComponent[Dummy3](w, e) {
			t.Error("Builder6.Remove failed")
		}
		if *bD4.Get(e) != (Dummy4{Val: 400}) {
			t.Error("Dummy4 should be preserved after Builder6.Remove")
		}
		b6.RemoveBatch([]Entity{eEmpty})
		if b6.Has(eEmpty) {
			t.Error("Builder6.RemoveBatch failed")
		}
	})

	t.Run("Builder8", func(t *testing.T) {
		w := NewWorld()
		b8 := NewBuilder8[Position, Velocity, Health, Dummy1, Dummy2, Dummy3, Dummy4, Dummy5](w)
		bTag := NewBuilder[Tag](w)

		e := b8.NewEntityWith(
			Position{X: 1, Y: 2},
			Velocity{DX: 3, DY: 4},
			Health{HP: 5},
			Dummy1{Val: 6},
			Dummy2{Val: 7},
			Dummy3{Val: 8},
			Dummy4{Val: 9},
			Dummy5{Val: 10},
		)
		bTag.Set(e, Tag{})

		if !b8.Has(e) {
			t.Fatal("Builder8.Has failed")
		}
		c1, c2, c3, c4, c5, c6, c7, c8 := b8.Get(e)
		if c1.X != 1 || c2.DX != 3 || c3.HP != 5 || c4.Val != 6 || c5.Val != 7 || c6.Val != 8 || c7.Val != 9 || c8.Val != 10 {
			t.Error("Builder8.Get values mismatch")
		}

		b8.NewEntities(2)
		b8.NewEntitiesWith(2, Position{X: 2}, Velocity{DX: 2}, Health{HP: 2}, Dummy1{Val: 2}, Dummy2{Val: 2}, Dummy3{Val: 2}, Dummy4{Val: 2}, Dummy5{Val: 2})
		b8.NewEntitiesWith(0, Position{}, Velocity{}, Health{}, Dummy1{}, Dummy2{}, Dummy3{}, Dummy4{}, Dummy5{})
		if w.Len() != 5 {
			t.Errorf("expected Len 5, got %d", w.Len())
		}

		eEmpty := w.CreateEntity()
		b8.SetBatch([]Entity{e, eEmpty, Entity{}}, Position{X: 21}, Velocity{DX: 22}, Health{HP: 23}, Dummy1{Val: 24}, Dummy2{Val: 25}, Dummy3{Val: 26}, Dummy4{Val: 27}, Dummy5{Val: 28})
		c1, _, _, _, _, _, _, c8 = b8.Get(eEmpty)
		if c1 == nil || c1.X != 21 || c8.Val != 28 {
			t.Error("Builder8.SetBatch migration failed")
		}

		b8.Remove(e)
		if b8.Has(e) || !bTag.Has(e) {
			t.Error("Builder8.Remove should remove all 8 and preserve Tag")
		}
		b8.RemoveBatch([]Entity{eEmpty})
		if b8.Has(eEmpty) {
			t.Error("Builder8.RemoveBatch failed")
		}
	})
}

func TestBuilderDuplicatePanic(t *testing.T) {
	w := NewWorld()
	assertPanics(t, "NewBuilder2 duplicate", func() {
		NewBuilder2[Position, Position](w)
	})
	assertPanics(t, "NewBuilder3 duplicate", func() {
		NewBuilder3[Position, Velocity, Position](w)
	})
	assertPanics(t, "NewBuilder8 duplicate", func() {
		NewBuilder8[Position, Velocity, Health, Dummy1, Dummy2, Dummy3, Dummy4, Position](w)
	})
}

// Filter Tests (Filter0, Filter, Filter2, Filter3, Filter6, Filter8)

func TestFilter0(t *testing.T) {
	w := NewWorld()
	bPos := NewBuilder[Position](w)
	bPos.NewEntities(4)
	w.CreateEntities(3)
	// Also create and strip an entity so it migrates back to emptyArch
	stripped := bPos.NewEntityWith(Position{X: 1})
	bPos.Remove(stripped)

	f0 := NewFilter0(w)
	if f0.Count() != 4 {
		t.Errorf("expected Filter0 Count 4, got %d", f0.Count())
	}
	if len(f0.Entities()) != 4 {
		t.Errorf("expected Filter0 Entities len 4, got %d", len(f0.Entities()))
	}

	qCount := 0
	q := f0.Query()
	for q.Next() {
		if !w.IsValid(q.Entity()) || HasComponent[Position](w, q.Entity()) {
			t.Errorf("Filter0 yielded invalid or non-empty entity: %+v", q.Entity())
		}
		qCount++
	}
	if qCount != 4 {
		t.Errorf("expected Query0 count 4, got %d", qCount)
	}

	chunkTotal := 0
	chunks := f0.Chunks()
	for chunks.Next() {
		chunkTotal += len(chunks.Entities())
	}
	if chunkTotal != 4 {
		t.Errorf("expected Chunks0 total 4, got %d", chunkTotal)
	}

	eachTotal := 0
	f0.EachChunk(func(ents []Entity) {
		eachTotal += len(ents)
	})
	if eachTotal != 4 {
		t.Errorf("expected EachChunk total 4, got %d", eachTotal)
	}

	// Without chaining on Filter0
	Without[Position](f0).Without(ComponentID[Velocity](w))
	if f0.Count() != 4 {
		t.Errorf("expected Filter0 Count 4 with Without, got %d", f0.Count())
	}

	f0.RemoveEntities()
	if f0.Count() != 0 {
		t.Errorf("expected Filter0 Count 0 after RemoveEntities, got %d", f0.Count())
	}
	if w.Len() != 4 {
		t.Errorf("expected 4 entities with Position untouched, got %d", w.Len())
	}
}

func TestFilterQuerySingleAndMultiArchetype(t *testing.T) {
	t.Run("Filter1_SingleAndMulti", func(t *testing.T) {
		w := NewWorld()
		b1 := NewBuilder[Position](w)
		b2 := NewBuilder2[Position, Velocity](w)
		b3 := NewBuilder3[Position, Velocity, Health](w)
		bVel := NewBuilder[Velocity](w)

		e1 := b1.NewEntityWith(Position{X: 1, Y: 10})
		e2 := b2.NewEntityWith(Position{X: 2, Y: 20}, Velocity{DX: 1, DY: 2})
		e3 := b3.NewEntityWith(Position{X: 3, Y: 30}, Velocity{DX: 2, DY: 3}, Health{HP: 100})
		_ = bVel.NewEntityWith(Velocity{DX: 9, DY: 9}) // should not match

		f := NewFilter[Position](w)
		q := f.Query()
		seen := make(map[Entity]bool)
		for q.Next() {
			e := q.Entity()
			p := q.Get()
			p.X += 100
			p.Y += 200
			seen[e] = true
		}
		if len(seen) != 3 {
			t.Fatalf("expected 3 matched entities, got %d", len(seen))
		}
		for i, e := range []Entity{e1, e2, e3} {
			if !seen[e] {
				t.Errorf("entity %d missing from Query", i)
			}
			got := GetComponent[Position](w, e)
			want := Position{X: float32(i+1) + 100, Y: float32((i+1)*10) + 200}
			if got == nil || *got != want {
				t.Errorf("entity %d: expected %+v, got %+v", i, want, got)
			}
		}
	})

	t.Run("Filter2_MultiArchetype", func(t *testing.T) {
		w := NewWorld()
		b2 := NewBuilder2[Position, Velocity](w)
		b3 := NewBuilder3[Position, Velocity, Health](w)
		b1 := NewBuilder[Position](w)

		e1 := b2.NewEntityWith(Position{X: 1, Y: 2}, Velocity{DX: 10, DY: 20})
		e2 := b3.NewEntityWith(Position{X: 3, Y: 4}, Velocity{DX: 30, DY: 40}, Health{HP: 50})
		_ = b1.NewEntityWith(Position{X: 99, Y: 99})

		f2 := NewFilter2[Position, Velocity](w)
		q := f2.Query()
		count := 0
		for q.Next() {
			p, v := q.Get()
			p.X += v.DX
			p.Y += v.DY
			v.DX *= 2
			count++
		}
		if count != 2 {
			t.Fatalf("expected 2 entities, got %d", count)
		}
		if *GetComponent[Position](w, e1) != (Position{X: 11, Y: 22}) || GetComponent[Velocity](w, e1).DX != 20 {
			t.Error("e1 mutation failed")
		}
		if *GetComponent[Position](w, e2) != (Position{X: 33, Y: 44}) || GetComponent[Velocity](w, e2).DX != 60 {
			t.Error("e2 mutation failed")
		}
	})

	t.Run("Filter3_6_8", func(t *testing.T) {
		w := NewWorld()
		b3 := NewBuilder3[Position, Velocity, Health](w)
		b6 := NewBuilder6[Position, Velocity, Health, Dummy1, Dummy2, Dummy3](w)
		b8 := NewBuilder8[Position, Velocity, Health, Dummy1, Dummy2, Dummy3, Dummy4, Dummy5](w)

		e3 := b3.NewEntityWith(Position{X: 1}, Velocity{DX: 2}, Health{HP: 3})
		e6 := b6.NewEntityWith(Position{X: 10}, Velocity{DX: 20}, Health{HP: 30}, Dummy1{Val: 4}, Dummy2{Val: 5}, Dummy3{Val: 6})
		e8 := b8.NewEntityWith(Position{X: 100}, Velocity{DX: 200}, Health{HP: 300}, Dummy1{Val: 4}, Dummy2{Val: 5}, Dummy3{Val: 6}, Dummy4{Val: 7}, Dummy5{Val: 8})

		// Filter3 matches all 3 archetypes
		f3 := NewFilter3[Position, Velocity, Health](w)
		q3 := f3.Query()
		c3 := 0
		for q3.Next() {
			p, v, h := q3.Get()
			p.X += 1
			v.DX += 1
			h.HP += 1
			c3++
		}
		if c3 != 3 {
			t.Errorf("Filter3 expected 3, got %d", c3)
		}
		if GetComponent[Health](w, e3).HP != 4 || GetComponent[Health](w, e6).HP != 31 || GetComponent[Health](w, e8).HP != 301 {
			t.Error("Filter3 mutations failed")
		}

		// Filter6 matches e6 and e8
		f6 := NewFilter6[Position, Velocity, Health, Dummy1, Dummy2, Dummy3](w)
		q6 := f6.Query()
		c6 := 0
		for q6.Next() {
			p, v, h, d1, d2, d3 := q6.Get()
			p.Y = 55
			v.DY = 66
			h.HP += 10
			d1.Val += 1
			d2.Val += 2
			d3.Val += 3
			c6++
		}
		if c6 != 2 {
			t.Errorf("Filter6 expected 2, got %d", c6)
		}
		if GetComponent[Dummy3](w, e6).Val != 9 || GetComponent[Dummy3](w, e8).Val != 9 {
			t.Error("Filter6 mutations failed")
		}

		// Filter8 matches e8
		f8 := NewFilter8[Position, Velocity, Health, Dummy1, Dummy2, Dummy3, Dummy4, Dummy5](w)
		q8 := f8.Query()
		c8 := 0
		for q8.Next() {
			p, v, h, d1, d2, d3, d4, d5 := q8.Get()
			p.X += 5
			v.DX += 5
			h.HP += 5
			d1.Val += 5
			d2.Val += 5
			d3.Val += 5
			d4.Val += 5
			d5.Val += 5
			c8++
		}
		if c8 != 1 {
			t.Errorf("Filter8 expected 1, got %d", c8)
		}
		if GetComponent[Dummy4](w, e8).Val != 12 || GetComponent[Dummy5](w, e8).Val != 13 {
			t.Error("Filter8 mutations failed")
		}
	})
}

func TestFilterNestedQuery(t *testing.T) {
	w := NewWorld()
	b2 := NewBuilder2[Position, Velocity](w)
	b3 := NewBuilder3[Position, Velocity, Health](w)

	b2.NewEntityWith(Position{X: 1}, Velocity{DX: 1})
	b2.NewEntityWith(Position{X: 2}, Velocity{DX: 1})
	b3.NewEntityWith(Position{X: 3}, Velocity{DX: 1}, Health{HP: 10})

	f2 := NewFilter2[Position, Velocity](w)
	qOuter := f2.Query()
	pairs := 0
	var sum float32
	for qOuter.Next() {
		pOut, _ := qOuter.Get()
		qInner := f2.Query()
		for qInner.Next() {
			pIn, vIn := qInner.Get()
			sum += pOut.X*10 + pIn.X + vIn.DX
			pairs++
		}
	}
	if pairs != 9 {
		t.Errorf("expected 9 pairs from nested 3x3 queries, got %d", pairs)
	}
	// Outer sum(1+2+3)*10*3 = 180; Inner sum(1+2+3)*3 = 18; vIn.DX*9 = 9 => 207
	if sum != 207 {
		t.Errorf("expected nested sum 207, got %v", sum)
	}
}

func TestFilterChunksAndEachChunk(t *testing.T) {
	w := NewWorld()
	b2 := NewBuilder2[Position, Velocity](w)
	b3 := NewBuilder3[Position, Velocity, Health](w)

	e1 := b2.NewEntityWith(Position{X: 1, Y: 2}, Velocity{DX: 10, DY: 20})
	e2 := b2.NewEntityWith(Position{X: 3, Y: 4}, Velocity{DX: 30, DY: 40})
	e3 := b3.NewEntityWith(Position{X: 5, Y: 6}, Velocity{DX: 50, DY: 60}, Health{HP: 100})

	f2 := NewFilter2[Position, Velocity](w)

	// Chunks() across multiple archetypes
	chunks := f2.Chunks()
	chunkCount := 0
	totalEnts := 0
	for chunks.Next() {
		chunkCount++
		ents := chunks.Entities()
		pos, vel := chunks.Get()
		if len(ents) != len(pos) || len(pos) != len(vel) {
			t.Fatalf("chunk slice length mismatch: ents=%d pos=%d vel=%d", len(ents), len(pos), len(vel))
		}
		for i := range ents {
			if *GetComponent[Position](w, ents[i]) != pos[i] {
				t.Errorf("chunk entity %+v position alignment mismatch", ents[i])
			}
			pos[i].X += vel[i].DX
			pos[i].Y += vel[i].DY
			totalEnts++
		}
	}
	if chunkCount != 2 || totalEnts != 3 {
		t.Errorf("expected 2 chunks and 3 entities, got chunks=%d ents=%d", chunkCount, totalEnts)
	}
	if *GetComponent[Position](w, e1) != (Position{X: 11, Y: 22}) ||
		*GetComponent[Position](w, e2) != (Position{X: 33, Y: 44}) ||
		*GetComponent[Position](w, e3) != (Position{X: 55, Y: 66}) {
		t.Error("Chunks slice mutations did not persist")
	}

	// EachChunk() across multiple archetypes
	eachCalls := 0
	f2.EachChunk(func(pos []Position, vel []Velocity) {
		eachCalls++
		for i := range pos {
			pos[i].X += 100
			vel[i].DX += 1
		}
	})
	if eachCalls != 2 {
		t.Errorf("expected 2 EachChunk calls, got %d", eachCalls)
	}
	if GetComponent[Position](w, e1).X != 111 || GetComponent[Position](w, e3).X != 155 {
		t.Error("EachChunk slice mutations did not persist")
	}
	if GetComponent[Velocity](w, e2).DX != 31 {
		t.Error("EachChunk velocity mutation did not persist")
	}
}

func TestFilterWithout(t *testing.T) {
	w := NewWorld()
	b2 := NewBuilder2[Position, Velocity](w)
	b3 := NewBuilder3[Position, Velocity, Health](w)
	b4 := NewBuilder4[Position, Velocity, Health, Dummy1](w)

	eKeep := b2.NewEntityWith(Position{X: 1}, Velocity{DX: 1})
	eExHealth := b3.NewEntityWith(Position{X: 2}, Velocity{DX: 2}, Health{HP: 10})
	eExDummy := b4.NewEntityWith(Position{X: 3}, Velocity{DX: 3}, Health{HP: 20}, Dummy1{Val: 5})

	// Generic Without[T](filter)
	fGeneric := Without[Health](NewFilter2[Position, Velocity](w))
	if fGeneric.Count() != 1 {
		t.Errorf("Without[Health] expected Count 1, got %d", fGeneric.Count())
	}
	ents := fGeneric.Entities()
	if len(ents) != 1 || ents[0] != eKeep {
		t.Errorf("Without[Health] expected [%+v], got %+v", eKeep, ents)
	}

	// Method filter.Without(compIDs...)
	fMethod := NewFilter2[Position, Velocity](w).Without(ComponentID[Dummy1](w))
	if fMethod.Count() != 2 {
		t.Errorf("filter.Without(Dummy1) expected Count 2, got %d", fMethod.Count())
	}

	// New archetype created after Without must still be excluded!
	bWithTag := NewBuilder4[Position, Velocity, Health, Tag](w)
	eExLater := bWithTag.NewEntityWith(Position{X: 4}, Velocity{DX: 4}, Health{HP: 30}, Tag{})
	if fGeneric.Count() != 1 {
		t.Errorf("expected Count 1 after excluded archetype added, got %d", fGeneric.Count())
	}
	if fMethod.Count() != 3 {
		t.Errorf("expected Count 3 for fMethod after non-Dummy1 archetype added, got %d", fMethod.Count())
	}

	// RemoveEntities on excluded filter only removes non-excluded entities
	fGeneric.RemoveEntities()
	if w.IsValid(eKeep) {
		t.Error("eKeep should have been removed")
	}
	if !w.IsValid(eExHealth) || !w.IsValid(eExDummy) || !w.IsValid(eExLater) {
		t.Error("excluded entities must not be removed by RemoveEntities")
	}
}

func TestFilterCountEntitiesRemoveEntities(t *testing.T) {
	w := NewWorld()
	bPos := NewBuilder[Position](w)
	bPosVel := NewBuilder2[Position, Velocity](w)
	bHealth := NewBuilder[Health](w)

	bPos.NewEntities(3)
	bPosVel.NewEntities(2)
	hEnt := bHealth.NewEntityWith(Health{HP: 42})

	f := NewFilter[Position](w)
	if f.Count() != 5 {
		t.Fatalf("expected Count 5, got %d", f.Count())
	}
	ents1 := f.Entities()
	if len(ents1) != 5 {
		t.Fatalf("expected 5 entities, got %d", len(ents1))
	}
	// Cached Entities() returns same underlying slice while world hasn't mutated
	ents2 := f.Entities()
	if len(ents2) != 5 || &ents1[0] != &ents2[0] {
		t.Error("expected cached Entities() slice reuse when world unmutated")
	}

	// Mutating world invalidates cached Entities()
	bPos.NewEntity()
	if f.Count() != 6 || len(f.Entities()) != 6 {
		t.Errorf("expected 6 after adding entity, got Count=%d Entities=%d", f.Count(), len(f.Entities()))
	}

	f.RemoveEntities()
	if f.Count() != 0 || len(f.Entities()) != 0 {
		t.Errorf("expected 0 after RemoveEntities, got Count=%d Entities=%d", f.Count(), len(f.Entities()))
	}
	if !w.IsValid(hEnt) || w.Len() != 1 {
		t.Error("unrelated Health entity should remain intact")
	}
}

func TestFilterEmptyToNonEmptyRegression(t *testing.T) {
	w := NewWorld()
	b2 := NewBuilder2[Position, Velocity](w)

	// Create archetype (Position, Velocity) and immediately remove its only entity so size == 0.
	tmp := b2.NewEntity()
	w.RemoveEntity(tmp)
	if b2.arch.size != 0 {
		t.Fatalf("expected archetype size 0, got %d", b2.arch.size)
	}

	// Create Filter2 while the matching archetype exists with size == 0.
	f2 := NewFilter2[Position, Velocity](w)
	if f2.Count() != 0 || len(f2.Entities()) != 0 {
		t.Fatal("expected empty filter before entities added")
	}
	qEmpty := f2.Query()
	if qEmpty.Next() {
		t.Fatal("expected empty Query.Next() == false")
	}
	cEmpty := f2.Chunks()
	if cEmpty.Next() {
		t.Fatal("expected empty Chunks.Next() == false")
	}
	f2.EachChunk(func([]Position, []Velocity) {
		t.Fatal("EachChunk should not invoke callback for empty archetype")
	})

	// Add entities to the previously-empty archetype.
	e1 := b2.NewEntityWith(Position{X: 1, Y: 2}, Velocity{DX: 3, DY: 4})
	e2 := b2.NewEntityWith(Position{X: 5, Y: 6}, Velocity{DX: 7, DY: 8})

	if f2.Count() != 2 {
		t.Errorf("Count expected 2, got %d", f2.Count())
	}
	if ents := f2.Entities(); len(ents) != 2 || ents[0] != e1 || ents[1] != e2 {
		t.Errorf("Entities mismatch: %+v", ents)
	}

	qCount := 0
	q := f2.Query()
	for q.Next() {
		p, v := q.Get()
		p.X += v.DX
		qCount++
	}
	if qCount != 2 {
		t.Errorf("Query expected 2, got %d", qCount)
	}

	chunkSeen := 0
	chunks := f2.Chunks()
	for chunks.Next() {
		pos, _ := chunks.Get()
		chunkSeen += len(pos)
	}
	if chunkSeen != 2 {
		t.Errorf("Chunks expected 2, got %d", chunkSeen)
	}

	eachSeen := 0
	f2.EachChunk(func(pos []Position, vel []Velocity) {
		eachSeen += len(pos)
		if pos[0].X != 4 || pos[1].X != 12 {
			t.Errorf("unexpected values in EachChunk: %+v", pos)
		}
	})
	if eachSeen != 2 {
		t.Errorf("EachChunk expected 2, got %d", eachSeen)
	}
}

func TestFilterDuplicatePanic(t *testing.T) {
	w := NewWorld()
	assertPanics(t, "NewFilter2 duplicate", func() {
		NewFilter2[Position, Position](w)
	})
	assertPanics(t, "NewFilter3 duplicate", func() {
		NewFilter3[Position, Velocity, Position](w)
	})
	assertPanics(t, "NewFilter8 duplicate", func() {
		NewFilter8[Position, Velocity, Health, Dummy1, Dummy2, Dummy3, Dummy4, Velocity](w)
	})
}

// Standalone Functions Tests (functions.go)

func TestFunctions(t *testing.T) {
	w := NewWorld()
	e := w.CreateEntity()

	// Invalid / absent
	if HasComponent[Position](w, Entity{}) || GetComponent[Position](w, Entity{}) != nil {
		t.Error("zero entity should return false/nil")
	}
	if HasComponent[Position](w, e) || GetComponent[Position](w, e) != nil {
		t.Error("empty entity should not have Position")
	}
	SetComponent(w, Entity{}, Position{X: 1}) // safe no-op
	RemoveComponent[Position](w, Entity{})    // safe no-op
	RemoveComponent[Position](w, e)           // safe no-op when absent

	// SetComponent new -> migrate from empty
	SetComponent(w, e, Position{X: 10, Y: 20})
	if !HasComponent[Position](w, e) {
		t.Error("expected Position after SetComponent")
	}
	if p := GetComponent[Position](w, e); p == nil || *p != (Position{X: 10, Y: 20}) {
		t.Errorf("unexpected GetComponent: %+v", p)
	}

	// SetComponent existing -> in-place update
	SetComponent(w, e, Position{X: 30, Y: 40})
	if p := GetComponent[Position](w, e); p == nil || *p != (Position{X: 30, Y: 40}) {
		t.Errorf("unexpected GetComponent after update: %+v", p)
	}

	// Add second component + zero-sized tag component
	SetComponent(w, e, Velocity{DX: 5, DY: 6})
	SetComponent(w, e, Tag{})
	if !HasComponent[Position](w, e) || !HasComponent[Velocity](w, e) || !HasComponent[Tag](w, e) {
		t.Error("expected Position, Velocity, and Tag")
	}

	// RemoveComponent preserves remaining components
	RemoveComponent[Position](w, e)
	if HasComponent[Position](w, e) || GetComponent[Position](w, e) != nil {
		t.Error("Position should be removed")
	}
	if *GetComponent[Velocity](w, e) != (Velocity{DX: 5, DY: 6}) || !HasComponent[Tag](w, e) {
		t.Error("Velocity and Tag should remain after RemoveComponent[Position]")
	}
}

// Swapper Tests (swap.go)

func TestSwapper(t *testing.T) {
	w := NewWorld()

	assertPanics(t, "NewSwapper duplicate", func() {
		NewSwapper[Position, Position](w)
	})

	swapper := NewSwapper[Velocity, Health](w)
	b2 := NewBuilder2[Position, Velocity](w)
	b3 := NewBuilder3[Position, Velocity, Dummy1](w)
	bPos := NewBuilder[Position](w)

	e1 := b2.NewEntityWith(Position{X: 1, Y: 2}, Velocity{DX: 3, DY: 4})
	e2 := b2.NewEntityWith(Position{X: 5, Y: 6}, Velocity{DX: 7, DY: 8})
	e3 := b3.NewEntityWith(Position{X: 9, Y: 10}, Velocity{DX: 11, DY: 12}, Dummy1{Val: 99})
	eNoVel := bPos.NewEntityWith(Position{X: 100, Y: 200})

	// No-op on entity lacking Old component or invalid entity
	swapper.Swap(eNoVel, Health{HP: 1})
	if HasComponent[Health](w, eNoVel) || *GetComponent[Position](w, eNoVel) != (Position{X: 100, Y: 200}) {
		t.Error("Swap should be no-op when entity lacks Old component")
	}
	swapper.Swap(Entity{}, Health{HP: 1})
	swapper.Swap(Entity{ID: 9999, Version: 1}, Health{HP: 1})

	// Single Swap preserving Position
	swapper.Swap(e1, Health{HP: 75})
	if HasComponent[Velocity](w, e1) {
		t.Error("Velocity should be removed after Swap")
	}
	if h := GetComponent[Health](w, e1); h == nil || *h != (Health{HP: 75}) {
		t.Errorf("Health mismatch after Swap: %+v", h)
	}
	if p := GetComponent[Position](w, e1); p == nil || *p != (Position{X: 1, Y: 2}) {
		t.Errorf("Position not preserved after Swap: %+v", p)
	}

	// SwapBatch across cached-edge (e2), second archetype (e3), lacking Old (eNoVel), and invalid
	swapper.SwapBatch([]Entity{e2, e3, eNoVel, Entity{}}, Health{HP: 150})
	if HasComponent[Velocity](w, e2) || *GetComponent[Health](w, e2) != (Health{HP: 150}) || *GetComponent[Position](w, e2) != (Position{X: 5, Y: 6}) {
		t.Error("SwapBatch failed on e2")
	}
	if HasComponent[Velocity](w, e3) || *GetComponent[Health](w, e3) != (Health{HP: 150}) ||
		*GetComponent[Position](w, e3) != (Position{X: 9, Y: 10}) || *GetComponent[Dummy1](w, e3) != (Dummy1{Val: 99}) {
		t.Error("SwapBatch failed on e3")
	}
	if HasComponent[Health](w, eNoVel) {
		t.Error("SwapBatch should not touch entity lacking Old component")
	}
}

// Pointer-Bearing Component & GC Safety Tests

func TestPointerComponentGCSafety(t *testing.T) {
	w := NewWorld()
	bRich := NewBuilder[RichPointer](w)
	bPos := NewBuilder[Position](w)

	mkVal := func(n int) RichPointer {
		v := n
		return RichPointer{
			Ptr:   &v,
			Label: fmt.Sprintf("item-%d", n),
			Bytes: []byte{byte(n), byte(n + 1)},
		}
	}
	verifyVal := func(t *testing.T, got RichPointer, n int) {
		t.Helper()
		if got.Ptr == nil || *got.Ptr != n {
			t.Errorf("Ptr mismatch for %d: got %v", n, got.Ptr)
		}
		if got.Label != fmt.Sprintf("item-%d", n) {
			t.Errorf("Label mismatch for %d: got %q", n, got.Label)
		}
		if len(got.Bytes) != 2 || got.Bytes[0] != byte(n) || got.Bytes[1] != byte(n+1) {
			t.Errorf("Bytes mismatch for %d: got %v", n, got.Bytes)
		}
	}
	assertSlotZeroed := func(t *testing.T, a *archetype, compID uint8, slot int) {
		t.Helper()
		col := a.columns[a.compIdx[compID]]
		backing := *col.backing.(*[]RichPointer)
		rp := backing[slot]
		if rp.Ptr != nil || rp.Label != "" || rp.Bytes != nil {
			t.Errorf("backing slot %d not zeroed for GC safety: %+v", slot, rp)
		}
	}

	idRich := ComponentID[RichPointer](w)
	if !bRich.arch.hasPtrs {
		t.Fatal("expected RichPointer archetype hasPtrs == true")
	}

	// 1. NewEntityWith
	ents := make([]Entity, 5)
	for i := range ents {
		ents[i] = bRich.NewEntityWith(mkVal(i + 1))
	}
	for i, e := range ents {
		verifyVal(t, *bRich.Get(e), i+1)
	}

	// 2. Set (migration from RichPointer -> RichPointer+Position) zeros vacated slot in source arch
	richArch := bRich.arch
	oldSize := richArch.size // 5
	bPos.Set(ents[0], Position{X: 1, Y: 2})
	verifyVal(t, *bRich.Get(ents[0]), 1)
	if richArch.size != oldSize-1 {
		t.Fatalf("expected richArch size %d, got %d", oldSize-1, richArch.size)
	}
	// Slot oldSize-1 (index 4) in richArch was vacated via swapRemove and must be zeroed.
	assertSlotZeroed(t, richArch, idRich, oldSize-1)
	// Swap-moved entity (originally ents[4], val 5) now lives at index 0 in richArch.
	verifyVal(t, *bRich.Get(ents[4]), 5)

	// 3. Remove component (RichPointer) from migrated entity zeros slot in (RichPointer, Position) arch
	migratedArch := w.metas[ents[0].ID].arch
	bRich.Remove(ents[0])
	if bRich.Has(ents[0]) {
		t.Error("expected RichPointer removed")
	}
	assertSlotZeroed(t, migratedArch, idRich, 0)

	// 4. RemoveEntity (non-last swapRemove) and RemoveEntity (last element)
	// richArch currently holds 4 entities: [ents[4](5), ents[1](2), ents[2](3), ents[3](4)]
	w.RemoveEntity(ents[1]) // non-last swapRemove -> index 3 becomes trailing zeroed slot
	assertSlotZeroed(t, richArch, idRich, 3)
	verifyVal(t, *bRich.Get(ents[3]), 4)

	w.RemoveEntity(ents[2]) // index 2 is now last element -> index 2 zeroed
	assertSlotZeroed(t, richArch, idRich, 2)

	// 5. RemoveEntities batch
	eExtra1 := bRich.NewEntityWith(mkVal(10))
	eExtra2 := bRich.NewEntityWith(mkVal(11))
	prevSize := richArch.size
	w.RemoveEntities([]Entity{ents[4], eExtra1})
	for s := richArch.size; s < prevSize; s++ {
		assertSlotZeroed(t, richArch, idRich, s)
	}
	verifyVal(t, *bRich.Get(ents[3]), 4)
	verifyVal(t, *bRich.Get(eExtra2), 11)

	// 6. ClearEntities zeroes all remaining backing slots
	prevSize = richArch.size
	w.ClearEntities()
	for s := 0; s < prevSize; s++ {
		assertSlotZeroed(t, richArch, idRich, s)
	}
}

func TestTransitionEdgeAndPairHeapFallback(t *testing.T) {
	type ec1 struct{ V int }
	type ec2 struct{ V int }
	type ec3 struct{ V int }
	type ec4 struct{ V int }
	type ec5 struct{ V int }
	type ec6 struct{ V int }
	type ec7 struct{ V int }
	type ec8 struct{ V int }
	type ec9 struct{ V int }
	type ec10 struct{ V int }
	type ec11 struct{ V int }
	type ec12 struct{ V int }
	type ec13 struct{ V int }
	type ec14 struct{ V int }
	type ec15 struct{ V int }
	type ec16 struct{ V int }
	type ec17 struct{ V int }
	type ec18 struct{ V int }
	type ec19 struct{ V int }

	w := NewWorld(16, 4)
	applyChain := func(e Entity, base int) {
		ptrVal := base
		SetComponent(w, e, RichPointer{
			Ptr:   &ptrVal,
			Label: fmt.Sprintf("rich-%d", base),
			Bytes: []byte{byte(base), byte(base + 1)},
		})
		SetComponent(w, e, ec1{V: base + 1})
		SetComponent(w, e, ec2{V: base + 2})
		SetComponent(w, e, ec3{V: base + 3})
		SetComponent(w, e, ec4{V: base + 4})
		SetComponent(w, e, ec5{V: base + 5})
		SetComponent(w, e, ec6{V: base + 6})
		SetComponent(w, e, ec7{V: base + 7})
		SetComponent(w, e, ec8{V: base + 8})
		SetComponent(w, e, ec9{V: base + 9})
		SetComponent(w, e, ec10{V: base + 10})
		SetComponent(w, e, ec11{V: base + 11})
		SetComponent(w, e, ec12{V: base + 12})
		SetComponent(w, e, ec13{V: base + 13})
		SetComponent(w, e, ec14{V: base + 14})
		SetComponent(w, e, ec15{V: base + 15})
		SetComponent(w, e, ec16{V: base + 16})
		SetComponent(w, e, ec17{V: base + 17})
		SetComponent(w, e, ec18{V: base + 18})
		SetComponent(w, e, ec19{V: base + 19})
	}

	e1 := w.CreateEntity()
	applyChain(e1, 100)

	// 20 transitions exhaust both inlineEdges (16) and inlinePairs (64)
	if w.inlineEdgeCount != len(w.inlineEdges) {
		t.Errorf("expected inlineEdgeCount == %d, got %d", len(w.inlineEdges), w.inlineEdgeCount)
	}
	if w.inlinePairCount == 0 || w.inlinePairCount > len(w.inlinePairs) {
		t.Errorf("unexpected inlinePairCount: %d", w.inlinePairCount)
	}

	// Second entity reuses cached heap-allocated edges and pairs cleanly
	e2 := w.CreateEntity()
	applyChain(e2, 200)

	verifyEntity := func(e Entity, base int) {
		t.Helper()
		rp := GetComponent[RichPointer](w, e)
		if rp == nil || rp.Ptr == nil || *rp.Ptr != base || rp.Label != fmt.Sprintf("rich-%d", base) {
			t.Fatalf("RichPointer mismatch for base %d: %+v", base, rp)
		}
		vals := []int{
			GetComponent[ec1](w, e).V,
			GetComponent[ec2](w, e).V,
			GetComponent[ec3](w, e).V,
			GetComponent[ec4](w, e).V,
			GetComponent[ec5](w, e).V,
			GetComponent[ec6](w, e).V,
			GetComponent[ec7](w, e).V,
			GetComponent[ec8](w, e).V,
			GetComponent[ec9](w, e).V,
			GetComponent[ec10](w, e).V,
			GetComponent[ec11](w, e).V,
			GetComponent[ec12](w, e).V,
			GetComponent[ec13](w, e).V,
			GetComponent[ec14](w, e).V,
			GetComponent[ec15](w, e).V,
			GetComponent[ec16](w, e).V,
			GetComponent[ec17](w, e).V,
			GetComponent[ec18](w, e).V,
			GetComponent[ec19](w, e).V,
		}
		for i, got := range vals {
			want := base + i + 1
			if got != want {
				t.Errorf("ec%d mismatch: expected %d, got %d", i+1, want, got)
			}
		}
	}

	verifyEntity(e1, 100)
	verifyEntity(e2, 200)

	// Removing components via heap-allocated remove edges preserves remaining values
	RemoveComponent[ec19](w, e1)
	RemoveComponent[ec10](w, e1)
	if HasComponent[ec19](w, e1) || HasComponent[ec10](w, e1) {
		t.Error("expected ec19 and ec10 removed from e1")
	}
	if got := GetComponent[ec18](w, e1); got == nil || got.V != 118 {
		t.Errorf("expected ec18 == 118 after removal, got %+v", got)
	}
	if rp := GetComponent[RichPointer](w, e1); rp == nil || rp.Ptr == nil || *rp.Ptr != 100 {
		t.Errorf("expected RichPointer preserved on e1 after removal, got %+v", rp)
	}
}

func TestArchetypeMultiOverflow(t *testing.T) {
	w := NewWorld()
	b2 := NewBuilder2[Position, Velocity](w)
	srcArch := b2.arch

	ents := make([]Entity, 8)
	for i := range ents {
		ents[i] = b2.NewEntityWith(
			Position{X: float32(i + 1), Y: float32((i + 1) * 10)},
			Velocity{DX: 1, DY: 2},
		)
	}

	swHealth := NewSwapper[Velocity, Health](w)
	swD1 := NewSwapper[Velocity, Dummy1](w)
	swD2 := NewSwapper[Velocity, Dummy2](w)
	swD3 := NewSwapper[Velocity, Dummy3](w)
	swD4 := NewSwapper[Velocity, Dummy4](w)
	swD5 := NewSwapper[Velocity, Dummy5](w)

	swHealth.Swap(ents[0], Health{HP: 10})
	swD1.Swap(ents[1], Dummy1{Val: 11})
	swD2.Swap(ents[2], Dummy2{Val: 12})
	swD3.Swap(ents[3], Dummy3{Val: 13})
	if srcArch.multiCount != 4 || len(srcArch.multiOverflow) != 0 {
		t.Fatalf("expected 4 inline multi-edges and 0 overflow, got multiCount=%d overflow=%d",
			srcArch.multiCount, len(srcArch.multiOverflow))
	}

	// 5th and 6th distinct transitions from srcArch overflow into multiOverflow
	swD4.Swap(ents[4], Dummy4{Val: 14})
	swD5.Swap(ents[5], Dummy5{Val: 15})
	if srcArch.multiCount != 4 || len(srcArch.multiOverflow) != 2 {
		t.Fatalf("expected multiCount=4 and overflow=2, got multiCount=%d overflow=%d",
			srcArch.multiCount, len(srcArch.multiOverflow))
	}

	// Fresh swappers bypass Swapper.lastSrc cache and hit both multiInline and multiOverflow
	NewSwapper[Velocity, Health](w).Swap(ents[6], Health{HP: 16})
	NewSwapper[Velocity, Dummy5](w).Swap(ents[7], Dummy5{Val: 17})
	if srcArch.multiCount != 4 || len(srcArch.multiOverflow) != 2 {
		t.Fatalf("expected cached lookup without duplicate edges, got multiCount=%d overflow=%d",
			srcArch.multiCount, len(srcArch.multiOverflow))
	}

	for i, e := range ents {
		if HasComponent[Velocity](w, e) {
			t.Errorf("entity %d should no longer have Velocity", i)
		}
		p := GetComponent[Position](w, e)
		wantPos := Position{X: float32(i + 1), Y: float32((i + 1) * 10)}
		if p == nil || *p != wantPos {
			t.Errorf("entity %d Position mismatch: expected %+v, got %+v", i, wantPos, p)
		}
	}

	if got := GetComponent[Health](w, ents[0]); got == nil || got.HP != 10 {
		t.Errorf("ents[0] Health mismatch: %+v", got)
	}
	if got := GetComponent[Dummy1](w, ents[1]); got == nil || got.Val != 11 {
		t.Errorf("ents[1] Dummy1 mismatch: %+v", got)
	}
	if got := GetComponent[Dummy2](w, ents[2]); got == nil || got.Val != 12 {
		t.Errorf("ents[2] Dummy2 mismatch: %+v", got)
	}
	if got := GetComponent[Dummy3](w, ents[3]); got == nil || got.Val != 13 {
		t.Errorf("ents[3] Dummy3 mismatch: %+v", got)
	}
	if got := GetComponent[Dummy4](w, ents[4]); got == nil || got.Val != 14 {
		t.Errorf("ents[4] Dummy4 mismatch: %+v", got)
	}
	if got := GetComponent[Dummy5](w, ents[5]); got == nil || got.Val != 15 {
		t.Errorf("ents[5] Dummy5 mismatch: %+v", got)
	}
	if got := GetComponent[Health](w, ents[6]); got == nil || got.HP != 16 {
		t.Errorf("ents[6] Health mismatch: %+v", got)
	}
	if got := GetComponent[Dummy5](w, ents[7]); got == nil || got.Val != 17 {
		t.Errorf("ents[7] Dummy5 mismatch: %+v", got)
	}
}
