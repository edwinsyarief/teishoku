package teishoku

import (
	"fmt"
	"reflect"
	"testing"
)

// EventBus test components
type TestEvent struct {
	Value int
}

func TestEventBusSubscribeAndPublish(t *testing.T) {
	bus := &EventBus{}
	received := 0
	Subscribe(bus, func(e TestEvent) {
		received += e.Value
	})
	Subscribe(bus, func(e TestEvent) {
		received += e.Value * 2
	})
	Publish(bus, TestEvent{Value: 1})
	if received != 3 {
		t.Errorf("expected received 3, got %d", received)
	}
	Publish(bus, TestEvent{Value: 2})
	if received != 3+6 {
		t.Errorf("expected received 9, got %d", received)
	}
}

func TestEventBusMultipleTypes(t *testing.T) {
	bus := &EventBus{}
	received1 := 0
	received2 := 0
	Subscribe(bus, func(e TestEvent) {
		received1 += e.Value
	})
	Subscribe(bus, func(p Position) {
		received2 += int(p.X)
	})
	Publish(bus, TestEvent{Value: 42})
	Publish(bus, Position{X: 10})
	if received1 != 42 {
		t.Errorf("expected received1 42, got %d", received1)
	}
	if received2 != 10 {
		t.Errorf("expected received2 10, got %d", received2)
	}
}

func TestEventBusNoHandlers(t *testing.T) {
	bus := &EventBus{}
	// No panic expected
	Publish(bus, TestEvent{Value: 42})
}

func TestEventBusManySubscribers(t *testing.T) {
	bus := &EventBus{}
	const numSubs = 100
	received := 0
	for i := 0; i < numSubs; i++ {
		Subscribe(bus, func(e TestEvent) {
			received += e.Value
		})
	}
	Publish(bus, TestEvent{Value: 1})
	if received != numSubs {
		t.Errorf("expected %d, got %d", numSubs, received)
	}
}

func TestEventChannel(t *testing.T) {
	t.Run("ZeroHandlers", func(t *testing.T) {
		bus := &EventBus{}
		ch := NewEventChannel[TestEvent](bus)
		// Should not panic with zero handlers
		ch.Publish(TestEvent{Value: 123})
	})

	t.Run("OneHandler", func(t *testing.T) {
		bus := &EventBus{}
		ch := NewEventChannel[TestEvent](bus)
		got := 0
		ch.Subscribe(func(e TestEvent) {
			got += e.Value
		})
		ch.Publish(TestEvent{Value: 7})
		if got != 7 {
			t.Errorf("expected 7, got %d", got)
		}
	})

	t.Run("ManyHandlersAndBusInterop", func(t *testing.T) {
		bus := &EventBus{}
		ch := NewEventChannel[TestEvent](bus)
		order := make([]int, 0, 5)
		ch.Subscribe(func(e TestEvent) { order = append(order, e.Value+1) })
		Subscribe(bus, func(e TestEvent) { order = append(order, e.Value+2) })
		ch.Subscribe(func(e TestEvent) { order = append(order, e.Value+3) })

		ch.Publish(TestEvent{Value: 10})
		Publish(bus, TestEvent{Value: 20})

		want := []int{11, 12, 13, 21, 22, 23}
		if len(order) != len(want) {
			t.Fatalf("expected %d deliveries, got %d (%v)", len(want), len(order), order)
		}
		for i := range want {
			if order[i] != want[i] {
				t.Errorf("delivery %d: expected %d, got %d", i, want[i], order[i])
			}
		}
	})
}

