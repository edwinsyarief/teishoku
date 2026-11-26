// Builder{{.Suffix}} provides a type-safe, zero-allocation API for creating,
// querying, modifying, and removing entities with components {{.TypeVars}}.
type Builder{{.Suffix}}[{{.Types}}] struct {
	world       *World
	arch        *archetype
	lastAddSrc  *archetype
	lastAddEdge *transitionEdge
	lastRemSrc  *archetype
	lastRemEdge *transitionEdge
	mask        bitmask256
	{{- range .Components}}
	id{{.Index}}      uint8
	{{- end}}
	{{- range .Components}}
	archCol{{.Index}} uint8
	{{- end}}
	lastAddCols [{{.N}}]uint8
}

// NewBuilder{{.Suffix}} creates a Builder{{.Suffix}} bound to World w for
// components {{.TypeVars}}.
{{- if gt .N 1}}
// Panics if any component types resolve to the same component ID.
{{- end}}
func NewBuilder{{.Suffix}}[{{.Types}}](w *World) *Builder{{.Suffix}}[{{.TypeVars}}] {
	{{- range .Components}}
	id{{.Index}} := ComponentID[{{.TypeName}}](w)
	{{- end}}
	{{- if gt .N 1}}
	if {{.DuplicateIDs}} {
		panic("teishoku: duplicate component types in Builder{{.Suffix}}")
	}
	{{- end}}
	var mask bitmask256
	{{- range .Components}}
	mask.set(id{{.Index}})
	{{- end}}
	compIDs := []uint8{ {{.IDs}} }
	arch := w.getOrCreateArchetype(mask, compIDs)
	return &Builder{{.Suffix}}[{{.TypeVars}}]{
		world: w,
		arch:  arch,
		mask:  mask,
		{{- range .Components}}
		id{{.Index}}:      id{{.Index}},
		archCol{{.Index}}: uint8(arch.compIdx[id{{.Index}}]),
		{{- end}}
	}
}

// NewEntity creates a single new entity with the builder's components.
func (b *Builder{{.Suffix}}[{{.TypeVars}}]) NewEntity() Entity {
	e, _ := b.world.createEntityInArch(b.arch)
	return e
}

// NewEntityWith creates a single new entity and initializes its components to
// the provided values.
func (b *Builder{{.Suffix}}[{{.TypeVars}}]) NewEntityWith({{.Vars}}) Entity {
	e, row := b.world.createEntityInArch(b.arch)
	{{- range .Components}}
	var z{{.Index}} {{.TypeName}}
	if unsafe.Sizeof(z{{.Index}}) > 0 {
		*(*{{.TypeName}})(unsafe.Add(b.arch.columns[b.archCol{{.Index}}].data, uintptr(row)*unsafe.Sizeof(z{{.Index}}))) = v{{.Index}}
	}
	{{- end}}
	return e
}

// NewEntities creates count new entities with the builder's components.
func (b *Builder{{.Suffix}}[{{.TypeVars}}]) NewEntities(count int) {
	b.world.createEntitiesInArch(b.arch, count)
}

// NewEntitiesWith creates count new entities and initializes all of their
// components to the provided values.
func (b *Builder{{.Suffix}}[{{.TypeVars}}]) NewEntitiesWith(count int, {{.Vars}}) {
	startRow := b.world.createEntitiesInArch(b.arch, count)
	if startRow < 0 {
		return
	}
	{{- range .Components}}
	var z{{.Index}} {{.TypeName}}
	if unsafe.Sizeof(z{{.Index}}) > 0 {
		s{{.Index}} := unsafe.Slice((*{{.TypeName}})(b.arch.columns[b.archCol{{.Index}}].data), startRow+count)[startRow:]
		for i := range s{{.Index}} {
			s{{.Index}}[i] = v{{.Index}}
		}
	}
	{{- end}}
}

