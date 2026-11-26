package teishoku

import (
	"math/bits"
	"reflect"
	"unsafe"
)

// MaxComponentTypes defines the maximum number of unique component types that can be
// registered in a World. This value is fixed at 256.
const MaxComponentTypes = 256

const (
	defaultWorldCapacity = 1024
	defaultArchCapacity  = 32
	defaultArchInitCap   = 64
)

var defaultCompIdx [MaxComponentTypes]int8

func init() {
	for i := range defaultCompIdx {
		defaultCompIdx[i] = -1
	}
}

// Entity represents a unique identifier for an object in the World. It combines
// a 32-bit ID with a 32-bit version to ensure that recycled IDs are not confused
// with new entities.
type Entity struct {
	// ID is the unique, recyclable identifier for the entity.
	ID uint32
	// Version is a generation counter to protect against stale entity references.
	// It is incremented each time an entity ID is reused.
	Version uint32
}

// entitySlot holds the compact 16-byte internal location and generation state of an entity.
type entitySlot struct {
	arch    *archetype
	index   uint32
	version uint32
}

type columnPair struct {
	elemSize uintptr
	ops      *columnOps
	srcCol   uint8
	dstCol   uint8
	hasPtrs  bool
}

type transitionEdge struct {
	target *archetype
	pairs  []columnPair
	allPOD bool
}

type multiEdgeEntry struct {
	mask bitmask256
	edge *transitionEdge
}

// archetype holds dense SoA storage for one unique component-set mask.
type archetype struct {
	entityIDs     []Entity
	columns       []column
	mask          bitmask256
	compIdx       [MaxComponentTypes]int8 // -1 if absent, else 0..len(columns)-1
	addEdges      [MaxComponentTypes]*transitionEdge
	remEdges      [MaxComponentTypes]*transitionEdge
	multiInline   [4]multiEdgeEntry
	multiCount    uint8
	multiOverflow []multiEdgeEntry
	size          int
	capacity      int
	index         int
	hasPtrs       bool
}

// ensureCap ensures the archetype has storage capacity for at least needed entities.
func (a *archetype) ensureCap(needed int) {
	if needed <= a.capacity {
		return
	}
	newCap := a.capacity * 2
	if newCap < defaultArchCapacity {
		newCap = defaultArchCapacity
	}
	if newCap < needed {
		newCap = needed
	}
	newEnts := make([]Entity, newCap)
	if a.size > 0 {
		copy(newEnts[:a.size], a.entityIDs[:a.size])
	}
	a.entityIDs = newEnts
	for i := range a.columns {
		col := &a.columns[i]
		col.data, col.backing = col.ops.grow(col.backing, a.size, newCap)
	}
	a.capacity = newCap
}

// swapRemove removes the entity at idx by swapping the last entity into its slot.
func (a *archetype) swapRemove(idx int, metas []entitySlot) {
	lastIdx := a.size - 1
	if idx < lastIdx {
		lastEnt := a.entityIDs[lastIdx]
		a.entityIDs[idx] = lastEnt
		metas[lastEnt.ID].index = uint32(idx)
		if !a.hasPtrs {
			for i := range a.columns {
				c := &a.columns[i]
				copyPOD(
					unsafe.Add(c.data, uintptr(idx)*c.elemSize),
					unsafe.Add(c.data, uintptr(lastIdx)*c.elemSize),
					c.elemSize,
				)
			}
		} else {
			for i := range a.columns {
				c := &a.columns[i]
				if c.hasPtrs {
					c.ops.swapRemove(c.backing, idx, lastIdx)
				} else {
					copyPOD(
						unsafe.Add(c.data, uintptr(idx)*c.elemSize),
						unsafe.Add(c.data, uintptr(lastIdx)*c.elemSize),
						c.elemSize,
					)
				}
			}
		}
	} else if a.hasPtrs {
		for i := range a.columns {
			c := &a.columns[i]
			if c.hasPtrs {
				c.ops.zeroRange(c.backing, lastIdx, lastIdx+1)
			}
		}
	}
	a.size = lastIdx
}

