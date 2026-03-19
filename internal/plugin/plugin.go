package plugin

import "context"

type Plugin interface {
	Name() string
	Run(ctx context.Context) error
}
