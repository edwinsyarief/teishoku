package teishoku

import (
	"fmt"
	"testing"
)

var eventBenchSink int

func BenchmarkEventBusSubscribe(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		b.Run(name, func(b *testing.B) {
			bus := NewEventBus(size)
			handler := func(e TestEvent) {
				eventBenchSink += e.Value
			}
			Subscribe(bus, handler)
			bus.Clear()
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				bus.Clear()
				b.StartTimer()
				for range size {
					Subscribe(bus, handler)
				}
			}
		})
	}
}

func BenchmarkEventBusPublishNoHandlers(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			bus := &EventBus{}
			event := TestEvent{Value: 42}
			b.ReportAllocs()
			for b.Loop() {
				for range size {
					Publish(bus, event)
				}
			}
		})
	}
}

func BenchmarkEventBusPublishOneHandler(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			bus := &EventBus{}
			Subscribe(bus, func(e TestEvent) {
				eventBenchSink += e.Value
			})
			event := TestEvent{Value: 42}
			b.ReportAllocs()
			for b.Loop() {
				for range size {
					Publish(bus, event)
				}
			}
		})
	}
}

func BenchmarkEventBusPublishManyHandlers(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		b.Run(name, func(b *testing.B) {
			bus := &EventBus{}
			for range size {
				Subscribe(bus, func(e TestEvent) {
					eventBenchSink += e.Value
				})
			}
			event := TestEvent{Value: 42}
			b.ReportAllocs()
			for b.Loop() {
				Publish(bus, event)
			}
		})
	}
}

func BenchmarkEventChannelPublishOneHandler(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		if size == 1000000 {
			name = "1M"
		}
		b.Run(name, func(b *testing.B) {
			bus := &EventBus{}
			ch := NewEventChannel[TestEvent](bus)
			ch.Subscribe(func(e TestEvent) {
				eventBenchSink += e.Value
			})
			event := TestEvent{Value: 42}
			b.ReportAllocs()
			for b.Loop() {
				for range size {
					ch.Publish(event)
				}
			}
		})
	}
}

func BenchmarkEventChannelPublishManyHandlers(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	for _, size := range sizes {
		name := fmt.Sprintf("%dK", size/1000)
		b.Run(name, func(b *testing.B) {
			bus := &EventBus{}
			ch := NewEventChannel[TestEvent](bus)
			for range size {
				ch.Subscribe(func(e TestEvent) {
					eventBenchSink += e.Value
				})
			}
			event := TestEvent{Value: 42}
			b.ReportAllocs()
			for b.Loop() {
				ch.Publish(event)
			}
		})
	}
}