// World is the central single-threaded container for all entities, components,
// and archetypes.
type World struct {
	resources       Resources
	metas           []entitySlot
	freeIDs         []uint32
	nextID          uint32
	entityCount     int
	archInitCap     int
	archetypes      []*archetype
	emptyArch       *archetype
	maskToArch      map[bitmask256]*archetype
	compTypeMap     map[reflect.Type]uint8
	compOps         [MaxComponentTypes]*columnOps
	nextCompID      uint16
	mutationVersion uint32
	inlineEdges     [8]transitionEdge
	inlineEdgeCount int
	inlinePairs     [24]columnPair
	inlinePairCount int
}

// NewWorld creates and initializes a new World with an optional initial entity capacity.
func NewWorld(initialCapacity ...int) *World {
	entityCap := defaultWorldCapacity
	archCap := defaultArchInitCap
	if len(initialCapacity) >= 1 && initialCapacity[0] > 0 {
		entityCap = initialCapacity[0]
		archCap = initialCapacity[0]
	}
	if len(initialCapacity) >= 2 && initialCapacity[1] >= 0 {
		archCap = initialCapacity[1]
	}
	// Slot 0 is a reserved sentinel (arch: nil, version: 0).
	metas := make([]entitySlot, 1, entityCap+1)
	w := &World{
		metas:       metas,
		freeIDs:     make([]uint32, 0, entityCap),
		nextID:      1,
		archInitCap: archCap,
		archetypes:  make([]*archetype, 0, 16),
	}
	w.emptyArch = w.getOrCreateArchetype(bitmask256{}, nil)
	return w
}

// ComponentID returns the unique component type ID for T within w, registering
// it on first use.
func ComponentID[T any](w *World) uint8 {
	t := reflect.TypeFor[T]()
	if w.compTypeMap != nil {
		if id, ok := w.compTypeMap[t]; ok {
			return id
		}
	}
	if w.nextCompID >= MaxComponentTypes {
		panic("teishoku: too many component types")
	}
	id := uint8(w.nextCompID)
	w.nextCompID++
	ops := newColumnOps[T]()
	if w.compTypeMap == nil {
		w.compTypeMap = make(map[reflect.Type]uint8, 16)
	}
	w.compTypeMap[t] = id
	w.compOps[id] = ops
	return id
}

// CreateEntity creates a new entity with no components.
func (w *World) CreateEntity() Entity {
	e, _ := w.createEntityInArch(w.emptyArch)
	return e
}

// CreateEntities creates count entities with no components in a single batch.
func (w *World) CreateEntities(count int) {
	w.createEntitiesInArch(w.emptyArch, count)
}

func (w *World) createEntityInArch(a *archetype) (Entity, int) {
	if a.size == a.capacity {
		a.ensureCap(a.size + 1)
	}
	row := a.size
	a.size++

	var id uint32
	var ver uint32
	if n := len(w.freeIDs); n > 0 {
		id = w.freeIDs[n-1]
		w.freeIDs = w.freeIDs[:n-1]
		meta := &w.metas[id]
		ver = meta.version
		if ver == 0 {
			ver = 1
		}
		meta.arch = a
		meta.index = uint32(row)
		meta.version = ver
	} else {
		id = w.nextID
		w.nextID++
		ver = 1
		if int(id) < len(w.metas) {
			meta := &w.metas[id]
			meta.arch = a
			meta.index = uint32(row)
			meta.version = ver
		} else {
			w.metas = append(w.metas, entitySlot{arch: a, index: uint32(row), version: ver})
		}
	}

	ent := Entity{ID: id, Version: ver}
	a.entityIDs[row] = ent
	w.entityCount++
	w.mutationVersion++
	return ent, row
}

