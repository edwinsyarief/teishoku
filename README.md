# Teishoku ECS

![Go Version](https://img.shields.io/badge/Go-1.26+-blue?logo=go&style=flat&logoColor=white)
![Go Reference](https://pkg.go.dev/badge/github.com/edwinsyarief/teishoku.svg)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/edwinsyarief/teishoku/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/edwinsyarief/teishoku)](https://goreportcard.com/report/github.com/edwinsyarief/teishoku)

`Teishoku` is a **zero-dependency**, high-performance, archetype-based Entity Component System (ECS) library for Go.

Designed for games, real-time simulations, and data-oriented workloads, `Teishoku` pairs a clean, cohesive generic API (`World`, `Builder`, `Filter`, `Swapper`, `Resources`, `EventBus`) with Go-native systems optimizations:

- **Zero External Dependencies**: Pure Go standard library only.
- **Monomorphized Typed Column Kernels**: Component storage uses compile-time generic column ops (`alloc`/`grow`/`copy`/`zero`) — zero `reflect.MakeSlice` on archetype growth, zeroing skipped for POD structs, and full GC write-barrier safety for pointer-bearing components.
- **16-Byte Direct-Pointer Entity Slots**: Each entity slot stores a direct `*archetype` pointer, row index, and per-slot generation counter (4 slots per 64-byte cache line; no table-array indirection).
- **Precomputed Transition Copy-Plans + Inline Monomorphic Cache**: Structural changes (`Set`, `Remove`, `Swap`) cache `(lastSrc -> transitionEdge)` directly on `Builder`/`Swapper` with pre-matched surviving column pairs.
- **Compile-Time Constant Strides & `EachChunk` Slice Iteration**: `Query` uses compile-time `unsafe.Sizeof` strides with pre-resolved column indices, while `EachChunk` / `Chunks` expose contiguous typed component slices directly at native Go slice speed (~0.73 ns/entity).
- **Batteries Included**: Built-in typed `Resources` (with `ResourceHandle[T]`) and typed `EventBus` (with zero-lookup `EventChannel[T]`).

---

## Getting Started

### Installation

```bash
go get github.com/edwinsyarief/teishoku
```

### Complete Example

```go
package main

import (
	"fmt"
	"github.com/edwinsyarief/teishoku"
)

type Position struct {
	X, Y float64
}

type Velocity struct {
	VX, VY float64
}

type Static struct{}

func main() {
	// 1. Create a World (optional initial capacity hint).
	world := teishoku.NewWorld(10000)

	// 2. Spawn entities with a Builder (supports 1 to 8 components).
	builder := teishoku.NewBuilder2[Position, Velocity](world)
	builder.NewEntitiesWith(100, Position{X: 0, Y: 0}, Velocity{VX: 1.0, VY: 0.5})

	// Spawn a single initialized entity and inspect it.
	hero := builder.NewEntityWith(Position{X: 10, Y: 20}, Velocity{VX: 2.0, VY: -1.0})

	// 3. Query with a Filter (supports optional Without exclusions).
	movement := teishoku.Without[Static](
		teishoku.NewFilter2[Position, Velocity](world),
	)

	// Ultra-fast bulk slice iteration (~0.73 ns/entity, 0 allocs):
	for range 5 {
		movement.EachChunk(func(pos []Position, vel []Velocity) {
			for i := range pos {
				pos[i].X += vel[i].VX
				pos[i].Y += vel[i].VY
			}
		})
	}

	// Or row-by-row iteration (supports nested queries, 0 allocs):
	q := movement.Query()
	for q.Next() {
		pos, vel := q.Get()
		pos.X += vel.VX
		pos.Y += vel.VY
	}

	// 4. Direct component access via Builder or 1-component helpers.
	heroPos, _ := builder.Get(hero)
	fmt.Printf("Hero final position: %+v\n", *heroPos)
}
```

---

## API Reference

### World

| Function / Method | Description |
| :--- | :--- |
| `NewWorld(initialCapacity ...int) *World` | Creates a new `World` with an optional entity capacity hint (default `1024`). |
| `ComponentID[T any](w *World) uint8` | Returns (and registers if needed) the component type ID for `T`. |
| `(w *World) CreateEntity() Entity` | Creates an empty entity. |
| `(w *World) CreateEntities(count int)` | Creates `count` empty entities in a batch. |
| `(w *World) RemoveEntity(e Entity)` | Removes an entity and recycles its ID. |
| `(w *World) RemoveEntities(ents []Entity)` | Removes a slice of entities in a batch. |
| `(w *World) ClearEntities()` | Removes all entities while preserving allocated archetype capacity. |
| `(w *World) IsValid(e Entity) bool` | Reports whether `e` is currently alive. |
| `(w *World) Len() int` | Returns the number of living entities. |
| `(w *World) Resources() *Resources` | Returns the world's `Resources` container. |

### Builders (`Builder` .. `Builder8`)

Unified, pre-cached handles for spawning, reading, updating, and removing sets of 1 to 8 components (`NewBuilder`, `NewBuilder2` .. `NewBuilder8`).

| Method | Description |
| :--- | :--- |
| `NewEntity() Entity` | Spawns one zero-initialized entity in the builder's archetype. |
| `NewEntityWith(v1, ...) Entity` | Spawns one entity and initializes its components in a single step. |
| `NewEntities(count int)` | Batch-spawns `count` zero-initialized entities. |
| `NewEntitiesWith(count int, v1, ...)` | Batch-spawns `count` entities initialized to `v1, ...`. |
| `Get(e Entity) (*T1, ...)` | Returns pointers to the components, or `nil`s if invalid/missing. |
| `Has(e Entity) bool` | Reports whether `e` is alive and possesses all builder components. |
| `Set(e Entity, v1, ...)` | Updates components in-place, or migrates `e` via inline-cached copy-plan. |
| `SetBatch(ents []Entity, v1, ...)` | Batch `Set` across a slice of entities. |
| `Remove(e Entity)` | Removes the builder's components from `e` via inline-cached copy-plan. |
| `RemoveBatch(ents []Entity)` | Batch `Remove` across a slice of entities. |

### Single-Component Helpers & Swapper

| Function / Type | Description |
| :--- | :--- |
| `GetComponent[T](w, e) *T` | Convenience single-component lookup. |
| `HasComponent[T](w, e) bool` | Convenience single-component presence check. |
| `SetComponent[T](w, e, val)` | Convenience single-component add/update. |
| `RemoveComponent[T](w, e)` | Convenience single-component removal. |
| `NewSwapper[RemT, AddT](w)` | Replaces `RemT` with `AddT` in a single archetype transition (`Swap`, `SwapBatch`). |

### Filters & Queries (`Filter0`, `Filter` .. `Filter8`)

| Method / Function | Description |
| :--- | :--- |
| `NewFilter[T](w)` / `NewFilter2..8(w)` | Creates an incrementally synced filter for 1 to 8 components (`NewFilter0` for empty entities). |
| `Without[T](filter)` / `filter.Without(ids...)` | Excludes entities possessing the given component(s) at zero iteration cost. |
| `filter.EachChunk(func([]T1, ...))` | Invokes callback once per non-empty matching archetype with direct typed slices. |
| `filter.Chunks()` | Returns a `Chunks` cursor (`Next()`, `Get()`, `Entities()`) for slice-level iteration with early `break`. |
| `filter.Query()` | Returns a stack-allocated `Query` cursor (`Next()`, `Get()`, `Entity()`) supporting nested iteration. |
| `filter.Count() int` | Returns the total number of matching entities. |
| `filter.Entities() []Entity` | Returns a lazily cached slice of all matching entities. |
| `filter.RemoveEntities()` | Bulk-destroys all matching entities in O(N) without swap-remove copies. |

### Resources & EventBus

- **Resources**: `NewResources(cap...)`, `AddResource[T]`, `HasResource[T]`, `GetResource[T]`, `RemoveResource[T]`, `Clear()`, and zero-lookup cached `UseResource[T](r)` -> `ResourceHandle[T]` (`Get`, `Has`, `Add`).
- **EventBus**: `NewEventBus(handlerCap...)`, `Clear()`, typed `Subscribe[T]`, `Publish[T]`, and zero-lookup `NewEventChannel[T](bus)` -> `EventChannel[T]` (`Subscribe`, `Publish`).

### Concurrency Model

Like other high-performance archetype ECS engines, `World`, `Resources`, and `EventBus` are designed for **single-threaded mutations** (lock-free hot paths). Concurrent read-only `Query`/`Chunks`/`EachChunk` iterations are safe as long as no structural mutations occur simultaneously.

---

## Code Generation

Multi-component `Builder` and `Filter`/`Query`/`Chunks` implementations (arities 1 to 8) are generated from templates in `templates/`:

```bash
go generate ./...
```

---

## Benchmarks

Run the native benchmark suite locally:

```bash
go test -bench=. -benchmem -run=^$ .
```

The tables below show the `/1K` suite (`go test -bench=/1K -benchtime=200ms -benchmem -run=^$ .`), measured on `11th Gen Intel(R) Core(TM) i7-1185G7 @ 3.00GHz` (`windows/amd64`, Go `1.27.4`). Only `BenchmarkCreateWorld` and `BenchmarkAutoExpand` perform heap allocations; every other benchmark runs at **`0 B/op, 0 allocs/op`**.

> **Note on batch vs. single-op measurements:** Most `/1K` benchmarks measure a **1,000-entity** or **1,000-op batch** per `b.N` iteration (e.g., creating, updating, querying, or publishing 1,000 items). Benchmarks that perform a **single op** per `b.N` iteration are explicitly noted below (`CreateWorld`, `FilterGetEntitiesCached`, `PublishManyHandlers` across 1,000 subscribers, and `ResourcesClear` / `FilterRemoveEntities` / `WorldClearEntities` single-call batch operations).

### Entity, World, Builder, Filter, Functions & Swapper

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `BenchmarkCreateWorld/1K` *(single op, cap 1,000)* | 5,609 | 36,864 | 6 |
| `BenchmarkAutoExpand/1K_init_x2` *(2,000-op expansion)* | 21,792 | 98,328 | 5 |
| `BenchmarkWorldCreateEntity/1K` *(1,000-op batch)* | 4,935 | 0 | 0 |
| `BenchmarkWorldCreateEntities/1K` *(1,000-entity batch)* | 2,925 | 0 | 0 |
| `BenchmarkBuilderNewEntity/1K` *(1,000-op batch)* | 5,420 | 0 | 0 |
| `BenchmarkBuilderNewEntityWith/1K` *(1,000-op batch)* | 7,279 | 0 | 0 |
| `BenchmarkBuilderNewEntities/1K` *(1,000-entity batch)* | 2,953 | 0 | 0 |
| `BenchmarkBuilderNewEntitiesWith/1K` *(1,000-entity batch)* | 3,539 | 0 | 0 |
| `BenchmarkBuilderNewEntitiesWith2/1K` *(1,000-entity batch)* | 4,000 | 0 | 0 |
| `BenchmarkBuilderSetComponent/1K` *(1,000-op batch)* | 11,080 | 0 | 0 |
| `BenchmarkBuilderSetComponent2/1K` *(1,000-op batch)* | 12,603 | 0 | 0 |
| `BenchmarkBuilderRemove/1K` *(1,000-op batch)* | 17,984 | 0 | 0 |
| `BenchmarkBuilderGetComponent/1K` *(1,000-op batch)* | 2,298 | 0 | 0 |
| `BenchmarkBuilderGetComponent2/1K` *(1,000-op batch)* | 3,688 | 0 | 0 |
| `BenchmarkFunctionsGetComponent/1K` *(1,000-op batch)* | 16,238 | 0 | 0 |
| `BenchmarkFunctionsSetComponentExisting/1K` *(1,000-op batch)* | 16,512 | 0 | 0 |
| `BenchmarkFunctionsSetComponentNew/1K` *(1,000-op batch)* | 35,445 | 0 | 0 |
| `BenchmarkFunctionsRemoveComponent/1K` *(1,000-op batch)* | 35,487 | 0 | 0 |
| `BenchmarkSwapperSwap/1K` *(1,000-op batch)* | 18,526 | 0 | 0 |
| `BenchmarkWorldRemoveEntity/1K` *(1,000-op batch)* | 8,660 | 0 | 0 |
| `BenchmarkWorldRemoveEntities/1K` *(1,000-entity batch)* | 7,475 | 0 | 0 |
| `BenchmarkWorldClearEntities/1K` *(single call, 1,000 entities)* | 1,777 | 0 | 0 |
| `BenchmarkFilterRemoveEntities/1K` *(single call, 1,000 entities)* | 1,824 | 0 | 0 |
| `BenchmarkFilter2RemoveEntities/1K` *(single call, 1,000 entities)* | 1,949 | 0 | 0 |
| `BenchmarkFilterIterate/1K` *(1,000-entity batch)* | 2,280 | 0 | 0 |
| `BenchmarkFilter2Iterate/1K` *(1,000-entity batch)* | 2,897 | 0 | 0 |
| `BenchmarkFilter2EachChunk/1K` *(1,000-entity batch)* | 890.3 | 0 | 0 |
| `BenchmarkFilter2Chunks/1K` *(1,000-entity batch)* | 1,536 | 0 | 0 |
| `BenchmarkFilter3Iterate/1K` *(1,000-entity batch)* | 2,734 | 0 | 0 |
| `BenchmarkFilter4Iterate/1K` *(1,000-entity batch)* | 3,181 | 0 | 0 |
| `BenchmarkFilter5Iterate/1K` *(1,000-entity batch)* | 4,023 | 0 | 0 |
| `BenchmarkFilter6Iterate/1K` *(1,000-entity batch)* | 3,721 | 0 | 0 |
| `BenchmarkFilterGetEntitiesCached/1K` *(single op, 1,000 entities)* | 2.437 | 0 | 0 |
| `BenchmarkFilterGetEntitiesUncached/1K` *(single call, 1,000 entities)* | 527.4 | 0 | 0 |

### EventBus

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `BenchmarkEventBusSubscribe/1K` *(1,000-op batch)* | 21,618 | 0 | 0 |
| `BenchmarkEventBusPublishNoHandlers/1K` *(1,000-op batch)* | 1,594 | 0 | 0 |
| `BenchmarkEventBusPublishOneHandler/1K` *(1,000-op batch)* | 19,379 | 0 | 0 |
| `BenchmarkEventBusPublishManyHandlers/1K` *(single op, 1,000 handlers)* | 1,725 | 0 | 0 |
| `BenchmarkEventChannelPublishOneHandler/1K` *(1,000-op batch)* | 2,481 | 0 | 0 |
| `BenchmarkEventChannelPublishManyHandlers/1K` *(single op, 1,000 handlers)* | 1,771 | 0 | 0 |

### Resources

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `BenchmarkResourcesAdd/1K` *(1,000-op batch)* | 42,398 | 0 | 0 |
| `BenchmarkResourcesHas/1K` *(1,000-op batch)* | 777.8 | 0 | 0 |
| `BenchmarkResourcesGet/1K` *(1,000-op batch)* | 740.1 | 0 | 0 |
| `BenchmarkResourceHandleGet/1K` *(1,000-op batch)* | 859.7 | 0 | 0 |
| `BenchmarkResourcesRemove/1K` *(1,000-op batch)* | 32,615 | 0 | 0 |
| `BenchmarkResourcesClear/1K` *(single call, 1,000 resources)* | 7,297 | 0 | 0 |

---

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.
