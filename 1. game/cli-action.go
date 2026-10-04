package main

import "errors"

// Action the Router

var (
	ErrNoRoute = errors.New("no route")
)

type ConfigAction struct{}

type Handler func(c *Command) (*Result, error)

type Action struct {
	handlers map[string]Handler
}

func NewAction(c ...*ConfigAction) *Action {
	return &Action{}
}

func (a *Action) Handle(route string, h Handler) {
	a.handlers[route] = h
}

func (a *Action) Handler(c *Command) (Handler, error) {
	h, ok := a.handlers[c.name]
	if !ok {
		return nil, ErrNoRoute
	}

	return h, nil
}
