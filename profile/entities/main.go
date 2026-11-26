// Profiling:
// go build ./profile/entities
// go tool pprof -http=":8000" -nodefraction=0.001 ./entities mem.pprof

package main

import (
	"log"
	"os"
	"runtime/pprof"

	"github.com/edwinsyarief/teishoku"
)

type comp1 struct {
	V int64
	W int64
}

type comp2 struct {
	V int64
	W int64
}

func main() {
	count := 50
	iters := 10000
	entities := 1000

	f, err := os.Create("mem.pprof")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	run(count, iters, entities)

	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal(err)
	}
}

func run(rounds, iters, numEntities int) {
	for range rounds {
		w := teishoku.NewWorld(numEntities)
		filter := teishoku.NewFilter2[comp1, comp2](w)
		batch := teishoku.NewBuilder2[comp1, comp2](w)
		entities := make([]teishoku.Entity, 0, numEntities)

		for range iters {
			batch.NewEntities(numEntities)
			entities = entities[:0]
			q := filter.Query()
			for q.Next() {
				entities = append(entities, q.Entity())
				c1, c2 := q.Get()
				c1.V += c2.V
				c1.W += c2.W
			}
			for _, e := range entities {
				w.RemoveEntity(e)
			}
		}
	}
}
