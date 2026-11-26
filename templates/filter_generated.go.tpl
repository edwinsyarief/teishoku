type matchedArch{{.N}} struct {
	arch *archetype
	{{- range .Components}}
	c{{.ZeroIndex}} uint8
	{{- end}}
}

// Filter{{.Suffix}} matches all entities possessing components {{.TypeVars}},
// with optional component exclusions.
type Filter{{.Suffix}}[{{.Types}}] struct {
	filterBase
	matched []matchedArch{{.N}}
	{{- range .Components}}
	id{{.ZeroIndex}} uint8
	{{- end}}
}

// NewFilter{{.Suffix}} creates a Filter{{.Suffix}} for components {{.TypeVars}}
// in World w.
{{- if gt .N 1}}
// Panics if any component types resolve to the same component ID.
{{- end}}
func NewFilter{{.Suffix}}[{{.Types}}](w *World) *Filter{{.Suffix}}[{{.TypeVars}}] {
	{{- range .Components}}
	id{{.Index}} := ComponentID[{{.TypeName}}](w)
	{{- end}}
	{{- if gt .N 1}}
	if {{.DuplicateIDs}} {
		panic("teishoku: duplicate component types in Filter{{.Suffix}}")
	}
	{{- end}}
	var mask bitmask256
	{{- range .Components}}
	mask.set(id{{.Index}})
	{{- end}}
	f := &Filter{{.Suffix}}[{{.TypeVars}}]{
		filterBase: filterBase{
			world: w,
			mask:  mask,
		},
		{{- range .Components}}
		id{{.ZeroIndex}}: id{{.Index}},
		{{- end}}
	}
	f.sync()
	return f
}

func (f *Filter{{.Suffix}}[{{.TypeVars}}]) sync() {
	arches := f.world.archetypes
	for i := f.lastArchIdx; i < len(arches); i++ {
		a := arches[i]
		if f.matches(a) {
			f.matched = append(f.matched, matchedArch{{.N}}{
				arch: a,
				{{- range .Components}}
				c{{.ZeroIndex}}: uint8(a.compIdx[f.id{{.ZeroIndex}}]),
				{{- end}}
			})
		}
	}
	f.lastArchIdx = len(arches)
}

func (f *Filter{{.Suffix}}[{{.TypeVars}}]) addExclude(id uint8) {
	f.exclude.set(id)
	f.matched = f.matched[:0]
	f.lastArchIdx = 0
	f.cachedEntities = nil
	f.sync()
}