// Get returns pointers to the builder's components for entity e, or nils if e
// is invalid or missing any of the components.
func (b *Builder{{.Suffix}}[{{.TypeVars}}]) Get(e Entity) ({{.ReturnTypes}}) {
	w := b.world
	if uint(e.ID) >= uint(len(w.metas)) {
		return {{.ReturnNil}}
	}
	meta := w.metas[e.ID]
	a := meta.arch
	if a == nil || meta.version != e.Version {
		return {{.ReturnNil}}
	}
	{{- range .Components}}
	c{{.Index}} := a.compIdx[b.id{{.Index}}]
	{{- end}}
	if {{.CompIdxCheck}} {
		return {{.ReturnNil}}
	}
	{{- range .Components}}
	var z{{.Index}} {{.TypeName}}
	{{- end}}
	return {{range $i, $e := .Components}}{{if $i}},
		{{end}}(*{{$e.TypeName}})(unsafe.Add(a.columns[c{{$e.Index}}].data, uintptr(meta.index)*unsafe.Sizeof(z{{$e.Index}}))){{end}}
}

// Has reports whether entity e is valid and has all of the builder's
// components.
func (b *Builder{{.Suffix}}[{{.TypeVars}}]) Has(e Entity) bool {
	w := b.world
	if uint(e.ID) >= uint(len(w.metas)) {
		return false
	}
	meta := w.metas[e.ID]
	a := meta.arch
	if a == nil || meta.version != e.Version {
		return false
	}
	return {{range $i, $e := .Components}}{{if $i}} && {{end}}a.compIdx[b.id{{$e.Index}}] >= 0{{end}}
}

// Set adds or updates the builder's components on entity e.
// Does nothing if e is invalid.
func (b *Builder{{.Suffix}}[{{.TypeVars}}]) Set(e Entity, {{.Vars}}) {
	w := b.world
	if uint(e.ID) >= uint(len(w.metas)) {
		return
	}
	meta := &w.metas[e.ID]
	a := meta.arch
	if a == nil || meta.version != e.Version {
		return
	}
	{{- range .Components}}
	c{{.Index}} := a.compIdx[b.id{{.Index}}]
	{{- end}}
	if {{range $i, $e := .Components}}{{if $i}} && {{end}}c{{$e.Index}} >= 0{{end}} {
		{{- range .Components}}
		var z{{.Index}} {{.TypeName}}
		if unsafe.Sizeof(z{{.Index}}) > 0 {
			*(*{{.TypeName}})(unsafe.Add(a.columns[c{{.Index}}].data, uintptr(meta.index)*unsafe.Sizeof(z{{.Index}}))) = v{{.Index}}
		}
		{{- end}}
		return
	}
	var edge *transitionEdge
	if a != b.lastAddSrc {
		{{- if eq .N 1}}
		edge = w.getAddEdge(a, b.id1)
		{{- else}}
		newMask := a.mask
		newMask[0] |= b.mask[0]
		newMask[1] |= b.mask[1]
		newMask[2] |= b.mask[2]
		newMask[3] |= b.mask[3]
		edge = w.getTransitionEdge(a, newMask)
		{{- end}}
		b.lastAddSrc = a
		b.lastAddEdge = edge
		dst := edge.target
		{{- range $i, $e := .Components}}
		b.lastAddCols[{{$i}}] = uint8(dst.compIdx[b.id{{$e.Index}}])
		{{- end}}
	} else {
		edge = b.lastAddEdge
	}
	newIdx := w.moveEntity(e, meta, a, edge)
	dst := edge.target
	{{- range $i, $e := .Components}}
	var z{{.Index}} {{$e.TypeName}}
	if unsafe.Sizeof(z{{.Index}}) > 0 {
		*(*{{$e.TypeName}})(unsafe.Add(dst.columns[b.lastAddCols[{{$i}}]].data, uintptr(newIdx)*unsafe.Sizeof(z{{.Index}}))) = v{{$e.Index}}
	}
	{{- end}}
}

