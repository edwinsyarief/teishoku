package teishoku

import (
	"fmt"
	"testing"
)

var (
	benchSink     float64
	benchEntsSink []Entity
)

// World Creation Benchmarks
func BenchmarkCreateWorld(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = NewWorld(size)
			}
		})
	}
}

// Expansion Benchmarks
func BenchmarkAutoExpand(b *testing.B) {
	initialSizes := []int{1000, 10000, 100000, 1000000}
	expandMultiplier := 2
	for _, initSize := range initialSizes {
		name := fmt.Sprintf("%dK_init_x%d", initSize/1000, expandMultiplier)
		b.Run(name, func(b *testing.B) {
			targetEntities := initSize * expandMultiplier
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(initSize)
				builder := NewBuilder[Position](w)
				b.StartTimer()
				for range targetEntities {
					builder.NewEntity()
				}
			}
		})
	}
}

// World Entity Creation Benchmarks
func BenchmarkWorldCreateEntity(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				b.StartTimer()
				for range size {
					w.CreateEntity()
				}
			}
		})
	}
}

func BenchmarkWorldCreateEntities(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				b.StartTimer()
				w.CreateEntities(size)
			}
		})
	}
}

// Builder Benchmarks
func BenchmarkBuilderNewEntity(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder := NewBuilder[Position](w)
				b.StartTimer()
				for range size {
					builder.NewEntity()
				}
			}
		})
	}
}

func BenchmarkBuilderNewEntityWith(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			val := Position{1, 2}
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder := NewBuilder[Position](w)
				b.StartTimer()
				for range size {
					builder.NewEntityWith(val)
				}
			}
		})
	}
}

func BenchmarkBuilderNewEntities(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder := NewBuilder[Position](w)
				b.StartTimer()
				builder.NewEntities(size)
			}
		})
	}
}

func BenchmarkBuilderNewEntitiesWith(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			val := Position{1, 2}
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder := NewBuilder[Position](w)
				b.StartTimer()
				builder.NewEntitiesWith(size, val)
			}
		})
	}
}

func BenchmarkBuilderNewEntitiesWith2(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			pos := Position{1, 2}
			vel := Velocity{3, 4}
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder2 := NewBuilder2[Position, Velocity](w)
				b.StartTimer()
				builder2.NewEntitiesWith(size, pos, vel)
			}
		})
	}
}

func BenchmarkBuilderSetComponent(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				ents := make([]Entity, size)
				for j := range size {
					ents[j] = w.CreateEntity()
				}
				builder := NewBuilder[Position](w)
				b.StartTimer()
				for j := range size {
					builder.Set(ents[j], Position{})
				}
			}
		})
	}
}

func BenchmarkBuilderSetComponent2(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				ents := make([]Entity, size)
				for j := range size {
					ents[j] = w.CreateEntity()
				}
				builder := NewBuilder2[Position, Velocity](w)
				b.StartTimer()
				for j := range size {
					builder.Set(ents[j], Position{}, Velocity{})
				}
			}
		})
	}
}

func BenchmarkBuilderRemove(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder2 := NewBuilder2[Position, Velocity](w)
				_ = NewBuilder[Position](w)
				builderVel := NewBuilder[Velocity](w)
				builder2.NewEntities(size)
				ents := make([]Entity, size)
				copy(ents, builder2.arch.entityIDs[:size])
				b.StartTimer()
				for j := range size {
					builderVel.Remove(ents[j])
				}
			}
		})
	}
}

// Component Operation Benchmarks
func BenchmarkBuilderGetComponent(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder := NewBuilder[Position](w)
			builder.NewEntitiesWith(size, Position{X: 1, Y: 2})
			ents := builder.arch.entityIDs[:size]
			b.ReportAllocs()
			for b.Loop() {
				var sum float64
				for j := range size {
					p := builder.Get(ents[j])
					sum += float64(p.X + p.Y)
				}
				benchSink = sum
			}
		})
	}
}

func BenchmarkBuilderGetComponent2(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder := NewBuilder2[Position, Velocity](w)
			builder.NewEntitiesWith(size, Position{X: 1, Y: 2}, Velocity{DX: 3, DY: 4})
			ents := builder.arch.entityIDs[:size]
			b.ReportAllocs()
			for b.Loop() {
				var sum float64
				for j := range size {
					p, v := builder.Get(ents[j])
					sum += float64(p.X + v.DX)
				}
				benchSink = sum
			}
		})
	}
}

func BenchmarkFunctionsGetComponent(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder := NewBuilder[Position](w)
			builder.NewEntitiesWith(size, Position{X: 1, Y: 2})
			ents := builder.arch.entityIDs[:size]
			b.ReportAllocs()
			for b.Loop() {
				var sum float64
				for j := range size {
					p := GetComponent[Position](w, ents[j])
					sum += float64(p.X + p.Y)
				}
				benchSink = sum
			}
		})
	}
}

