package teishoku

// zeroZST provides a non-nil backing address for zero-sized component slices
// when a column's data pointer is nil.
var zeroZST struct{}

// filterBase holds the shared matching state and lazy entity cache across all
// Filter arities (Filter0..Filter8).
type filterBase struct {
	world               *World
	mask                bitmask256
	exclude             bitmask256
	cachedEntities      []Entity
	lastArchIdx         int
	lastMutationVersion uint32
}

func (fb *filterBase) worldPtr() *World {
	return fb.world
}

func (fb *filterBase) matches(a *archetype) bool {
	if fb.mask.isZero() {
		return a.mask.isZero() && !a.mask.intersects(fb.exclude)
	}
	return a.mask.contains(fb.mask) && !a.mask.intersects(fb.exclude)
}

type filterWithWorld interface {
	worldPtr() *World
	addExclude(uint8)
}

// Without adds component type T to the filter's exclusion mask and returns the
// filter for fluent chaining.
func Without[T any, F filterWithWorld](f F) F {
	id := ComponentID[T](f.worldPtr())
	f.addExclude(id)
	return f
}
