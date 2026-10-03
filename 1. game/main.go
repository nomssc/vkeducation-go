package main

import (
	"errors"
	"strings"
)

func main() {
	/*
		в этой функции можно ничего не писать,
		но тогда у вас не будет работать через go run main.go
		очень круто будет сделать построчный ввод команд тут, хотя это и не требуется по заданию
	*/

	initGame()

	_ = handleCommand("some command")
}

func initGame() {
	c = &CLI{
		route: NewAction(&ConfigAction{}),
		codec: NewCodec(ConfigCodec{}),
	}
}

/*
 * CLI LAYER
 *
 * Responsible for user
 * input and game output,
 * routing and validation.
 */

// Errors

var (
	ErrUnknown        = errors.New("неизвестная ошибка")
	ErrCommandEmpty   = errors.New("команда не предоставлена")
	ErrUnknownCommand = errors.New("неизвестная команда")
)

// Command Object

type Command struct {
	name string
	args []string
}

// Result Object

type Result struct {
	resp string
}

// CLI the Transport Master

var c *CLI

type Router interface {
	PickHandler(route string) (func(c Command) Result, error)
}

type Serializer interface {
	Unparse(command string) ([]string, error)
}

type CLI struct {
	route Router
	codec Serializer
}

func handleCommand(command string) string {
	cmd, err := c.codec.Unparse(command)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoCommand):
			return ErrCommandEmpty.Error()
		default:
			return ErrUnknown.Error()
		}
	}

	handler, err := c.route.PickHandler(cmd[0])
	if err != nil {
		switch {
		case errors.Is(err, ErrNoRoute):
			return ErrUnknownCommand.Error()
		default:
			return ErrUnknown.Error()
		}
	}

	c := Command{
		name: cmd[0],
		args: cmd[1:],
	}

	return handler(c).resp
}

// Action the Router

var (
	ErrNoRoute = errors.New("no route")
)

type ConfigAction struct {
	Commands map[string]func(c Command) Result
}

type Action struct {
	c map[string]func(c Command) Result
}

func NewAction(c *ConfigAction) *Action {
	return &Action{
		c: c.Commands,
	}
}

func (a *Action) PickHandler(route string) (func(c Command) Result, error) {
	handler, ok := a.c[route]
	if !ok {
		return nil, ErrNoRoute
	}

	return handler, nil
}

// Codec the Input Parser

var (
	ErrNoCommand = errors.New("no command")
)

type ConfigCodec struct{}

type Codec struct{}

func NewCodec(c ConfigCodec) Codec {
	return Codec{}
}

func (c Codec) Unparse(command string) ([]string, error) {
	cmd := strings.Split(command, " ")
	if len(cmd) < 1 {
		return nil, ErrNoCommand
	}

	return cmd, nil
}
