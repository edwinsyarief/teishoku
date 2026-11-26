package teishoku

// bitmask256 represents a set of up to 256 component IDs. It is used to
// uniquely identify archetypes. Each bit corresponds to a component ID, and if
// the bit is set, it indicates that the component is present in the archetype.
type bitmask256 [4]uint64

// set enables the bit corresponding to the given component ID.
func (m *bitmask256) set(bit uint8) {
	m[bit>>6] |= uint64(1) << (bit & 63)
}

// unset disables the bit corresponding to the given component ID.
func (m *bitmask256) unset(bit uint8) {
	m[bit>>6] &^= uint64(1) << (bit & 63)
}

// has reports whether the bit corresponding to the given component ID is set.
func (m bitmask256) has(bit uint8) bool {
	return (m[bit>>6] & (uint64(1) << (bit & 63))) != 0
}

// contains checks if all the bits set in the `sub` bitmask are also set in the
// receiver bitmask `m`. This is used to determine if an archetype's component
// set is a superset of a filter's required components.
func (m bitmask256) contains(sub bitmask256) bool {
	return (m[0]&sub[0]) == sub[0] &&
		(m[1]&sub[1]) == sub[1] &&
		(m[2]&sub[2]) == sub[2] &&
		(m[3]&sub[3]) == sub[3]
}

// intersects reports whether `m` and `other` share at least one set bit.
func (m bitmask256) intersects(other bitmask256) bool {
	return ((m[0] & other[0]) | (m[1] & other[1]) | (m[2] & other[2]) | (m[3] & other[3])) != 0
}

// isZero reports whether no bits are set in `m`.
func (m bitmask256) isZero() bool {
	return (m[0] | m[1] | m[2] | m[3]) == 0
}
