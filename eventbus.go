package teishoku

import "reflect"

// MaxEventTypes defines the maximum number of unique event types that can be
// registered in the EventBus. This value is fixed at 256.
const MaxEventTypes = 256

type eventHandlerGroup interface {
	clear()
}

type typedEventHandlers[T any] struct {
	fns []func(T)
}

func (th *typedEventHandlers[T]) clear() {
	clear(th.fns)
	th.fns = th.fns[:0]
}

// EventBus provides a simple, efficient, and type-safe event bus for decoupled
// communication between different parts of an application. It allows systems to
// subscribe to specific event types and publish events to all interested
// listeners without direct dependencies.
//
// The EventBus is designed for high performance, with `Publish` operations being
// allocation-free and avoiding per-handler interface boxing.
type EventBus struct {
	eventTypeMap      map[reflect.Type]uint8
	handlers          [MaxEventTypes]eventHandlerGroup // holds *typedEventHandlers[T]
	nextEventTypeID   uint16
	defaultHandlerCap int
}

// NewEventBus creates a new EventBus with a pre-allocated event type map.
// If a positive handlerCap is provided, newly registered event types pre-allocate
// their handler slices to that capacity (defaulting to 4 otherwise).
func NewEventBus(handlerCap ...int) *EventBus {
	bus := &EventBus{
		eventTypeMap: make(map[reflect.Type]uint8, 16),
	}
	if len(handlerCap) > 0 && handlerCap[0] > 0 {
		bus.defaultHandlerCap = handlerCap[0]
	}
	return bus
}

// Clear removes all subscribed handlers from the EventBus while retaining
// internal backing slices and type registrations so subsequent Subscribe calls
// avoid allocations.
func (bus *EventBus) Clear() {
	for i := uint16(0); i < bus.nextEventTypeID; i++ {
		if bus.handlers[i] != nil {
			bus.handlers[i].clear()
		}
	}
}

// Subscribe registers a handler function to be called when an event of type `T`
// is published. Handlers are stored in the order they are subscribed.
//
// This operation may allocate memory if it's the first time subscribing to a
// particular event type or if the internal handler list needs to be resized.
//
// Parameters:
//   - bus: The EventBus instance to subscribe to.
//   - handler: A function that takes a single argument of type `T`.
func Subscribe[T any](bus *EventBus, handler func(T)) {
	th := getOrCreateEventHandlers[T](bus)
	th.fns = append(th.fns, handler)
}

// Publish broadcasts an event of type `T` to all registered handlers for that
// type. The handlers are called synchronously in the order they were subscribed.
//
// This operation is allocation-free and performs at most one type assertion for
// the handler container.
//
// Parameters:
//   - bus: The EventBus instance to publish to.
//   - event: The event data of type `T` to be sent to handlers.
func Publish[T any](bus *EventBus, event T) {
	if bus.eventTypeMap == nil {
		return
	}
	if id, ok := bus.eventTypeMap[reflect.TypeFor[T]()]; ok {
		if h := bus.handlers[id]; h != nil {
			th := h.(*typedEventHandlers[T])
			for _, fn := range th.fns {
				fn(event)
			}
		}
	}
}

// EventChannel provides zero-lookup, zero-reflection, zero-assertion event
// subscription and publishing for a single event type `T`.
type EventChannel[T any] struct {
	th *typedEventHandlers[T]
}

// NewEventChannel resolves or creates the typed handler container for event
// type `T` on the given EventBus and returns a zero-lookup EventChannel[T].
func NewEventChannel[T any](bus *EventBus) EventChannel[T] {
	return EventChannel[T]{
		th: getOrCreateEventHandlers[T](bus),
	}
}

// Subscribe registers a handler function on this EventChannel.
func (c EventChannel[T]) Subscribe(handler func(T)) {
	c.th.fns = append(c.th.fns, handler)
}

// Publish broadcasts event to all handlers registered for type `T` without
// reflection, map lookups, or type assertions.
func (c EventChannel[T]) Publish(event T) {
	for _, fn := range c.th.fns {
		fn(event)
	}
}

func getOrCreateEventHandlers[T any](bus *EventBus) *typedEventHandlers[T] {
	t := reflect.TypeFor[T]()
	id := bus.getEventTypeID(t)
	if bus.handlers[id] == nil {
		initCap := bus.defaultHandlerCap
		if initCap <= 0 {
			initCap = 4
		}
		th := &typedEventHandlers[T]{
			fns: make([]func(T), 0, initCap),
		}
		bus.handlers[id] = th
		return th
	}
	return bus.handlers[id].(*typedEventHandlers[T])
}

// getEventTypeID retrieves or assigns an ID for the event type.
func (bus *EventBus) getEventTypeID(t reflect.Type) uint8 {
	if bus.eventTypeMap == nil {
		bus.eventTypeMap = make(map[reflect.Type]uint8)
	}
	if id, ok := bus.eventTypeMap[t]; ok {
		return id
	}
	if bus.nextEventTypeID >= MaxEventTypes {
		panic("teishoku: too many event types")
	}
	id := uint8(bus.nextEventTypeID)
	bus.nextEventTypeID++
	bus.eventTypeMap[t] = id
	return id
}
