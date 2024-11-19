package commander

import "context"

type Executor interface {
	Execute(ctx context.Context, command Command) (interface{}, error)
}