func (w *World) createEntitiesInArch(a *archetype, count int) int {
	if count <= 0 {
		return -1
	}
	startRow := a.size
	a.ensureCap(startRow + count)
	a.size = startRow + count

	fromFree := min(len(w.freeIDs), count)
	if fromFree > 0 {
		freeStart := len(w.freeIDs) - fromFree
		popped := w.freeIDs[freeStart:]
		w.freeIDs = w.freeIDs[:freeStart]
		metas := w.metas
		entSlice := a.entityIDs[startRow : startRow+fromFree]
		for k := 0; k < fromFree; k++ {
			id := popped[fromFree-1-k]
			m := &metas[id]
			ver := m.version
			if ver == 0 {
				ver = 1
			}
			row := startRow + k
			m.arch = a
			m.index = uint32(row)
			m.version = ver
			entSlice[k] = Entity{ID: id, Version: ver}
		}
	}

	if rem := count - fromFree; rem > 0 {
		startID := w.nextID
		newNextID := startID + uint32(rem)
		w.nextID = newNextID
		neededMetas := int(newNextID)
		if cap(w.metas) < neededMetas {
			newCap := cap(w.metas) * 2
			if newCap < neededMetas {
				newCap = neededMetas
			}
			newMetas := make([]entitySlot, neededMetas, newCap)
			copy(newMetas, w.metas)
			w.metas = newMetas
		} else {
			w.metas = w.metas[:neededMetas]
		}
		metasSlice := w.metas[startID:neededMetas]
		entSlice := a.entityIDs[startRow+fromFree : startRow+count]
		baseRow := uint32(startRow + fromFree)
		for k := 0; k < rem; k++ {
			row := baseRow + uint32(k)
			id := startID + uint32(k)
			metasSlice[k] = entitySlot{arch: a, index: row, version: 1}
			entSlice[k] = Entity{ID: id, Version: 1}
		}
	}

	w.entityCount += count
	w.mutationVersion++
	return startRow
}

// RemoveEntity marks the entity as invalid and recycles its ID for future use.
func (w *World) RemoveEntity(e Entity) {
	if int(e.ID) >= len(w.metas) {
		return
	}
	meta := &w.metas[e.ID]
	a := meta.arch
	if a == nil || meta.version != e.Version {
		return
	}
	a.swapRemove(int(meta.index), w.metas)
	meta.arch = nil
	meta.version++
	if meta.version == 0 {
		meta.version = 1
	}
	w.freeIDs = append(w.freeIDs, e.ID)
	w.entityCount--
	w.mutationVersion++
}

// RemoveEntities removes a slice of entities from the world in a batch.
func (w *World) RemoveEntities(ents []Entity) {
	removed := false
	for _, e := range ents {
		if int(e.ID) >= len(w.metas) {
			continue
		}
		meta := &w.metas[e.ID]
		a := meta.arch
		if a == nil || meta.version != e.Version {
			continue
		}
		a.swapRemove(int(meta.index), w.metas)
		meta.arch = nil
		meta.version++
		if meta.version == 0 {
			meta.version = 1
		}
		w.freeIDs = append(w.freeIDs, e.ID)
		w.entityCount--
		removed = true
	}
	if removed {
		w.mutationVersion++
	}
}

func (w *World) removeAllInArch(a *archetype) {
	n := a.size
	if n == 0 {
		return
	}
	eids := a.entityIDs[:n]
	metas := w.metas
	oldLen := len(w.freeIDs)
	newLen := oldLen + n
	if cap(w.freeIDs) < newLen {
		newCap := cap(w.freeIDs) * 2
		if newCap < newLen {
			newCap = newLen
		}
		newFree := make([]uint32, newLen, newCap)
		copy(newFree, w.freeIDs)
		w.freeIDs = newFree
	} else {
		w.freeIDs = w.freeIDs[:newLen]
	}
	dstFree := w.freeIDs[oldLen:newLen]
	for i := 0; i < n; i++ {
		id := eids[i].ID
		m := &metas[id]
		m.arch = nil
		m.version++
		if m.version == 0 {
			m.version = 1
		}
		dstFree[i] = id
	}
	if a.hasPtrs {
		for i := range a.columns {
			if a.columns[i].hasPtrs {
				a.columns[i].ops.zeroRange(a.columns[i].backing, 0, n)
			}
		}
	}
	a.size = 0
	w.entityCount -= n
	w.mutationVersion++
}

// ClearEntities removes all entities from the world while keeping archetypes allocated.
func (w *World) ClearEntities() {
	for _, a := range w.archetypes {
		if a.size > 0 {
			w.removeAllInArch(a)
		}
	}
}

// IsValid checks whether the given entity is currently alive.
func (w *World) IsValid(e Entity) bool {
	if int(e.ID) >= len(w.metas) {
		return false
	}
	m := w.metas[e.ID]
	return m.arch != nil && m.version == e.Version
}

