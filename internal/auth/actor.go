package auth

import (
	"context"

	"warta/internal/model"
)

// Actor adalah pengguna yang sedang memanggil API. Nilai kosongnya berarti
// pengunjung anonim.
type Actor struct {
	ID   int64
	Role model.Role
}

func (a Actor) Authenticated() bool {
	return a.ID > 0
}

func (a Actor) IsAdmin() bool {
	return a.Authenticated() && a.Role == model.RoleAdmin
}

func (a Actor) CanWrite() bool {
	return a.Authenticated() && a.Role.CanWrite()
}

type actorKey struct{}

func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

func ActorFrom(ctx context.Context) Actor {
	a, _ := ctx.Value(actorKey{}).(Actor)
	return a
}
