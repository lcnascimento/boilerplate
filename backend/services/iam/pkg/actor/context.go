package actor

import (
	"context"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor/internal/ctxkey"
)

// FromContext fails with a KindInternal error when no middleware put an Actor in ctx: a route without the middleware is a bug.
func FromContext(ctx context.Context) (*Actor, error) {
	a, ok := ctx.Value(ctxkey.Actor{}).(*Actor)
	if !ok || a == nil {
		return nil, ErrNoActor
	}

	return a, nil
}

func newContext(ctx context.Context, a *Actor) context.Context {
	return context.WithValue(ctx, ctxkey.Actor{}, a)
}