func (w *World) isValidSlot(e Entity) (*entitySlot, bool) {
	if int(e.ID) >= len(w.metas) {
		return nil, false
	}
	m := &w.metas[e.ID]
	if m.arch == nil || m.version != e.Version {
		return nil, false
	}
	return m, true
}

// Len returns the number of living entities in the world.
func (w *World) Len() int {
	return w.entityCount
}

// Resources returns the world's resource manager.
func (w *World) Resources() *Resources {
	return &w.resources
}

func (w *World) getOrCreateArchetype(mask bitmask256, compIDs []uint8) *archetype {
	if mask.isZero() && w.emptyArch != nil {
		return w.emptyArch
	}
	if !mask.isZero() && w.maskToArch != nil {
		if a, ok := w.maskToArch[mask]; ok {
			return a
		}
	}
	var cols []column
	if len(compIDs) > 0 {
		cols = make([]column, len(compIDs))
	}
	a := &archetype{
		index:   len(w.archetypes),
		mask:    mask,
		compIdx: defaultCompIdx,
		columns: cols,
	}
	if w.archInitCap > 0 {
		a.entityIDs = make([]Entity, w.archInitCap)
		a.capacity = w.archInitCap
	}
	for i, cid := range compIDs {
		ops := w.compOps[cid]
		a.compIdx[cid] = int8(i)
		col := column{
			elemSize: ops.elemSize,
			compID:   cid,
			hasPtrs:  ops.hasPtrs,
			ops:      ops,
		}
		if w.archInitCap > 0 {
			col.data, col.backing = ops.alloc(w.archInitCap)
		}
		a.columns[i] = col
		if ops.hasPtrs {
			a.hasPtrs = true
		}
	}
	w.archetypes = append(w.archetypes, a)
	if !mask.isZero() {
		if w.maskToArch == nil {
			w.maskToArch = make(map[bitmask256]*archetype, 16)
		}
		w.maskToArch[mask] = a
	}
	return a
}

func (w *World) buildTransitionEdge(src, dst *archetype) *transitionEdge {
	var pairs []columnPair
	allPOD := true
	if need := len(src.columns); need > 0 {
		if w.inlinePairCount+need <= len(w.inlinePairs) {
			start := w.inlinePairCount
			n := 0
			for srcIdx, sc := range src.columns {
				dstIdx := dst.compIdx[sc.compID]
				if dstIdx >= 0 {
					w.inlinePairs[start+n] = columnPair{
						srcCol:   uint8(srcIdx),
						dstCol:   uint8(dstIdx),
						elemSize: sc.elemSize,
						hasPtrs:  sc.hasPtrs,
						ops:      sc.ops,
					}
					n++
					if sc.hasPtrs {
						allPOD = false
					}
				}
			}
			if n > 0 {
				w.inlinePairCount += n
				pairs = w.inlinePairs[start : start+n]
			}
		} else {
			pairs = make([]columnPair, 0, need)
			for srcIdx, sc := range src.columns {
				dstIdx := dst.compIdx[sc.compID]
				if dstIdx >= 0 {
					pairs = append(pairs, columnPair{
						srcCol:   uint8(srcIdx),
						dstCol:   uint8(dstIdx),
						elemSize: sc.elemSize,
						hasPtrs:  sc.hasPtrs,
						ops:      sc.ops,
					})
					if sc.hasPtrs {
						allPOD = false
					}
				}
			}
		}
	}
	var edge *transitionEdge
	if w.inlineEdgeCount < len(w.inlineEdges) {
		edge = &w.inlineEdges[w.inlineEdgeCount]
		w.inlineEdgeCount++
	} else {
		edge = &transitionEdge{}
	}
	*edge = transitionEdge{
		target: dst,
		pairs:  pairs,
		allPOD: allPOD,
	}
	return edge
}

func (w *World) getAddEdge(src *archetype, addID uint8) *transitionEdge {
	if edge := src.addEdges[addID]; edge != nil {
		return edge
	}
	newMask := src.mask
	newMask.set(addID)
	var dst *archetype
	if w.maskToArch != nil {
		dst = w.maskToArch[newMask]
	}
	if dst == nil {
		var buf [MaxComponentTypes]uint8
		n := len(src.columns)
		for i := 0; i < n; i++ {
			buf[i] = src.columns[i].compID
		}
		var compIDs []uint8
		if src.compIdx[addID] < 0 {
			buf[n] = addID
			compIDs = buf[:n+1]
		} else {
			compIDs = buf[:n]
		}
		dst = w.getOrCreateArchetype(newMask, compIDs)
	}
	edge := w.buildTransitionEdge(src, dst)
	src.addEdges[addID] = edge
	return edge
}

