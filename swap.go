package teishoku

import "unsafe"

// Swapper atomically removes component RemT and adds component AddT on an
// entity in a single archetype transition.
type Swapper[RemT, AddT any] struct {
	world      *World
	lastSrc    *archetype
	lastEdge   *transitionEdge
	remID      uint8
	addID      uint8
	lastAddCol uint8
}

// NewSwapper creates a Swapper replacing RemT with AddT in World w.
// Panics if RemT and AddT resolve to the same component type.
func NewSwapper[RemT, AddT any](w *World) *Swapper[RemT, AddT] {
	remID := ComponentID[RemT](w)
	addID := ComponentID[AddT](w)
	if remID == addID {
		panic("teishoku: duplicate component types in Swapper")
	}
	return &Swapper[RemT, AddT]{
		world: w,
		remID: remID,
		addID: addID,
	}
}

// Swap removes RemT from entity e and sets AddT to addVal in a single
// archetype transition. Does nothing if e is invalid or lacks RemT.
func (s *Swapper[RemT, AddT]) Swap(e Entity, addVal AddT) {
	w := s.world
	if uint(e.ID) >= uint(len(w.metas)) {
		return
	}
	meta := &w.metas[e.ID]
	a := meta.arch
	if a == nil || meta.version != e.Version {
		return
	}
	if a.compIdx[s.remID] < 0 {
		return
	}
	var edge *transitionEdge
	if a != s.lastSrc {
		newMask := a.mask
		newMask.unset(s.remID)
		newMask.set(s.addID)
		edge = w.getTransitionEdge(a, newMask)
		s.lastSrc = a
		s.lastEdge = edge
		s.lastAddCol = uint8(edge.target.compIdx[s.addID])
	} else {
		edge = s.lastEdge
	}
	newIdx := w.moveEntity(e, meta, a, edge)
	dst := edge.target
	var z AddT
	if unsafe.Sizeof(z) > 0 {
		*(*AddT)(unsafe.Add(dst.columns[s.lastAddCol].data, uintptr(newIdx)*unsafe.Sizeof(z))) = addVal
	}
}

// SwapBatch invokes Swap on every entity in entities with addVal.
func (s *Swapper[RemT, AddT]) SwapBatch(entities []Entity, addVal AddT) {
	for _, e := range entities {
		s.Swap(e, addVal)
	}
}
