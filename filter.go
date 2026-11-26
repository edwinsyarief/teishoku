package teishoku

// Filter0 matches all entities that currently possess zero components, with
// optional exclusion support.
type Filter0 struct {
	filterBase
	matched []*archetype
}

// NewFilter0 creates a Filter0 matching entities with no components in World w.
func NewFilter0(w *World) *Filter0 {
	f := &Filter0{
		filterBase: filterBase{
			world: w,
		},
	}
	f.sync()
	return f
}

func (f *Filter0) sync() {
	arches := f.world.archetypes
	for i := f.lastArchIdx; i < len(arches); i++ {
		a := arches[i]
		if f.matches(a) {
			f.matched = append(f.matched, a)
		}
	}
	f.lastArchIdx = len(arches)
}

func (f *Filter0) addExclude(id uint8) {
	f.exclude.set(id)
	f.matched = f.matched[:0]
	f.lastArchIdx = 0
	f.cachedEntities = nil
	f.sync()
}

// Without excludes entities possessing any of compIDs from matching this
// filter and returns the filter for chaining.
func (f *Filter0) Without(compIDs ...uint8) *Filter0 {
	if len(compIDs) == 0 {
		return f
	}
	for _, id := range compIDs {
		f.exclude.set(id)
	}
	f.matched = f.matched[:0]
	f.lastArchIdx = 0
	f.cachedEntities = nil
	f.sync()
	return f
}

// Count returns the total number of entities currently matching the filter.
func (f *Filter0) Count() int {
	if len(f.world.archetypes) != f.lastArchIdx {
		f.sync()
	}
	total := 0
	for _, a := range f.matched {
		total += a.size
	}
	return total
}

// Entities returns a cached slice of all entities currently matching the
// filter. The slice is lazily built and reused across calls while the world
// has not mutated.
func (f *Filter0) Entities() []Entity {
	if len(f.world.archetypes) != f.lastArchIdx {
		f.sync()
	}
	if f.cachedEntities == nil || f.lastMutationVersion != f.world.mutationVersion {
		total := 0
		for _, a := range f.matched {
			total += a.size
		}
		if cap(f.cachedEntities) < total {
			f.cachedEntities = make([]Entity, total)
		} else {
			f.cachedEntities = f.cachedEntities[:total]
		}
		idx := 0
		for _, a := range f.matched {
			if a.size == 0 {
				continue
			}
			copy(f.cachedEntities[idx:idx+a.size], a.entityIDs[:a.size])
			idx += a.size
		}
		f.lastMutationVersion = f.world.mutationVersion
	}
	return f.cachedEntities
}

// RemoveEntities removes all entities matching the filter from the world.
func (f *Filter0) RemoveEntities() {
	if len(f.world.archetypes) != f.lastArchIdx {
		f.sync()
	}
	for _, a := range f.matched {
		if a.size > 0 {
			f.world.removeAllInArch(a)
		}
	}
}

// EachChunk invokes fn once per non-empty matching archetype with its entity
// slice.
func (f *Filter0) EachChunk(fn func([]Entity)) {
	if len(f.world.archetypes) != f.lastArchIdx {
		f.sync()
	}
	for _, a := range f.matched {
		if a.size == 0 {
			continue
		}
		fn(a.entityIDs[:a.size])
	}
}

// Chunks0 iterates over non-empty matching archetype chunks for Filter0.
type Chunks0 struct {
	matched  []*archetype
	matchIdx int
	cur      *archetype
}

// Chunks returns a Chunks0 iterator over matching archetypes.
func (f *Filter0) Chunks() Chunks0 {
	if len(f.world.archetypes) != f.lastArchIdx {
		f.sync()
	}
	return Chunks0{
		matched:  f.matched,
		matchIdx: -1,
	}
}

// Next advances to the next non-empty matching archetype chunk.
func (c *Chunks0) Next() bool {
	for {
		c.matchIdx++
		if c.matchIdx >= len(c.matched) {
			return false
		}
		a := c.matched[c.matchIdx]
		if a.size == 0 {
			continue
		}
		c.cur = a
		return true
	}
}

// Entities returns the contiguous slice of entities in the current chunk.
func (c *Chunks0) Entities() []Entity {
	return c.cur.entityIDs[:c.cur.size]
}

// Query0 is an allocation-free per-entity iterator snapshot for Filter0.
type Query0 struct {
	matched  []*archetype
	entities []Entity
	matchIdx int
	idx      int
	archSize int
}

// Query returns a Query0 iterator for Filter0.
func (f *Filter0) Query() Query0 {
	if len(f.world.archetypes) != f.lastArchIdx {
		f.sync()
	}
	q := Query0{
		matched: f.matched,
	}
	if len(f.matched) > 0 && f.matched[0].size > 0 {
		a := f.matched[0]
		q.entities = a.entityIDs
		q.archSize = a.size
		q.matchIdx = 0
		q.idx = -1
	} else {
		q.matchIdx = -1
		q.idx = 0
		q.archSize = 0
	}
	return q
}

// Next advances the query to the next matching entity.
func (q *Query0) Next() bool {
	q.idx++
	if q.idx < q.archSize {
		return true
	}
	return q.nextArchetype()
}

//go:noinline
func (q *Query0) nextArchetype() bool {
	for {
		q.matchIdx++
		if q.matchIdx >= len(q.matched) {
			return false
		}
		a := q.matched[q.matchIdx]
		if a.size == 0 {
			continue
		}
		q.entities = a.entityIDs
		q.archSize = a.size
		q.idx = 0
		return true
	}
}

// Entity returns the current entity in the query.
func (q *Query0) Entity() Entity {
	return q.entities[q.idx]
}
