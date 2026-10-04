package main

import (
	"errors"
	"fmt"
	"strings"
)

func main() {
	/*
		в этой функции можно ничего не писать,
		но тогда у вас не будет работать через go run main.go
		очень круто будет сделать построчный ввод команд тут, хотя это и не требуется по заданию
	*/

	initGame()

	_ = handleCommand("завтракать")
}

func initGame() {
	cli = NewCLI(&ConfigCLI{
		&CLI{
			Mux:   NewMux(),
			Codec: NewCodec(),
		},
	})

	cli.Handle("осмотреться", nil)
	cli.Handle("идти", nil)
	cli.Handle("применить", nil)
	cli.Handle("взять", nil)
	cli.Handle("надеть", nil)
}

/*
 * TRANSPORT LAYER
 *
 * Responsible for user
 * input and game output,
 * routing and validation.
 */

/*
 * SERVICE LAYER
 *
 * Processes business logic.
 */

/*
 * CLI PACKAGE
 *
 * Backend backbone.
 */

/* CLI THE TRANSPORT MASTER */

var (
	ErrCLIUnknown              = errors.New("неизвестная ошибка")
	ErrCLINoCommand            = errors.New("команда не предоставлена")
	ErrCLICommandDoesNotExists = errors.New("неизвестная команда")
)

var cli *CLI

// Responsible for storing request
// state throughout handling the
// command.
type Request struct {
	command string
	args    []string
}

func (c Request) Command() string {
	return c.command
}

func (c Request) Args() []string {
	return c.args
}

// Stores responses and used to
// write errors.
type ResponseWriter struct {
	message string
}

func (r *ResponseWriter) Write(s string) error {
	r.message = s

	return nil
}

func (r *ResponseWriter) Error(s string) error {
	return fmt.Errorf("%v", s)
}

// Handler type.
type Handler func(r Request, w *ResponseWriter) error

// Used to configurate CLI on
// inititalization.
type ConfigCLI struct {
	*CLI
}

type Router interface {
	Attach(command string, h Handler)
	Process(command string) (Handler, error)
}

type Interpreter interface {
	UnparseCommandString(s string) ([]string, error)
}

// Responsible for controlling
// program life and managing all
// it's operations.
type CLI struct {
	Mux   Router
	Codec Interpreter
}

func NewCLI(c *ConfigCLI) *CLI {
	return &CLI{
		Mux:   c.CLI.Mux,
		Codec: c.CLI.Codec,
	}
}

func handleCommand(command string) string {
	c, err := cli.Codec.UnparseCommandString(command)
	if err != nil {
		switch {
		case errors.Is(err, ErrCodecNoCommand):
			return ErrCLINoCommand.Error()
		default:
			return ErrCLIUnknown.Error()
		}
	}

	h, err := cli.Mux.Process(c[0])
	if err != nil {
		switch {
		case errors.Is(err, ErrMuxCommandDoesNotExists):
			return ErrCLICommandDoesNotExists.Error()
		default:
			return ErrCLIUnknown.Error()
		}
	}

	r := Request{
		command: c[0],
		args:    c[1:],
	}

	w := &ResponseWriter{}

	err = h(r, w)
	if err != nil {
		return err.Error()
	} else {
		return w.message
	}
}

func (c *CLI) Handle(command string, h Handler) {
	c.Mux.Attach(command, h)
}

/* MUX THE ROUTER */

var (
	ErrMuxCommandDoesNotExists = errors.New("command does not exists")
)

// Used to configurate Mux on
// inititalization.
type ConfigMux struct {
	Mux
}

// Responsible managing
// handlers.
type Mux struct {
	handlers map[string]Handler
}

func NewMux(c ...*ConfigMux) *Mux {
	return &Mux{
		handlers: make(map[string]Handler),
	}
}

func (m *Mux) Attach(command string, h Handler) {
	m.handlers[command] = h
}

func (m *Mux) Process(command string) (Handler, error) {
	handler, ok := m.handlers[command]
	if !ok {
		return nil, ErrMuxCommandDoesNotExists
	}

	return handler, nil
}

/* CODEC THE INPUT PARSER */

var (
	ErrCodecNoCommand = errors.New("no command")
)

// Used to configurate Codec on
// inititalization.
type ConfigCodec struct {
	Codec
}

// Responsible for parsing and
// unparsing inputs.
type Codec struct{}

func NewCodec(c ...ConfigCodec) Codec {
	return Codec{}
}

func (c Codec) UnparseCommandString(s string) ([]string, error) {
	command := strings.Split(s, " ")
	if len(command) < 1 {
		return nil, ErrCodecNoCommand
	}

	return command, nil
}