// SetBatch adds or updates the builder's components on every entity in
// entities.
func (b *Builder{{.Suffix}}[{{.TypeVars}}]) SetBatch(entities []Entity, {{.Vars}}) {
	if len(entities) == 0 {
		return
	}
	w := b.world
	for _, e := range entities {
		if uint(e.ID) >= uint(len(w.metas)) {
			continue
		}
		meta := &w.metas[e.ID]
		a := meta.arch
		if a == nil || meta.version != e.Version {
			continue
		}
		{{- range .Components}}
		c{{.Index}} := a.compIdx[b.id{{.Index}}]
		{{- end}}
		if {{range $i, $e := .Components}}{{if $i}} && {{end}}c{{$e.Index}} >= 0{{end}} {
			{{- range .Components}}
			var z{{.Index}} {{.TypeName}}
			if unsafe.Sizeof(z{{.Index}}) > 0 {
				*(*{{.TypeName}})(unsafe.Add(a.columns[c{{.Index}}].data, uintptr(meta.index)*unsafe.Sizeof(z{{.Index}}))) = v{{.Index}}
			}
			{{- end}}
			continue
		}
		var edge *transitionEdge
		if a != b.lastAddSrc {
			{{- if eq .N 1}}
			edge = w.getAddEdge(a, b.id1)
			{{- else}}
			newMask := a.mask
			newMask[0] |= b.mask[0]
			newMask[1] |= b.mask[1]
			newMask[2] |= b.mask[2]
			newMask[3] |= b.mask[3]
			edge = w.getTransitionEdge(a, newMask)
			{{- end}}
			b.lastAddSrc = a
			b.lastAddEdge = edge
			dst := edge.target
			{{- range $i, $e := .Components}}
			b.lastAddCols[{{$i}}] = uint8(dst.compIdx[b.id{{$e.Index}}])
			{{- end}}
		} else {
			edge = b.lastAddEdge
		}
		newIdx := w.moveEntity(e, meta, a, edge)
		dst := edge.target
		{{- range $i, $e := .Components}}
		var z{{.Index}} {{$e.TypeName}}
		if unsafe.Sizeof(z{{.Index}}) > 0 {
			*(*{{$e.TypeName}})(unsafe.Add(dst.columns[b.lastAddCols[{{$i}}]].data, uintptr(newIdx)*unsafe.Sizeof(z{{.Index}}))) = v{{$e.Index}}
		}
		{{- end}}
	}
}

// Remove removes the builder's components from entity e.
// Does nothing if e is invalid or has none of the builder's components.
func (b *Builder{{.Suffix}}[{{.TypeVars}}]) Remove(e Entity) {
	w := b.world
	if uint(e.ID) >= uint(len(w.metas)) {
		return
	}
	meta := &w.metas[e.ID]
	a := meta.arch
	if a == nil || meta.version != e.Version {
		return
	}
	{{- if eq .N 1}}
	if a.compIdx[b.id1] < 0 {
		return
	}
	{{- else}}
	if !a.mask.intersects(b.mask) {
		return
	}
	{{- end}}
	var edge *transitionEdge
	if a != b.lastRemSrc {
		{{- if eq .N 1}}
		edge = w.getRemoveEdge(a, b.id1)
		{{- else}}
		newMask := a.mask
		newMask[0] &^= b.mask[0]
		newMask[1] &^= b.mask[1]
		newMask[2] &^= b.mask[2]
		newMask[3] &^= b.mask[3]
		edge = w.getTransitionEdge(a, newMask)
		{{- end}}
		b.lastRemSrc = a
		b.lastRemEdge = edge
	} else {
		edge = b.lastRemEdge
	}
	w.moveEntity(e, meta, a, edge)
}

// RemoveBatch removes the builder's components from every entity in entities.
func (b *Builder{{.Suffix}}[{{.TypeVars}}]) RemoveBatch(entities []Entity) {
	for _, e := range entities {
		b.Remove(e)
	}
}
