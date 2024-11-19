package commander

import (
	"context"
	"fmt"

	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/log"
)

type Commander struct {
	handlers map[string]Handler
}

type Handler interface {
	HandleIt(ctx context.Context, command Command) (any, error)
}

type Command interface {
	Key() string
}

func (commander Commander) WithHandler(handler Handler, zeroCommand Command) Commander {
	if commander.handlers == nil {
		commander.handlers = make(map[string]Handler)
	}

	commander.handlers[zeroCommand.Key()] = handler

	return commander
}

func (commander Commander) Execute(ctx context.Context, command Command) (any, error) {
	if handler, ok := commander.handlers[command.Key()]; ok {
		return handler.HandleIt(ctx, command)
	}

	err := fmt.Errorf("Commander encountered a command for which no handler was configured: %s", command.Key())

	logger := log.FromContext(ctx)
	logger.Error(err)

	return nil, err
}