func TestEventBusMaxEventTypesPanic(t *testing.T) {
	bus := &EventBus{}
	for i := 0; i < MaxEventTypes-1; i++ {
		typ := reflect.StructOf([]reflect.StructField{
			{Name: fmt.Sprintf("EventField%d", i), Type: reflect.TypeFor[int]()},
		})
		id := bus.getEventTypeID(typ)
		if int(id) != i {
			t.Fatalf("expected id %d, got %d", i, id)
		}
	}

	// 256th distinct type (ID 255) succeeds via public NewEventChannel
	ch := NewEventChannel[TestEvent](bus)
	called := false
	ch.Subscribe(func(e TestEvent) { called = true })
	ch.Publish(TestEvent{Value: 1})
	if !called {
		t.Fatal("expected 256th event type to publish normally")
	}
	if bus.nextEventTypeID != MaxEventTypes {
		t.Fatalf("expected nextEventTypeID %d, got %d", MaxEventTypes, bus.nextEventTypeID)
	}

	// 257th distinct type must panic rather than silently wrapping uint8 to 0
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when exceeding MaxEventTypes")
		}
		if bus.nextEventTypeID != MaxEventTypes {
			t.Errorf("nextEventTypeID corrupted after panic: got %d", bus.nextEventTypeID)
		}
	}()
	Subscribe(bus, func(Position) {})
}

func TestNewEventBusAndClear(t *testing.T) {
	t.Run("DefaultConstructor", func(t *testing.T) {
		bus := NewEventBus()
		if bus == nil {
			t.Fatal("NewEventBus() returned nil")
		}
		if bus.eventTypeMap == nil {
			t.Error("expected initialized eventTypeMap")
		}
		if bus.defaultHandlerCap != 0 {
			t.Errorf("expected defaultHandlerCap 0, got %d", bus.defaultHandlerCap)
		}
		sum := 0
		Subscribe(bus, func(e TestEvent) { sum += e.Value })
		Publish(bus, TestEvent{Value: 5})
		if sum != 5 {
			t.Errorf("expected sum 5, got %d", sum)
		}
	})

	t.Run("PreSizedCapacityAndClearZeroAllocs", func(t *testing.T) {
		const handlerCap = 16
		bus := NewEventBus(handlerCap)
		if bus == nil {
			t.Fatal("NewEventBus(cap) returned nil")
		}
		if bus.defaultHandlerCap != handlerCap {
			t.Errorf("expected defaultHandlerCap %d, got %d", handlerCap, bus.defaultHandlerCap)
		}

		// Clear on empty bus is safe
		bus.Clear()

		count := 0
		handler := func(e TestEvent) {
			count += e.Value
		}
		posCount := 0
		posHandler := func(p Position) {
			posCount += int(p.X)
		}

		Subscribe(bus, handler)
		Subscribe(bus, handler)
		Subscribe(bus, posHandler)

		// Verify initial handler slice capacity matches handlerCap
		id := bus.eventTypeMap[reflect.TypeFor[TestEvent]()]
		th := bus.handlers[id].(*typedEventHandlers[TestEvent])
		if cap(th.fns) < handlerCap {
			t.Errorf("expected handler slice cap >= %d, got %d", handlerCap, cap(th.fns))
		}

		Publish(bus, TestEvent{Value: 3})
		Publish(bus, Position{X: 4})
		if count != 6 || posCount != 4 {
			t.Fatalf("expected count=6 posCount=4, got count=%d posCount=%d", count, posCount)
		}

		// Clear removes all handlers so Publish invokes 0 handlers
		bus.Clear()
		count = 0
		posCount = 0
		Publish(bus, TestEvent{Value: 10})
		Publish(bus, Position{X: 20})
		if count != 0 || posCount != 0 {
			t.Errorf("expected 0 handler invocations after Clear, got count=%d posCount=%d", count, posCount)
		}

		// Re-subscribing after Clear allocates 0 bytes
		allocs := testing.AllocsPerRun(100, func() {
			bus.Clear()
			for range handlerCap {
				Subscribe(bus, handler)
			}
		})
		if allocs != 0 {
			t.Errorf("expected 0 allocs re-subscribing after Clear, got %v", allocs)
		}

		count = 0
		Publish(bus, TestEvent{Value: 2})
		if count != handlerCap*2 {
			t.Errorf("expected count %d after re-subscribe + Publish, got %d", handlerCap*2, count)
		}
	})
}
