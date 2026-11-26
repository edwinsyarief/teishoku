package teishoku

import (
	"fmt"
	"reflect"
	"testing"
)

func TestResources(t *testing.T) {
	type testStruct1 struct{}
	type testStruct2 struct{}

	t.Run("Add and Get", func(t *testing.T) {
		r := &Resources{}
		res1 := &testStruct1{}
		id := r.Add(res1)
		if id != 0 {
			t.Errorf("expected id 0, got %d", id)
		}
		if got := r.Get(0); got != res1 {
			t.Errorf("expected %v, got %v", res1, got)
		}
	})

	t.Run("Has", func(t *testing.T) {
		r := &Resources{}
		r.Add(&testStruct1{})
		if !r.Has(0) {
			t.Error("expected true")
		}
		if r.Has(1) {
			t.Error("expected false")
		}
		if r.Has(-1) {
			t.Error("expected false")
		}
	})

	t.Run("Add same type panics", func(t *testing.T) {
		r := &Resources{}
		r.Add(&testStruct1{})
		defer func() {
			if recover() == nil {
				t.Error("expected panic")
			}
		}()
		r.Add(&testStruct1{})
	})

	t.Run("Add different types", func(t *testing.T) {
		r := &Resources{}
		r.Add(&testStruct1{})
		id := r.Add(&testStruct2{})
		if id != 1 {
			t.Errorf("expected id 1, got %d", id)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		r := &Resources{}
		id := r.Add(&testStruct1{})
		r.Remove(id)
		if r.Has(id) {
			t.Error("expected false")
		}
		if r.Get(id) != nil {
			t.Error("expected nil")
		}
	})

	t.Run("Add after Remove same type", func(t *testing.T) {
		r := &Resources{}
		id1 := r.Add(&testStruct1{})
		r.Remove(id1)
		id2 := r.Add(&testStruct1{})
		if id2 != id1 {
			t.Errorf("expected reused id %d, got %d", id1, id2)
		}
		if !r.Has(id2) {
			t.Error("expected true")
		}
	})

	t.Run("Add after multiple Removes", func(t *testing.T) {
		r := &Resources{}
		id0 := r.Add(&testStruct1{})
		id1 := r.Add(&testStruct2{})
		r.Remove(id0)
		r.Remove(id1)
		id2 := r.Add(&testStruct1{})
		if id2 != 1 {
			t.Errorf("expected reused id 1, got %d", id2)
		}
		id3 := r.Add(&testStruct2{})
		if id3 != 0 {
			t.Errorf("expected reused id 0, got %d", id3)
		}
	})

	t.Run("Clear", func(t *testing.T) {
		r := &Resources{}
		r.Add(&testStruct1{})
		r.Add(&testStruct2{})
		r.Clear()
		if len(r.items) != 0 {
			t.Error("expected empty")
		}
		if len(r.types) != 0 {
			t.Error("expected empty types")
		}
		if len(r.freeIds) != 0 {
			t.Error("expected empty freeIds")
		}
		if r.Has(0) {
			t.Error("expected false")
		}
	})

	t.Run("Add nil panics", func(t *testing.T) {
		r := &Resources{}
		defer func() {
			if recover() == nil {
				t.Error("expected panic")
			}
		}()
		r.Add(nil)
	})

	t.Run("Remove non-existent", func(t *testing.T) {
		r := &Resources{}
		r.Remove(0) // no panic
	})

	t.Run("Get non-existent", func(t *testing.T) {
		r := &Resources{}
		if r.Get(0) != nil {
			t.Error("expected nil")
		}
	})

	t.Run("Pointers preserved", func(t *testing.T) {
		r := &Resources{}
		res := &testStruct1{}
		id := r.Add(res)
		if got := r.Get(id); got != res {
			t.Errorf("expected same pointer %p, got %p", res, got)
		}
	})
}

func TestNewResources(t *testing.T) {
	type resA struct{ A int }
	type resB struct{ B string }
	type resC struct{ C float64 }

	t.Run("Default", func(t *testing.T) {
		r := NewResources()
		if r == nil {
			t.Fatal("NewResources() returned nil")
		}
		if len(r.items) != 0 || len(r.types) != 0 || len(r.freeIds) != 0 {
			t.Errorf("expected empty container, got items=%d types=%d freeIds=%d", len(r.items), len(r.types), len(r.freeIds))
		}
		a := &resA{A: 7}
		id := r.Add(a)
		if id != 0 || !r.Has(id) || r.Get(id) != a {
			t.Errorf("Add/Has/Get failed on default NewResources(): id=%d got=%v", id, r.Get(id))
		}
		r.Remove(id)
		if r.Has(id) || r.Get(id) != nil {
			t.Error("expected resource removed")
		}
		r.Clear()
		if len(r.items) != 0 || len(r.types) != 0 || len(r.freeIds) != 0 {
			t.Error("expected empty after Clear")
		}
	})

	t.Run("PreSizedCapacity", func(t *testing.T) {
		const initCap = 16
		r := NewResources(initCap)
		if r == nil {
			t.Fatal("NewResources(cap) returned nil")
		}
		if cap(r.items) < initCap {
			t.Errorf("expected cap(items) >= %d, got %d", initCap, cap(r.items))
		}
		if cap(r.freeIds) < initCap {
			t.Errorf("expected cap(freeIds) >= %d, got %d", initCap, cap(r.freeIds))
		}
		if r.types == nil {
			t.Error("expected initialized types map")
		}

		a := &resA{A: 1}
		b := &resB{B: "two"}
		c := &resC{C: 3.5}
		idA := r.Add(a)
		idB := r.Add(b)
		idC := r.Add(c)
		if idA != 0 || idB != 1 || idC != 2 {
			t.Fatalf("unexpected IDs: %d, %d, %d", idA, idB, idC)
		}
		if r.Get(idA) != a || r.Get(idB) != b || r.Get(idC) != c {
			t.Error("Get mismatch on pre-sized Resources")
		}

		r.Remove(idB)
		if r.Has(idB) || r.Get(idB) != nil {
			t.Error("idB should be removed")
		}
		b2 := &resB{B: "reused"}
		idB2 := r.Add(b2)
		if idB2 != idB || r.Get(idB2) != b2 {
			t.Errorf("expected reused slot %d with %p, got %d with %v", idB, b2, idB2, r.Get(idB2))
		}

		r.Clear()
		if len(r.items) != 0 || len(r.types) != 0 || len(r.freeIds) != 0 {
			t.Error("expected empty after Clear")
		}
		if cap(r.items) < initCap || cap(r.freeIds) < initCap {
			t.Error("expected capacity retained after Clear")
		}
		if r.Has(idA) || r.Get(idA) != nil {
			t.Error("expected idA absent after Clear")
		}

		// Re-adding after Clear works cleanly
		idA2 := r.Add(a)
		if idA2 != 0 || r.Get(idA2) != a {
			t.Errorf("re-add after Clear failed: id=%d got=%v", idA2, r.Get(idA2))
		}
	})
}

func TestGenericResourceHelpers(t *testing.T) {
	type gameConfig struct {
		TickRate int
	}
	type audioState struct {
		Volume float32
	}

	r := &Resources{}
	if has, id := HasResource[gameConfig](r); has || id != -1 {
		t.Errorf("expected (false, -1), got (%v, %d)", has, id)
	}
	if got, id := GetResource[gameConfig](r); got != nil || id != -1 {
		t.Errorf("expected (nil, -1), got (%v, %d)", got, id)
	}
	RemoveResource[gameConfig](r) // safe no-op when absent

	cfg := &gameConfig{TickRate: 60}
	id := AddResource(r, cfg)
	if id != 0 {
		t.Errorf("expected id 0, got %d", id)
	}
	if has, gotID := HasResource[gameConfig](r); !has || gotID != id {
		t.Errorf("HasResource mismatch: (%v, %d)", has, gotID)
	}
	if got, gotID := GetResource[gameConfig](r); got != cfg || gotID != id {
		t.Errorf("GetResource mismatch: (%p, %d)", got, gotID)
	}

	// Distinct type
	aud := &audioState{Volume: 0.75}
	audID := AddResource(r, aud)
	if audID != 1 {
		t.Errorf("expected audioState id 1, got %d", audID)
	}

	RemoveResource[gameConfig](r)
	if has, gotID := HasResource[gameConfig](r); has || gotID != -1 {
		t.Errorf("expected (false, -1) after RemoveResource, got (%v, %d)", has, gotID)
	}
	if got, gotID := GetResource[gameConfig](r); got != nil || gotID != -1 {
		t.Errorf("expected (nil, -1) after RemoveResource, got (%v, %d)", got, gotID)
	}
	if got, gotID := GetResource[audioState](r); got != aud || gotID != audID {
		t.Errorf("audioState should remain intact: (%p, %d)", got, gotID)
	}
}

func TestResourceHandle(t *testing.T) {
	type physicsConfig struct {
		Gravity float32
	}
	type renderConfig struct {
		Width int
	}

	// Nil container safety
	nilHandle := UseResource[physicsConfig](nil)
	if nilHandle.Has() || nilHandle.Get() != nil {
		t.Error("UseResource(nil) handle should report Has()==false and Get()==nil")
	}

	r := &Resources{}

	// Deferred Add via ResourceHandle.Add
	hPhys := UseResource[physicsConfig](r)
	if hPhys.Has() || hPhys.Get() != nil {
		t.Error("unregistered handle should report Has()==false and Get()==nil")
	}
	phys := &physicsConfig{Gravity: 9.81}
	hPhys.Add(phys)
	if !hPhys.Has() {
		t.Error("expected hPhys.Has()==true after Add")
	}
	if got := hPhys.Get(); got != phys {
		t.Errorf("expected %p, got %p", phys, got)
	}

	// Pre-registered lookup via UseResource
	hPhys2 := UseResource[physicsConfig](r)
	if !hPhys2.Has() || hPhys2.Get() != phys {
		t.Error("UseResource on existing resource failed")
	}

	// Remove invalidates handle
	RemoveResource[physicsConfig](r)
	if hPhys.Has() || hPhys.Get() != nil {
		t.Error("removed resource handle should report Has()==false and Get()==nil")
	}

	// Slot reuse by a different type must not fool stale handle
	rc := &renderConfig{Width: 1920}
	reusedID := AddResource(r, rc)
	if reusedID != 0 {
		t.Fatalf("expected slot 0 reused, got %d", reusedID)
	}
	if hPhys.Has() || hPhys.Get() != nil {
		t.Error("stale physicsConfig handle must reject reused slot holding renderConfig")
	}
	hRender := UseResource[renderConfig](r)
	if !hRender.Has() || hRender.Get() != rc {
		t.Error("renderConfig handle should return newly added resource")
	}
}

func generateDistinctTypesAndRes(n int) ([]reflect.Type, []any) {
	types := make([]reflect.Type, n)
	res := make([]any, n)
	for i := 0; i < n; i++ {
		fields := []reflect.StructField{
			{Name: fmt.Sprintf("F%d", i), Type: reflect.TypeOf(0)},
		}
		types[i] = reflect.StructOf(fields)
		res[i] = reflect.New(types[i]).Interface()
	}
	return types, res
}
