package teishoku

import (
	"reflect"
	"unsafe"
)

// columnOps holds monomorphized operations for a specific component type T.
// This eliminates reflect.MakeSlice overhead and ensures GC write-barrier safety
// for pointer-bearing types while allowing POD types to use fast raw copies.
type columnOps struct {
	elemSize   uintptr
	hasPtrs    bool
	alloc      func(capacity int) (unsafe.Pointer, any)
	grow       func(oldBacking any, curLen, newCap int) (unsafe.Pointer, any)
	copyElem   func(dstBacking any, dstIdx int, srcBacking any, srcIdx int)
	copyRange  func(dstBacking any, dstStart int, srcBacking any, srcStart, count int)
	swapRemove func(backing any, idx, lastIdx int)
	zeroRange  func(backing any, start, end int)
}

// column stores homogeneous component instances for a single archetype.
type column struct {
	data     unsafe.Pointer
	backing  any
	elemSize uintptr
	compID   uint8
	hasPtrs  bool
	ops      *columnOps
}

// typeHasPointers recursively inspects t to determine if it contains any Go
// pointers, strings, slices, maps, channels, functions, or interfaces.
func typeHasPointers(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Ptr, reflect.UnsafePointer, reflect.String, reflect.Slice,
		reflect.Map, reflect.Chan, reflect.Func, reflect.Interface:
		return true
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if typeHasPointers(t.Field(i).Type) {
				return true
			}
		}
		return false
	case reflect.Array:
		if t.Len() == 0 {
			return false
		}
		return typeHasPointers(t.Elem())
	default:
		return false
	}
}

// newColumnOps constructs a monomorphized columnOps descriptor for component type T.
func newColumnOps[T any]() *columnOps {
	var zero T
	size := unsafe.Sizeof(zero)
	hasPtrs := typeHasPointers(reflect.TypeFor[T]())

	return &columnOps{
		elemSize: size,
		hasPtrs:  hasPtrs,
		alloc: func(capacity int) (unsafe.Pointer, any) {
			s := make([]T, capacity)
			h := &s
			return unsafe.Pointer(unsafe.SliceData(s)), h
		},
		grow: func(oldBacking any, curLen, newCap int) (unsafe.Pointer, any) {
			ns := make([]T, newCap)
			if oldBacking != nil {
				h := oldBacking.(*[]T)
				if curLen > 0 {
					copy(ns[:curLen], (*h)[:curLen])
				}
				*h = ns
				return unsafe.Pointer(unsafe.SliceData(ns)), oldBacking
			}
			h := &ns
			return unsafe.Pointer(unsafe.SliceData(ns)), h
		},
		copyElem: func(dstBacking any, dstIdx int, srcBacking any, srcIdx int) {
			(*dstBacking.(*[]T))[dstIdx] = (*srcBacking.(*[]T))[srcIdx]
		},
		copyRange: func(dstBacking any, dstStart int, srcBacking any, srcStart, count int) {
			if count > 0 {
				copy((*dstBacking.(*[]T))[dstStart:dstStart+count], (*srcBacking.(*[]T))[srcStart:srcStart+count])
			}
		},
		swapRemove: func(backing any, idx, lastIdx int) {
			s := *backing.(*[]T)
			if idx < lastIdx {
				s[idx] = s[lastIdx]
			}
			if hasPtrs {
				var z T
				s[lastIdx] = z
			}
		},
		zeroRange: func(backing any, start, end int) {
			if !hasPtrs || start >= end {
				return
			}
			clear((*backing.(*[]T))[start:end])
		},
	}
}

// copyPOD performs an inline-friendly element copy for plain-old-data types
// without Go pointers.
func copyPOD(dst, src unsafe.Pointer, size uintptr) {
	switch size {
	case 0:
		return
	case 1:
		*(*byte)(dst) = *(*byte)(src)
	case 2:
		*(*uint16)(dst) = *(*uint16)(src)
	case 4:
		*(*uint32)(dst) = *(*uint32)(src)
	case 8:
		*(*uint64)(dst) = *(*uint64)(src)
	case 12:
		*(*[3]uint32)(dst) = *(*[3]uint32)(src)
	case 16:
		*(*[2]uint64)(dst) = *(*[2]uint64)(src)
	case 24:
		*(*[3]uint64)(dst) = *(*[3]uint64)(src)
	case 32:
		*(*[4]uint64)(dst) = *(*[4]uint64)(src)
	case 48:
		*(*[6]uint64)(dst) = *(*[6]uint64)(src)
	case 64:
		*(*[8]uint64)(dst) = *(*[8]uint64)(src)
	default:
		copy(unsafe.Slice((*byte)(dst), size), unsafe.Slice((*byte)(src), size))
	}
}
