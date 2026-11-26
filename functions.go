package teishoku

import "unsafe"

// GetComponent retrieves a pointer to component T on entity e, or nil if e is
// invalid or does not possess T.
func GetComponent[T any](w *World, e Entity) *T {
	meta, ok := w.isValidSlot(e)
	if !ok {
		return nil
	}
	id := ComponentID[T](w)
	c := meta.arch.compIdx[id]
	if c < 0 {
		return nil
	}
	var z T
	return (*T)(unsafe.Add(meta.arch.columns[c].data, uintptr(meta.index)*unsafe.Sizeof(z)))
}

// HasComponent reports whether entity e is valid and possesses component T.
func HasComponent[T any](w *World, e Entity) bool {
	meta, ok := w.isValidSlot(e)
	if !ok {
		return false
	}
	id := ComponentID[T](w)
	return meta.arch.compIdx[id] >= 0
}

// SetComponent adds component T with value val to entity e, or updates it in
// place if e already has component T. Does nothing if e is invalid.
func SetComponent[T any](w *World, e Entity, val T) {
	meta, ok := w.isValidSlot(e)
	if !ok {
		return
	}
	id := ComponentID[T](w)
	a := meta.arch
	c := a.compIdx[id]
	var z T
	if c >= 0 {
		if unsafe.Sizeof(z) > 0 {
			*(*T)(unsafe.Add(a.columns[c].data, uintptr(meta.index)*unsafe.Sizeof(z))) = val
		}
		return
	}
	edge := w.getAddEdge(a, id)
	newIdx := w.moveEntity(e, meta, a, edge)
	dst := edge.target
	dstCol := dst.compIdx[id]
	if unsafe.Sizeof(z) > 0 {
		*(*T)(unsafe.Add(dst.columns[dstCol].data, uintptr(newIdx)*unsafe.Sizeof(z))) = val
	}
}

// RemoveComponent removes component T from entity e. Does nothing if e is
// invalid or does not possess component T.
func RemoveComponent[T any](w *World, e Entity) {
	meta, ok := w.isValidSlot(e)
	if !ok {
		return
	}
	id := ComponentID[T](w)
	a := meta.arch
	if a.compIdx[id] < 0 {
		return
	}
	edge := w.getRemoveEdge(a, id)
	w.moveEntity(e, meta, a, edge)
}