func (w *World) getRemoveEdge(src *archetype, remID uint8) *transitionEdge {
	if edge := src.remEdges[remID]; edge != nil {
		return edge
	}
	newMask := src.mask
	newMask.unset(remID)
	var dst *archetype
	if newMask.isZero() {
		dst = w.emptyArch
	} else if w.maskToArch != nil {
		dst = w.maskToArch[newMask]
	}
	if dst == nil {
		var buf [MaxComponentTypes]uint8
		n := 0
		for i := range src.columns {
			cid := src.columns[i].compID
			if cid != remID {
				buf[n] = cid
				n++
			}
		}
		dst = w.getOrCreateArchetype(newMask, buf[:n])
	}
	edge := w.buildTransitionEdge(src, dst)
	src.remEdges[remID] = edge
	return edge
}

func (w *World) getTransitionEdge(src *archetype, targetMask bitmask256) *transitionEdge {
	for i := uint8(0); i < src.multiCount; i++ {
		if src.multiInline[i].mask == targetMask {
			return src.multiInline[i].edge
		}
	}
	if len(src.multiOverflow) > 0 {
		for i := range src.multiOverflow {
			if src.multiOverflow[i].mask == targetMask {
				return src.multiOverflow[i].edge
			}
		}
	}
	var dst *archetype
	if targetMask.isZero() {
		dst = w.emptyArch
	} else if w.maskToArch != nil {
		dst = w.maskToArch[targetMask]
	}
	if dst == nil {
		var buf [MaxComponentTypes]uint8
		n := 0
		for wordIdx := 0; wordIdx < 4; wordIdx++ {
			word := targetMask[wordIdx]
			base := uint8(wordIdx << 6)
			for word != 0 {
				bit := uint8(bits.TrailingZeros64(word))
				buf[n] = base + bit
				n++
				word &= word - 1
			}
		}
		dst = w.getOrCreateArchetype(targetMask, buf[:n])
	}
	edge := w.buildTransitionEdge(src, dst)
	if src.multiCount < 4 {
		src.multiInline[src.multiCount] = multiEdgeEntry{mask: targetMask, edge: edge}
		src.multiCount++
	} else {
		src.multiOverflow = append(src.multiOverflow, multiEdgeEntry{mask: targetMask, edge: edge})
	}
	return edge
}

func (w *World) moveEntity(e Entity, meta *entitySlot, src *archetype, edge *transitionEdge) int {
	dst := edge.target
	newIdx := dst.size
	if newIdx == dst.capacity {
		dst.ensureCap(newIdx + 1)
	}
	dst.entityIDs[newIdx] = e
	dst.size = newIdx + 1

	oldIdx := int(meta.index)
	if edge.allPOD {
		for i := range edge.pairs {
			p := &edge.pairs[i]
			sc := &src.columns[p.srcCol]
			dc := &dst.columns[p.dstCol]
			copyPOD(
				unsafe.Add(dc.data, uintptr(newIdx)*p.elemSize),
				unsafe.Add(sc.data, uintptr(oldIdx)*p.elemSize),
				p.elemSize,
			)
		}
	} else {
		for i := range edge.pairs {
			p := &edge.pairs[i]
			sc := &src.columns[p.srcCol]
			dc := &dst.columns[p.dstCol]
			if p.hasPtrs {
				p.ops.copyElem(dc.backing, newIdx, sc.backing, oldIdx)
			} else {
				copyPOD(
					unsafe.Add(dc.data, uintptr(newIdx)*p.elemSize),
					unsafe.Add(sc.data, uintptr(oldIdx)*p.elemSize),
					p.elemSize,
				)
			}
		}
	}

	src.swapRemove(oldIdx, w.metas)
	meta.arch = dst
	meta.index = uint32(newIdx)
	w.mutationVersion++
	return newIdx
}
