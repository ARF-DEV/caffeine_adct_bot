package cache

import (
	"context"
	"time"
)

type (
	Cache interface {
		GetAndParse(ctx context.Context, key string, dst interface{}) error
		Set(ctx context.Context, key string, value interface{}) error
		SetExp(ctx context.Context, key string, value interface{}, exp time.Duration) error
		SetExpFunc(ctx context.Context, key string, value interface{}, expFunc ExpFunc) error
		Ping(ctx context.Context) error
	}

	ExpFunc func(ctx context.Context, val interface{}) time.Duration
)
