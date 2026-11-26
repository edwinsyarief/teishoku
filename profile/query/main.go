// Profiling:
// go build ./profile/query
// go tool pprof -http=":8000" -nodefraction=0.001 ./query mem.pprof

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
		batch.NewEntities(numEntities)

		for range iters {
			filter.EachChunk(func(c1 []comp1, c2 []comp2) {
				for i := range c1 {
					c1[i].V += c2[i].V
					c1[i].W += c2[i].W
				}
			})
		}
	}
}