func BenchmarkFunctionsSetComponentExisting(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder := NewBuilder[Position](w)
			builder.NewEntities(size)
			ents := builder.arch.entityIDs[:size]
			val := Position{1, 2}
			b.ReportAllocs()
			for b.Loop() {
				for j := range size {
					SetComponent(w, ents[j], val)
				}
			}
		})
	}
}

func BenchmarkFunctionsSetComponentNew(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			val := Velocity{3, 4}
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder := NewBuilder[Position](w)
				dummyBuilder := NewBuilder2[Position, Velocity](w)
				dummy := dummyBuilder.NewEntity()
				w.RemoveEntity(dummy)
				builder.NewEntities(size)
				ents := make([]Entity, size)
				copy(ents, builder.arch.entityIDs[:size])
				b.StartTimer()
				for j := range size {
					SetComponent(w, ents[j], val)
				}
			}
		})
	}
}

// Removal & Swap Benchmarks
func BenchmarkFunctionsRemoveComponent(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder2 := NewBuilder2[Position, Velocity](w)
				dummyBuilder := NewBuilder[Position](w)
				dummy := dummyBuilder.NewEntity()
				w.RemoveEntity(dummy)
				builder2.NewEntities(size)
				ents := make([]Entity, size)
				copy(ents, builder2.arch.entityIDs[:size])
				b.StartTimer()
				for j := range size {
					RemoveComponent[Velocity](w, ents[j])
				}
			}
		})
	}
}

func BenchmarkSwapperSwap(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder2 := NewBuilder2[Position, Velocity](w)
				_ = NewBuilder2[Position, Health](w)
				swapper := NewSwapper[Velocity, Health](w)
				builder2.NewEntities(size)
				ents := make([]Entity, size)
				copy(ents, builder2.arch.entityIDs[:size])
				b.StartTimer()
				for j := range size {
					swapper.Swap(ents[j], Health{HP: 100})
				}
			}
		})
	}
}

func BenchmarkWorldRemoveEntity(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder := NewBuilder[Position](w)
				builder.NewEntities(size)
				ents := make([]Entity, size)
				copy(ents, builder.arch.entityIDs[:size])
				b.StartTimer()
				for j := range size {
					w.RemoveEntity(ents[j])
				}
			}
		})
	}
}

func BenchmarkWorldRemoveEntities(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder := NewBuilder[Position](w)
				builder.NewEntities(size)
				ents := make([]Entity, size)
				copy(ents, builder.arch.entityIDs[:size])
				b.StartTimer()
				w.RemoveEntities(ents)
			}
		})
	}
}

func BenchmarkWorldClearEntities(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder := NewBuilder[Position](w)
				builder.NewEntities(size)
				b.StartTimer()
				w.ClearEntities()
			}
		})
	}
}

func BenchmarkFilterRemoveEntities(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder := NewBuilder[Position](w)
				builder.NewEntities(size)
				filter := NewFilter[Position](w)
				b.StartTimer()
				filter.RemoveEntities()
			}
		})
	}
}

func BenchmarkFilter2RemoveEntities(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w := NewWorld(size)
				builder2 := NewBuilder2[Position, Velocity](w)
				builder2.NewEntities(size)
				filter2 := NewFilter2[Position, Velocity](w)
				b.StartTimer()
				filter2.RemoveEntities()
			}
		})
	}
}

// Filter Iteration Benchmarks
func BenchmarkFilterIterate(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder := NewBuilder[Position](w)
			builder.NewEntitiesWith(size, Position{X: 1, Y: 2})
			filter := NewFilter[Position](w)
			b.ReportAllocs()
			for b.Loop() {
				var sum float64
				query := filter.Query()
				for query.Next() {
					p := query.Get()
					p.X += 1
					sum += float64(p.X + p.Y)
				}
				benchSink = sum
			}
		})
	}
}

func BenchmarkFilter2Iterate(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder2 := NewBuilder2[Position, Velocity](w)
			builder2.NewEntitiesWith(size, Position{X: 1, Y: 2}, Velocity{DX: 0.5, DY: 0.5})
			filter2 := NewFilter2[Position, Velocity](w)
			b.ReportAllocs()
			for b.Loop() {
				var sum float64
				query := filter2.Query()
				for query.Next() {
					p, v := query.Get()
					p.X += v.DX
					p.Y += v.DY
					sum += float64(p.X + p.Y)
				}
				benchSink = sum
			}
		})
	}
}

func BenchmarkFilter2EachChunk(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder2 := NewBuilder2[Position, Velocity](w)
			builder2.NewEntitiesWith(size, Position{X: 1, Y: 2}, Velocity{DX: 0.5, DY: 0.5})
			filter2 := NewFilter2[Position, Velocity](w)
			b.ReportAllocs()
			for b.Loop() {
				var sum float64
				filter2.EachChunk(func(pos []Position, vel []Velocity) {
					for i := range pos {
						pos[i].X += vel[i].DX
						pos[i].Y += vel[i].DY
					}
					if len(pos) > 0 {
						sum += float64(pos[0].X)
					}
				})
				benchSink = sum
			}
		})
	}
}

