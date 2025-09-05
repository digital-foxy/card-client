package operation

import (
	"context"

	"github.com/r3dpixel/toolkit/timestamp"
)

type Mutator[T any] func(T)

type MutationApplier[T any] func(Mutator[T]) error

type Handle[T any] struct {
	opID        ID
	context     context.Context
	timeStarted timestamp.Nano
	applier     func(Mutator[T]) error
	completer   func() error
}

func NewHandle[T any](id ID, ctx context.Context, timeStarted timestamp.Nano, applier MutationApplier[T], completer func() error) Handle[T] {
	return Handle[T]{
		opID:        id,
		timeStarted: timeStarted,
		context:     ctx,
		applier:     applier,
		completer:   completer,
	}
}

func (h *Handle[T]) ID() ID {
	return h.opID
}

func (h *Handle[T]) Context() context.Context {
	return h.context
}

func (h *Handle[T]) Mutate(mutator Mutator[T]) error {
	return h.applier(mutator)
}

func (h *Handle[T]) Complete() error {
	return h.completer()
}

func (h *Handle[T]) TimeStarted() timestamp.Nano {
	return h.timeStarted
}
