package main

import (
	"errors"
	"fmt"
	"strings"
)

func main() {
	initGame()

	fmt.Println()
	fmt.Println(world.Rooms[world.Player.Position].Inventory)
	fmt.Println(world.Player.Inventory)

	res := handleCommand("взять ключи")

	fmt.Println()
	fmt.Println(world.Rooms[world.Player.Position].Inventory)
	fmt.Println(world.Player.Inventory)

	fmt.Println(res)

	res = handleCommand("взять конспекты")

	fmt.Println()
	fmt.Println(world.Rooms[world.Player.Position].Inventory)
	fmt.Println(world.Player.Inventory)
	fmt.Println()

	fmt.Println(res)
}

func initGame() {
	var (
		ItemKeys     Item = "ключи"
		ItemWriting  Item = "конспекты"
		ItemBackpack Item = "рюкзак"
		ItemTea      Item = "чай"
	)

	var (
		FurnitureTable Furniture = "стол"
		FurnitureChair Furniture = "стул"
	)

	var (
		RoomNameKitchen = "кухня"
		RoomNameHallway = "коридор"
		RoomNameBedroom = "комната"
		RoomNameStreet  = "улица"
		RoomNameHome    = "дом"
	)

	var (
		RoomKitchen Room = Room{
			Name: RoomNameKitchen,
			Inventory: map[Furniture]map[Item]struct{}{
				FurnitureTable: map[Item]struct{}{
					ItemTea: struct{}{},
				},
			},
			Gateways: map[string]struct{}{
				RoomNameHallway: struct{}{},
			},
		}
		RoomHallway Room = Room{
			Name:      RoomNameHallway,
			Inventory: map[Furniture]map[Item]struct{}{},
			Gateways: map[string]struct{}{
				RoomNameBedroom: struct{}{},
				RoomNameKitchen: struct{}{},
				RoomNameStreet:  struct{}{},
			},
		}
		RoomBedroom Room = Room{
			Name: RoomNameBedroom,
			Inventory: map[Furniture]map[Item]struct{}{
				FurnitureTable: map[Item]struct{}{
					ItemKeys:    struct{}{},
					ItemWriting: struct{}{},
				},
				FurnitureChair: map[Item]struct{}{
					ItemBackpack: struct{}{},
				},
			},
			Gateways: map[string]struct{}{
				RoomNameHallway: struct{}{},
			},
		}
		RoomStreet Room = Room{
			Name:      RoomNameStreet,
			Inventory: map[Furniture]map[Item]struct{}{},
			Gateways: map[string]struct{}{
				RoomNameHome: struct{}{},
			},
		}
		RoomHome Room = Room{
			Name:      RoomNameHome,
			Inventory: map[Furniture]map[Item]struct{}{},
			Gateways:  map[string]struct{}{},
		}
	)

	world = NewWorld(ConfigWorld{
		World{
			Player: &Player{
				Inventory:          map[Item]struct{}{},
				InventoryAvailable: true,
				Position:           RoomNameBedroom,
			},

			Items: map[Item]struct{}{
				ItemKeys:     struct{}{},
				ItemWriting:  struct{}{},
				ItemBackpack: struct{}{},
				ItemTea:      struct{}{},
			},
			Furniture: map[Furniture]struct{}{
				FurnitureTable: struct{}{},
				FurnitureChair: struct{}{},
			},
			Rooms: map[string]*Room{
				RoomNameKitchen: &RoomKitchen,
				RoomNameHallway: &RoomHallway,
				RoomNameBedroom: &RoomBedroom,
				RoomNameStreet:  &RoomStreet,
				RoomNameHome:    &RoomHome,
			},
		},
	})

	cli = NewCLI(ConfigCLI{
		CLI{
			Mux:   NewMux(),
			Codec: NewCodec(),
		},
	})

	take := &TakeTransport{Usecase: &TakeUsecase{World: world}, AgrumentsCount: 1}

	cli.Handle("осмотреться", nil)
	cli.Handle("идти", nil)
	cli.Handle("применить", nil)
	cli.Handle("взять", take.Handler)
	cli.Handle("надеть", nil)
}

/*
 * TRANSPORT LAYER
 *
 * Responsible for user
 * input and game output,
 * routing and validation.
 */

var (
	ErrTransportUnexpectedStuff            = errors.New("неожиданная ошибка")
	ErrTransportIncorrectNumberOfArguments = errors.New("некорректное количество аргументов")
	ErrTransportInventoryNotAvailable      = errors.New("некуда класть")
	ErrTransportItemAlreadyInTheInventory  = errors.New("предмет уже в инвентаре")
	ErrTransportItemDoesNotExists          = errors.New("нет такого")
	ErrTransportItemIsNotInTheInventory    = errors.New("нет такого")
)

