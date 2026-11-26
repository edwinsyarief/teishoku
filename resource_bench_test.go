package teishoku

import (
	"fmt"
	"testing"
)

var (
	resourceAnySink  any
	resourceBoolSink bool
	resourceIntSink  int
)

func BenchmarkResourcesAdd(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		b.Run(name, func(b *testing.B) {
			_, reses := generateDistinctTypesAndRes(size)
			r := NewResources(size)
			for i := range size {
				r.Add(reses[i])
			}
			r.Clear()
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				r.Clear()
				b.StartTimer()
				for i := range size {
					r.Add(reses[i])
				}
			}
		})
	}
}

func BenchmarkResourcesHas(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		b.Run(name, func(b *testing.B) {
			_, reses := generateDistinctTypesAndRes(size)
			r := &Resources{}
			for i := 0; i < size; i++ {
				r.Add(reses[i])
			}
			b.ReportAllocs()
			for b.Loop() {
				var ok bool
				for i := 0; i < size; i++ {
					ok = r.Has(i)
				}
				resourceBoolSink = ok
			}
		})
	}
}

func BenchmarkResourcesGet(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		b.Run(name, func(b *testing.B) {
			_, reses := generateDistinctTypesAndRes(size)
			r := &Resources{}
			for i := 0; i < size; i++ {
				r.Add(reses[i])
			}
			b.ReportAllocs()
			for b.Loop() {
				var v any
				for i := 0; i < size; i++ {
					v = r.Get(i)
				}
				resourceAnySink = v
			}
		})
	}
}

func BenchmarkResourceHandleGet(b *testing.B) {
	type benchRes struct {
		Value int
	}
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			r := &Resources{}
			AddResource(r, &benchRes{Value: 42})
			handle := UseResource[benchRes](r)
			b.ReportAllocs()
			for b.Loop() {
				sum := 0
				for range size {
					sum += handle.Get().Value
				}
				resourceIntSink = sum
			}
		})
	}
}

func BenchmarkResourcesRemove(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		b.Run(name, func(b *testing.B) {
			_, reses := generateDistinctTypesAndRes(size)
			r := NewResources(size)
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				r.Clear()
				for i := range size {
					r.Add(reses[i])
				}
				b.StartTimer()
				for i := range size {
					r.Remove(i)
				}
			}
		})
	}
}

func BenchmarkResourcesClear(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		b.Run(name, func(b *testing.B) {
			_, reses := generateDistinctTypesAndRes(size)
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				r := &Resources{}
				for j := 0; j < size; j++ {
					r.Add(reses[j])
				}
				b.StartTimer()
				r.Clear()
			}
		})
	}
}