// Without excludes entities possessing any of compIDs from matching this
// filter and returns the filter for chaining.
func (f *Filter{{.Suffix}}[{{.TypeVars}}]) Without(compIDs ...uint8) *Filter{{.Suffix}}[{{.TypeVars}}] {
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
func (f *Filter{{.Suffix}}[{{.TypeVars}}]) Count() int {
	if len(f.world.archetypes) != f.lastArchIdx {
		f.sync()
	}
	total := 0
	for i := range f.matched {
		total += f.matched[i].arch.size
	}
	return total
}

// Entities returns a cached slice of all entities currently matching the
// filter. The slice is lazily built and reused across calls while the world
// has not mutated.
func (f *Filter{{.Suffix}}[{{.TypeVars}}]) Entities() []Entity {
	if len(f.world.archetypes) != f.lastArchIdx {
		f.sync()
	}
	if f.cachedEntities == nil || f.lastMutationVersion != f.world.mutationVersion {
		total := 0
		for i := range f.matched {
			total += f.matched[i].arch.size
		}
		if cap(f.cachedEntities) < total {
			f.cachedEntities = make([]Entity, total)
		} else {
			f.cachedEntities = f.cachedEntities[:total]
		}
		idx := 0
		for i := range f.matched {
			a := f.matched[i].arch
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
func (f *Filter{{.Suffix}}[{{.TypeVars}}]) RemoveEntities() {
	if len(f.world.archetypes) != f.lastArchIdx {
		f.sync()
	}
	for i := range f.matched {
		a := f.matched[i].arch
		if a.size > 0 {
			f.world.removeAllInArch(a)
		}
	}
}

// EachChunk invokes fn once per non-empty matching archetype with contiguous
// slices of the requested components.
func (f *Filter{{.Suffix}}[{{.TypeVars}}]) EachChunk(fn func({{.SliceTypes}})) {
	if len(f.world.archetypes) != f.lastArchIdx {
		f.sync()
	}
	for i := range f.matched {
		m := f.matched[i]
		n := m.arch.size
		if n == 0 {
			continue
		}
		{{- range .Components}}
		var z{{.ZeroIndex}} {{.TypeName}}
		p{{.ZeroIndex}} := m.arch.columns[m.c{{.ZeroIndex}}].data
		if unsafe.Sizeof(z{{.ZeroIndex}}) == 0 && p{{.ZeroIndex}} == nil {
			p{{.ZeroIndex}} = unsafe.Pointer(&zeroZST)
		}
		{{- end}}
		fn({{range $i, $e := .Components}}{{if $i}}, {{end}}unsafe.Slice((*{{$e.TypeName}})(p{{$e.ZeroIndex}}), n){{end}})
	}
}

// Chunks{{.Suffix}} iterates over non-empty matching archetype chunks, exposing
// contiguous component slices for batch processing.
type Chunks{{.Suffix}}[{{.Types}}] struct {
	matched  []matchedArch{{.N}}
	{{- range .Components}}
	base{{.ZeroIndex}} unsafe.Pointer
	{{- end}}
	entities []Entity
	matchIdx int
	archSize int
}

// Chunks returns a Chunks{{.Suffix}} iterator over matching archetypes.
func (f *Filter{{.Suffix}}[{{.TypeVars}}]) Chunks() Chunks{{.Suffix}}[{{.TypeVars}}] {
	if len(f.world.archetypes) != f.lastArchIdx {
		f.sync()
	}
	return Chunks{{.Suffix}}[{{.TypeVars}}]{
		matched:  f.matched,
		matchIdx: -1,
	}
}

// Next advances to the next non-empty matching archetype chunk.
func (c *Chunks{{.Suffix}}[{{.TypeVars}}]) Next() bool {
	for {
		c.matchIdx++
		if c.matchIdx >= len(c.matched) {
			return false
		}
		m := c.matched[c.matchIdx]
		a := m.arch
		if a.size == 0 {
			continue
		}
		{{- range .Components}}
		var z{{.ZeroIndex}} {{.TypeName}}
		p{{.ZeroIndex}} := a.columns[m.c{{.ZeroIndex}}].data
		if unsafe.Sizeof(z{{.ZeroIndex}}) == 0 && p{{.ZeroIndex}} == nil {
			p{{.ZeroIndex}} = unsafe.Pointer(&zeroZST)
		}
		c.base{{.ZeroIndex}} = p{{.ZeroIndex}}
		{{- end}}
		c.entities = a.entityIDs
		c.archSize = a.size
		return true
	}
}

// Entities returns the contiguous slice of entities in the current chunk.
func (c *Chunks{{.Suffix}}[{{.TypeVars}}]) Entities() []Entity {
	return c.entities[:c.archSize]
}

// Get returns the contiguous component slices for the current chunk.
func (c *Chunks{{.Suffix}}[{{.TypeVars}}]) Get() ({{.SliceTypes}}) {
	n := c.archSize
	return {{range $i, $e := .Components}}{{if $i}}, {{end}}unsafe.Slice((*{{$e.TypeName}})(c.base{{$e.ZeroIndex}}), n){{end}}
}

// Query{{.Suffix}} is an allocation-free per-entity iterator snapshot for
// Filter{{.Suffix}}.
type Query{{.Suffix}}[{{.Types}}] struct {
	matched  []matchedArch{{.N}}
	{{- range .Components}}
	base{{.ZeroIndex}} unsafe.Pointer
	{{- end}}
	entities []Entity
	matchIdx int
	idx      uintptr
	archSize uintptr
}

// Query returns a Query{{.Suffix}} iterator for Filter{{.Suffix}}.
func (f *Filter{{.Suffix}}[{{.TypeVars}}]) Query() Query{{.Suffix}}[{{.TypeVars}}] {
	if len(f.world.archetypes) != f.lastArchIdx {
		f.sync()
	}
	q := Query{{.Suffix}}[{{.TypeVars}}]{
		matched: f.matched,
	}
	if len(f.matched) > 0 && f.matched[0].arch.size > 0 {
		m := f.matched[0]
		a := m.arch
		{{- range .Components}}
		q.base{{.ZeroIndex}} = a.columns[m.c{{.ZeroIndex}}].data
		{{- end}}
		q.entities = a.entityIDs
		q.archSize = uintptr(a.size)
		q.matchIdx = 0
		q.idx = ^uintptr(0)
	} else {
		q.matchIdx = -1
		q.idx = 0
		q.archSize = 0
	}
	return q
}

// Next advances the query to the next matching entity.
func (q *Query{{.Suffix}}[{{.TypeVars}}]) Next() bool {
	q.idx++
	if q.idx < q.archSize {
		return true
	}
	return q.nextArchetype()
}

//go:noinline
func (q *Query{{.Suffix}}[{{.TypeVars}}]) nextArchetype() bool {
	for {
		q.matchIdx++
		if q.matchIdx >= len(q.matched) {
			return false
		}
		m := q.matched[q.matchIdx]
		a := m.arch
		if a.size == 0 {
			continue
		}
		{{- range .Components}}
		q.base{{.ZeroIndex}} = a.columns[m.c{{.ZeroIndex}}].data
		{{- end}}
		q.entities = a.entityIDs
		q.archSize = uintptr(a.size)
		q.idx = 0
		return true
	}
}

// Entity returns the current entity in the query.
func (q *Query{{.Suffix}}[{{.TypeVars}}]) Entity() Entity {
	return q.entities[q.idx]
}

// Get returns pointers to {{.TypeVars}} for the current entity.
func (q *Query{{.Suffix}}[{{.TypeVars}}]) Get() ({{.ReturnTypes}}) {
	{{- range .Components}}
	var z{{.ZeroIndex}} {{.TypeName}}
	{{- end}}
	return {{range $i, $e := .Components}}{{if $i}},
		{{end}}(*{{$e.TypeName}})(unsafe.Add(q.base{{$e.ZeroIndex}}, q.idx*unsafe.Sizeof(z{{$e.ZeroIndex}}))){{end}}
}