var (
	MsgTransportItemPlacedInTheInventory = "предмет добавлен в инвентарь"
)

type TakeUsecaser interface {
	Take(name string) error
}

type TakeTransport struct {
	Usecase TakeUsecaser

	AgrumentsCount int
}

func (t *TakeTransport) Handler(r Request, w *ResponseWriter) {
	if !validateArgumentsCount(r.Args(), t.AgrumentsCount) {
		w.Error(ErrTransportIncorrectNumberOfArguments)
		return
	}

	err := t.Usecase.Take(r.Args()[0])
	if err != nil {
		switch {
		case errors.Is(err, ErrWorldInventoryNotAvailable):
			w.Error(ErrTransportInventoryNotAvailable)
			return
		case errors.Is(err, ErrWorldItemAlreadyInTheInventory):
			w.Error(ErrTransportItemAlreadyInTheInventory)
			return
		case errors.Is(err, ErrWorldItemDoesNotExists):
			w.Error(ErrTransportItemDoesNotExists)
			return
		case errors.Is(err, ErrWorldItemIsNotInTheInventory):
			w.Error(ErrTransportItemIsNotInTheInventory)
			return
		default:
			w.Error(ErrTransportUnexpectedStuff)
			return
		}
	}

	w.Write(fmt.Sprintf("%s: %s", MsgTransportItemPlacedInTheInventory, r.Args()[0]))
}

func validateArgumentsCount(args []string, expectedCount int) bool {
	if len(args) != expectedCount {
		return false
	}

	return true
}

/*
 * SERVICE LAYER
 *
 * Processes business logic.
 */

type TakeUsecase struct {
	World *World
}

func (u *TakeUsecase) Take(name string) error {
	err := u.World.CheckItem(name)
	if err != nil {
		return err
	}

	position := u.World.Player.CurrentPosition()

	err = u.World.Rooms[position].TakeFromTheInventory(name)
	if err != nil {
		return err
	}

	err = u.World.Player.InventoryStatus()
	if err != nil {
		return err
	}

	err = u.World.Player.PlaceInTheInventory(name)
	if err != nil {
		return err
	}

	return nil
}

/*
 * WORLD LAYER
 *
 * Acts as database for world
 * resources.
 */

var (
	ErrWorldInventoryNotAvailable     = errors.New("no inventory")
	ErrWorldItemAlreadyInTheInventory = errors.New("item already in the inventory")
	ErrWorldItemDoesNotExists         = errors.New("item does not exists")
	ErrWorldItemIsNotInTheInventory   = errors.New("item is not the inventory")
)

var world *World

type ConfigWorld struct {
	World
}

type Item string

type Furniture string

type Room struct {
	Name      string
	Inventory map[Furniture]map[Item]struct{}
	Gateways  map[string]struct{}
}

func (r *Room) TakeFromTheInventory(name string) error {
	for _, furniture := range r.Inventory {
		_, ok := furniture[Item(name)]
		if ok {
			delete(furniture, Item(name))
			return nil
		}
	}

	return ErrWorldItemIsNotInTheInventory
}

type Player struct {
	Inventory          map[Item]struct{}
	InventoryAvailable bool
	Position           string
}

func (p *Player) InventoryStatus() error {
	if !p.InventoryAvailable {
		return ErrWorldInventoryNotAvailable
	}

	return nil
}

func (p *Player) CurrentPosition() string {
	return p.Position
}

func (p *Player) PlaceInTheInventory(name string) error {
	_, ok := p.Inventory[Item(name)]
	if ok {
		return ErrWorldItemAlreadyInTheInventory
	}

	p.Inventory[Item(name)] = struct{}{}

	return nil
}

type World struct {
	Player *Player

	Items     map[Item]struct{}
	Furniture map[Furniture]struct{}
	Rooms     map[string]*Room
}

func NewWorld(c ConfigWorld) *World {
	return &World{
		Player: c.World.Player,

		Items:     c.World.Items,
		Furniture: c.World.Furniture,
		Rooms:     c.World.Rooms,
	}
}

func (w *World) CheckItem(name string) error {
	_, ok := w.Items[Item(name)]
	if !ok {
		return ErrWorldItemDoesNotExists
	}

	return nil
}

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
	error   error
}

func (r *ResponseWriter) Write(s string) {
	r.message = s
}

func (r *ResponseWriter) Error(e error) {
	r.error = e
}

// Handler type.
type Handler func(r Request, w *ResponseWriter)

// Used to configurate CLI on
// inititalization.
type ConfigCLI struct {
	CLI
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

func NewCLI(c ConfigCLI) *CLI {
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

	h(r, w)

	if w.error != nil {
		return w.error.Error()
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

func NewMux(c ...ConfigMux) *Mux {
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
