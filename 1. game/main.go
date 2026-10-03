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

type Activity string

var (
	Lookout Activity = "осмотреться"
	Walk    Activity = "идти"
	Use     Activity = "применить"
	Take    Activity = "взять"
	PutOn   Activity = "надеть"
)

func initGame() {
	a := NewAction()
	a.Handle(string(Lookout), nil)
	a.Handle(string(Walk), nil)
	a.Handle(string(Use), nil)
	a.Handle(string(Take), nil)
	a.Handle(string(PutOn), nil)

	cli = NewCLI(&ConfigCLI{
		Route: a,
		Codec: NewCodec(),
	})
}

/*
 * CLI PACKAGE
 *
 * Backend backbone.
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

var cli *CLI

type ConfigCLI struct {
	Route Router
	Codec Codec
}

type Router interface {
	Handle(route string, h Handler)
	Handler(c *Command) (Handler, error)
}

type Serializer interface {
	UnparseCommandString(command string) ([]string, error)
}

type CLI struct {
	route Router
	codec Serializer
}

func NewCLI(c *ConfigCLI) *CLI {
	return &CLI{
		route: c.Route,
		codec: c.Codec,
	}
}

func handleCommand(command string) string {
	cmd, err := cli.codec.UnparseCommandString(command)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoCommand):
			return ErrCommandEmpty.Error()
		default:
			return ErrUnknown.Error()
		}
	}

	c := &Command{
		name: cmd[0],
		args: cmd[1:],
	}

	h, err := cli.route.Handler(c)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoRoute):
			return ErrUnknownCommand.Error()
		default:
			return ErrUnknown.Error()
		}
	}

	result, err := h(c)
	if err != nil {
		return err.Error()
	} else {
		return result.resp
	}
}

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

// Codec the Input Parser

var (
	ErrNoCommand = errors.New("no command")
)

type ConfigCodec struct{}

type Codec struct{}

func NewCodec(c ...ConfigCodec) Codec {
	return Codec{}
}

func (c Codec) UnparseCommandString(command string) ([]string, error) {
	cmd := strings.Split(command, " ")
	if len(cmd) < 1 {
		return nil, ErrNoCommand
	}

	return cmd, nil
}

/*
 * WORLD LAYER
 *
 * Stores player and world
 * and their current status.
 */

// Errors

var (
	ErrInventoryItemNotFound   = errors.New("no item in the inventory")
	ErrInventoryNotFound       = errors.New("inventory not found")
	ErrInventoryAlreadyHasItem = errors.New("inventory already has item")
	ErrInventoryAvailable      = errors.New("inventory available")
	ErrInventoryNotAvailable   = errors.New("inventory not available")
)

// World Object

type ConfigWorld struct {
	World
}

type World struct {
	Player *Player
	Rooms  map[string]*Room
}

type Item struct {
	Name       string
	Activities map[Activity]bool
}

type Object struct {
	Inventory  map[string]*Item
	Status     string
	AffectedBy map[string]*Item
	CanHold    map[string]*Item
}

type Room struct {
	Inventory map[string]*Object
	Gateways  map[string]*Room
}

type Player struct {
	Inventory          map[string]*Item
	InventoryAvailable bool
	Position           *Room
	Goals              []map[string][]string
}

func (p *Player) CreateInventory() error {
	if p.InventoryAvailable {
		return ErrInventoryAvailable
	}

	p.InventoryAvailable = true

	return nil
}

func (p *Player) RemoveInventory() error {
	if !p.InventoryAvailable {
		return ErrInventoryNotAvailable
	}

	p.InventoryAvailable = false

	return nil
}

func (p *Player) TakeItem(name string) (*Item, error) {
	if !p.InventoryAvailable {
		return nil, ErrInventoryNotFound
	}

	item, ok := p.Inventory[name]
	if !ok {
		return nil, ErrInventoryItemNotFound
	}

	delete(p.Inventory, name)

	return item, nil
}

func (p *Player) PutItem(i *Item) error {
	if !p.InventoryAvailable {
		return ErrInventoryNotFound
	}

	_, ok := p.Inventory[i.Name]
	if ok {
		return ErrInventoryAlreadyHasItem
	}

	p.Inventory[i.Name] = i

	return nil
}

func NewWorld(c *ConfigWorld) *World {
	return &World{
		Player: c.World.Player,
		Rooms:  c.World.Rooms,
	}
}

/*
 * TRANSPORT LAYER
 *
 * Responsible for user
 * input and game output,
 * routing and validation.
 */

// Lookout Route

type LookoutUsecaseInterface interface{}

type LookoutHandler struct {
	Usecase LookoutUsecaseInterface
}

func (l *LookoutHandler) LookoutHandlerFunc(c *Command) (*Result, error) {
	return &Result{}, nil
}

/*
 * SERVICE LAYER
 *
 * Responsible for business logic.
 */

// Common Interfaces

type Cache interface{}

// Lookout Logic

type LookoutUsecase struct {
	Cache Cache
}

func (l *LookoutUsecase) LookoutUsecaseFunc() {

}