func BenchmarkFilter2Chunks(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder2 := NewBuilder2[Position, Velocity](w)
			builder2.NewEntitiesWith(size, Position{X: 1, Y: 2}, Velocity{DX: 0.5, DY: 0.5})
			filter2 := NewFilter2[Position, Velocity](w)
			b.ReportAllocs()
			for b.Loop() {
				var sum float64
				chunks := filter2.Chunks()
				for chunks.Next() {
					pos, vel := chunks.Get()
					for i := range pos {
						pos[i].X += vel[i].DX
						pos[i].Y += vel[i].DY
					}
					if len(pos) > 0 {
						sum += float64(pos[0].X)
					}
				}
				benchSink = sum
			}
		})
	}
}

func BenchmarkFilter3Iterate(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder3 := NewBuilder3[Position, Velocity, Health](w)
			builder3.NewEntitiesWith(size, Position{X: 1, Y: 2}, Velocity{DX: 0.5, DY: 0.5}, Health{HP: 100})
			filter3 := NewFilter3[Position, Velocity, Health](w)
			b.ReportAllocs()
			for b.Loop() {
				var sum float64
				query := filter3.Query()
				for query.Next() {
					p, v, h := query.Get()
					p.X += v.DX
					sum += float64(p.X) + float64(h.HP)
				}
				benchSink = sum
			}
		})
	}
}

func BenchmarkFilter4Iterate(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder4 := NewBuilder4[Position, Velocity, Health, WithPointer](w)
			builder4.NewEntitiesWith(size, Position{X: 1, Y: 2}, Velocity{DX: 0.5, DY: 0.5}, Health{HP: 100}, WithPointer{})
			filter4 := NewFilter4[Position, Velocity, Health, WithPointer](w)
			b.ReportAllocs()
			for b.Loop() {
				var sum float64
				query := filter4.Query()
				for query.Next() {
					p, v, h, wp := query.Get()
					p.X += v.DX
					sum += float64(p.X) + float64(h.HP)
					if wp.Data != nil {
						sum += float64(*wp.Data)
					}
				}
				benchSink = sum
			}
		})
	}
}

func BenchmarkFilter5Iterate(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder5 := NewBuilder5[Position, Velocity, Health, WithPointer, Dummy1](w)
			builder5.NewEntitiesWith(size, Position{X: 1, Y: 2}, Velocity{DX: 0.5, DY: 0.5}, Health{HP: 100}, WithPointer{}, Dummy1{Val: 1})
			filter5 := NewFilter5[Position, Velocity, Health, WithPointer, Dummy1](w)
			b.ReportAllocs()
			for b.Loop() {
				var sum float64
				query := filter5.Query()
				for query.Next() {
					p, v, h, wp, d1 := query.Get()
					p.X += v.DX
					sum += float64(p.X) + float64(h.HP) + float64(d1.Val)
					if wp.Data != nil {
						sum += float64(*wp.Data)
					}
				}
				benchSink = sum
			}
		})
	}
}

func BenchmarkFilter6Iterate(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder6 := NewBuilder6[Position, Velocity, Health, WithPointer, Dummy1, Dummy2](w)
			builder6.NewEntitiesWith(size, Position{X: 1, Y: 2}, Velocity{DX: 0.5, DY: 0.5}, Health{HP: 100}, WithPointer{}, Dummy1{Val: 1}, Dummy2{Val: 2})
			filter6 := NewFilter6[Position, Velocity, Health, WithPointer, Dummy1, Dummy2](w)
			b.ReportAllocs()
			for b.Loop() {
				var sum float64
				query := filter6.Query()
				for query.Next() {
					p, v, h, wp, d1, d2 := query.Get()
					p.X += v.DX
					sum += float64(p.X) + float64(h.HP) + float64(d1.Val) + float64(d2.Val)
					if wp.Data != nil {
						sum += float64(*wp.Data)
					}
				}
				benchSink = sum
			}
		})
	}
}

func BenchmarkFilterGetEntitiesCached(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder := NewBuilder[Position](w)
			builder.NewEntities(size)
			filter := NewFilter[Position](w)
			b.ReportAllocs()
			for b.Loop() {
				benchEntsSink = filter.Entities()
			}
		})
	}
}

func BenchmarkFilterGetEntitiesUncached(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			w := NewWorld(size)
			builder := NewBuilder[Position](w)
			filter := NewFilter[Position](w)
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w.ClearEntities()
				builder.NewEntities(size)
				b.StartTimer()
				benchEntsSink = filter.Entities()
			}
		})
	}
}
