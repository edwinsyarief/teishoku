package teishoku

import "reflect"

// Resources provides a type-safe, high-performance container for managing
// global, singleton-like objects. It ensures that only one resource of each
// type exists at any given time.
//
// Resources is designed for single-threaded, lock-free access matching World.
// Operations like Add, Get, Has, and Remove are O(1) and minimize allocations,
// and ResourceHandle[T] provides zero-lookup, zero-reflection hot-path access.
type Resources struct {
	items   []any
	types   map[reflect.Type]int
	freeIds []int
	version uint32
}

// NewResources creates a new Resources container. If a positive initialCapacity
// is provided, the internal slices and type map are pre-allocated to avoid
// growth allocations during initial setup.
func NewResources(initialCapacity ...int) *Resources {
	if len(initialCapacity) > 0 && initialCapacity[0] > 0 {
		capHint := initialCapacity[0]
		return &Resources{
			items:   make([]any, 0, capHint),
			types:   make(map[reflect.Type]int, capHint),
			freeIds: make([]int, 0, capHint),
		}
	}
	return &Resources{}
}

// Add stores a new resource. It panics if a resource of the same type has
// already been added or if the provided resource is nil.
//
// This operation reuses slots from previously removed resources to prevent
// unnecessary allocations.
//
// Parameters:
//   - res: The resource object to add. Must be a non-nil pointer.
//
// Returns:
//   - The unique integer ID assigned to this resource.
func (r *Resources) Add(res any) int {
	if res == nil {
		panic("cannot add nil resource")
	}
	v := reflect.ValueOf(res)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		panic("cannot add nil resource")
	}
	t := v.Type()
	if r.types == nil {
		r.types = make(map[reflect.Type]int)
	}
	if _, ok := r.types[t]; ok {
		panic("resource of the same type already exists")
	}
	var id int
	if len(r.freeIds) > 0 {
		id = r.freeIds[len(r.freeIds)-1]
		r.freeIds = r.freeIds[:len(r.freeIds)-1]
		r.items[id] = res
	} else {
		r.items = append(r.items, res)
		id = len(r.items) - 1
	}
	r.types[t] = id
	return id
}

// Has checks if a resource with the given ID is currently stored.
//
// Parameters:
//   - id: The ID of the resource to check.
//
// Returns:
//   - true if the resource exists, false otherwise.
func (r *Resources) Has(id int) bool {
	return id >= 0 && id < len(r.items) && r.items[id] != nil
}

// Get retrieves a resource by its ID. It returns nil if no resource is found
// with the specified ID.
//
// Parameters:
//   - id: The ID of the resource to retrieve.
//
// Returns:
//   - The resource as an `any` type, or nil if not found.
func (r *Resources) Get(id int) any {
	if id >= 0 && id < len(r.items) && r.items[id] != nil {
		return r.items[id]
	}
	return nil
}

// Remove deletes a resource by its ID. If the resource exists, its ID is
// recycled for future use. If the ID is invalid, the operation does nothing.
//
// Parameters:
//   - id: The ID of the resource to remove.
func (r *Resources) Remove(id int) {
	if id < 0 || id >= len(r.items) || r.items[id] == nil {
		return
	}
	res := r.items[id]
	t := reflect.TypeOf(res)
	delete(r.types, t)
	r.items[id] = nil
	r.freeIds = append(r.freeIds, id)
	r.version++
}

// Clear removes all resources and resets the container to its initial state.
// This is a fast operation that avoids re-allocating the internal maps and slices.
func (r *Resources) Clear() {
	clear(r.items)
	r.items = r.items[:0]
	clear(r.types)
	r.freeIds = r.freeIds[:0]
	r.version++
}

// AddResource is a generic helper function that stores a resource of type `*T`
// in the container and returns its assigned integer ID.
func AddResource[T any](r *Resources, res *T) int {
	return r.Add(res)
}

// HasResource is a generic helper function that checks if a resource of type `T`
// exists in the container.
//
// Parameters:
//   - r: The Resources container to check.
//
// Returns:
//   - A boolean indicating if the resource was found, and its integer ID. If
//     not found, returns (false, -1).
func HasResource[T any](r *Resources) (bool, int) {
	t := reflect.TypeFor[*T]()
	if id, ok := r.types[t]; ok {
		return true, id
	}
	return false, -1
}

// GetResource is a generic helper function that retrieves a resource of type `T`.
//
// Parameters:
//   - r: The Resources container to query.
//
// Returns:
//   - A pointer to the resource of type `T` and its integer ID. If not found,
//     returns (nil, -1).
func GetResource[T any](r *Resources) (*T, int) {
	t := reflect.TypeFor[*T]()
	if id, ok := r.types[t]; ok {
		if v := r.items[id]; v != nil {
			return v.(*T), id
		}
	}
	return nil, -1
}

// RemoveResource is a generic helper function that removes a resource of type `T`
// from the container if present.
func RemoveResource[T any](r *Resources) {
	t := reflect.TypeFor[*T]()
	if id, ok := r.types[t]; ok {
		r.Remove(id)
	}
}

// ResourceHandle provides zero-lookup, zero-reflection cached access to a
// single resource of type `*T` stored in a Resources container.
type ResourceHandle[T any] struct {
	res *Resources
	ptr *T
	id  int
	ver uint32
}

// UseResource resolves the slot ID for resource type `*T` once and returns a
// ResourceHandle[T] for direct O(1) cached access. If the resource is not yet
// registered, the handle stores id = -1 until Add is called.
func UseResource[T any](r *Resources) ResourceHandle[T] {
	if r == nil {
		return ResourceHandle[T]{id: -1}
	}
	if id, ok := r.types[reflect.TypeFor[*T]()]; ok {
		var ptr *T
		if v := r.items[id]; v != nil {
			ptr, _ = v.(*T)
		}
		return ResourceHandle[T]{res: r, ptr: ptr, id: id, ver: r.version}
	}
	return ResourceHandle[T]{res: r, id: -1}
}

// Get returns the cached `*T` resource directly without reflection or map
// lookups, or nil if the resource is not registered or was removed.
func (h ResourceHandle[T]) Get() *T {
	if h.ptr != nil && h.ver == h.res.version {
		return h.ptr
	}
	return h.getSlow()
}

//go:noinline
func (h ResourceHandle[T]) getSlow() *T {
	if h.res != nil && h.id >= 0 && h.id < len(h.res.items) {
		if v := h.res.items[h.id]; v != nil {
			if typed, ok := v.(*T); ok {
				return typed
			}
		}
	}
	return nil
}

// Has reports whether the cached resource is currently present in the container.
func (h ResourceHandle[T]) Has() bool {
	if h.ptr != nil && h.ver == h.res.version {
		return true
	}
	return h.getSlow() != nil
}

// Add registers val in the underlying Resources container and caches its slot ID
// and direct pointer.
func (h *ResourceHandle[T]) Add(val *T) {
	h.id = h.res.Add(val)
	h.ptr = val
	h.ver = h.res.version
}
