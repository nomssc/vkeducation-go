package main

import (
	"errors"
	"fmt"
	"strings"
)

func main() {
	initGame()

	_ = handleCommand("")
}

func initGame() {
	var (
		ItemKeys = Item{
			Name: "ключи",

			OpenDoors: true,
			PutOn:     false,
			Carry:     true,
		}
		ItemWriting = Item{
			Name: "конспекты",

			OpenDoors: false,
			PutOn:     false,
			Carry:     true,
		}
		ItemBackpack = Item{
			Name: "рюкзак",

			OpenDoors: false,
			PutOn:     true,
			Carry:     false,
		}
		ItemTea = Item{
			Name: "чай",

			OpenDoors: false,
			PutOn:     true,
			Carry:     false,
		}
	)

	var (
		FurnitureTable = "стол"
		FurnitureChair = "стул"
	)

	var (
		DoorStreet = Door{
			Name:   "дверь на улицу",
			Status: false,

			OpenedWith: ItemKeys.Name,
		}
	)

	var (
		RoomNameKitchen         = "кухня"
		RoomEntryWelcomeKitchen = "кухня, ничего интересного"
		RoomNameHallway         = "коридор"
		RoomEntryWelcomeHallway = "ничего интересного"
		RoomNameBedroom         = "комната"
		RoomEntryWelcomeBedroom = "ты в своей комнате"
		RoomNameStreet          = "улица"
		RoomEntryWelcomeStreet  = "на улице весна"
		RoomNameHome            = "домой"
		RoomEntryWelcomeHome    = ""
	)

	var (
		RoomKitchen = Room{
			Name: RoomNameKitchen,
			Inventory: map[string]map[string]struct{}{
				FurnitureTable: map[string]struct{}{
					ItemTea.Name: struct{}{},
				},
			},
			Door:     "",
			Gateways: []string{RoomNameHallway},

			EntryWelcome: RoomEntryWelcomeKitchen,
		}
		RoomHallway = Room{
			Name:      RoomNameHallway,
			Inventory: map[string]map[string]struct{}{},
			Door:      "",
			Gateways:  []string{RoomNameKitchen, RoomNameBedroom, RoomNameStreet},

			EntryWelcome: RoomEntryWelcomeHallway,
		}
		RoomBedroom = Room{
			Name: RoomNameBedroom,
			Inventory: map[string]map[string]struct{}{
				FurnitureTable: map[string]struct{}{
					ItemKeys.Name:    struct{}{},
					ItemWriting.Name: struct{}{},
				},
				FurnitureChair: map[string]struct{}{
					ItemBackpack.Name: struct{}{},
				},
			},
			Door:     "",
			Gateways: []string{RoomNameHallway},

			EntryWelcome: RoomEntryWelcomeBedroom,
		}
		RoomStreet = Room{
			Name:      RoomNameStreet,
			Inventory: map[string]map[string]struct{}{},
			Door:      DoorStreet.Name,
			Gateways:  []string{RoomNameHome},

			EntryWelcome: RoomEntryWelcomeStreet,
		}
		RoomHome = Room{
			Name:      RoomNameHome,
			Inventory: map[string]map[string]struct{}{},
			Door:      "",
			Gateways:  []string{},

			EntryWelcome: RoomEntryWelcomeHome,
		}
	)

	var Player = &Player{
		Name:      "@nomssc",
		Inventory: map[string]struct{}{},
		Clothes:   map[string]struct{}{},
		Position:  RoomNameKitchen,

		InventoryAvailable: false,
	}

	world = NewWorld(ConfigWorld{
		World{
			Player: Player,
			Items: map[string]Item{
				ItemKeys.Name:     ItemKeys,
				ItemWriting.Name:  ItemWriting,
				ItemBackpack.Name: ItemBackpack,
				ItemTea.Name:      ItemTea,
			},
			Doors: map[string]*Door{
				DoorStreet.Name: &DoorStreet,
			},
			Rooms: map[string]*Room{
				RoomKitchen.Name: &RoomKitchen,
				RoomHallway.Name: &RoomHallway,
				RoomBedroom.Name: &RoomBedroom,
				RoomStreet.Name:  &RoomStreet,
				RoomHome.Name:    &RoomHome,
			},
		},
	})

	cli = NewCLI(ConfigCLI{
		CLI{
			Mux:   NewMux(),
			Codec: NewCodec(),
		},
	})

	walk := &WalkTransport{Usecase: &WalkUsecase{World: world}, AgrumentsCount: 1}
	take := &TakeTransport{Usecase: &TakeUsecase{World: world}, AgrumentsCount: 1}
	putOn := &PutOnTransport{Usecase: &PutOnUsecase{World: world}, AgrumentsCount: 1}

	//cli.Handle("осмотреться", nil)
	cli.Handle("идти", walk.Handler)
	//cli.Handle("применить", nil)
	cli.Handle("взять", take.Handler)
	cli.Handle("надеть", putOn.Handler)
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
	ErrTransportItemIsNotInTheInventory    = errors.New("нет такого")
	ErrTransportNoGateway                  = errors.New("нет пути")
	ErrTransportDoorIsClosed               = errors.New("дверь закрыта")
	ErrTransportRoomDoesNotExists          = errors.New("комната не существует")
	ErrTransportDoorDoesNotExists          = errors.New("дверь не существует")
	ErrTransportItemDoesNotExists          = errors.New("нет такого")
	ErrTransportItemCantBeCarried          = errors.New("предмет нельзя положить в инвентарь")
	ErrTransportItemAlreadyInTheInventory  = errors.New("предмент уже есть в инвентаре")
	ErrTransportItemCantBePutOn            = errors.New("предмет нельзя надеть")
	ErrTransportItemAlreadyPutOn           = errors.New("предмет уже надет")
)

type PutOnTransport struct {
	Usecase *PutOnUsecase

	AgrumentsCount int
}

func (t *PutOnTransport) Handler(r Request, w *ResponseWriter) {
	if !validateArgumentsCount(r.Args(), t.AgrumentsCount) {
		w.Error(ErrTransportIncorrectNumberOfArguments)
		return
	}

	err := t.Usecase.PutOn(r.Args()[0])
	if err != nil {
		switch {
		case errors.Is(err, ErrServiceItemDoesNotExists):
			w.Error(ErrTransportItemDoesNotExists)
			return
		case errors.Is(err, ErrServiceItemCantBePutOn):
			w.Error(ErrTransportItemCantBePutOn)
			return
		case errors.Is(err, ErrServiceRoomDoesNotExists):
			w.Error(fmt.Errorf("%w", ErrTransportRoomDoesNotExists))
			return
		case errors.Is(err, ErrServiceItemAlreadyPutOn):
			w.Error(ErrTransportItemAlreadyPutOn)
			return
		case errors.Is(err, ErrServiceItemIsNotInTheInventory):
			w.Error(ErrTransportItemIsNotInTheInventory)
			return
		default:
			w.Error(ErrTransportUnexpectedStuff)
			return
		}
	}

	w.Write(fmt.Sprintf("вы надели: %s", r.Args()[0]))
}

type TakeTransport struct {
	Usecase *TakeUsecase

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
		case errors.Is(err, ErrServiceRoomDoesNotExists):
			w.Error(fmt.Errorf("%w", ErrTransportRoomDoesNotExists))
			return
		case errors.Is(err, ErrServiceInventoryNotAvailable):
			w.Error(ErrTransportInventoryNotAvailable)
			return
		case errors.Is(err, ErrServiceItemIsNotInTheInventory):
			w.Error(ErrTransportItemIsNotInTheInventory)
			return
		case errors.Is(err, ErrServiceItemDoesNotExists):
			w.Error(ErrTransportItemDoesNotExists)
			return
		case errors.Is(err, ErrServiceItemCantBeCarried):
			w.Error(ErrTransportItemCantBeCarried)
			return
		case errors.Is(err, ErrServiceItemAlreadyInTheInventory):
			w.Error(ErrTransportItemAlreadyInTheInventory)
			return
		default:
			w.Error(ErrTransportUnexpectedStuff)
			return
		}
	}

	w.Write(fmt.Sprintf("предмет добавлен в инвентарь: %s", r.Args()[0]))
}

type WalkTransport struct {
	Usecase *WalkUsecase

	AgrumentsCount int
}

func (t *WalkTransport) Handler(r Request, w *ResponseWriter) {
	if !validateArgumentsCount(r.Args(), t.AgrumentsCount) {
		w.Error(ErrTransportIncorrectNumberOfArguments)
		return
	}

	entryWelcome, gateways, err := t.Usecase.Walk(r.Args()[0])
	if err != nil {
		switch {
		case errors.Is(err, ErrServiceRoomDoesNotExists):
			w.Error(fmt.Errorf("%w", ErrTransportRoomDoesNotExists))
			return
		case errors.Is(err, ErrServiceNoGateway):
			w.Error(fmt.Errorf("%w в %s", ErrTransportNoGateway, r.Args()[0]))
			return
		case errors.Is(err, ErrServiceNoGateway):
			w.Error(fmt.Errorf("%w", ErrTransportDoorIsClosed))
			return
		case errors.Is(err, ErrServiceDoorDoesNotExists):
			w.Error(fmt.Errorf("%w", ErrTransportDoorDoesNotExists))
			return
		case errors.Is(err, ErrServiceDoorIsClosed):
			w.Error(fmt.Errorf("%w", ErrTransportDoorIsClosed))
			return
		default:
			w.Error(ErrTransportUnexpectedStuff)
			return
		}
	}

	w.Write(fmt.Sprintf("%s. можно пройти - %s", entryWelcome, gateways))
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

var (
	ErrServiceNoGateway                 = errors.New("no gateway")
	ErrServiceDoorDoesNotExists         = errors.New("door does not exists")
	ErrServiceRoomDoesNotExists         = errors.New("room does not exists")
	ErrServiceItemDoesNotExists         = errors.New("item does not exists")
	ErrServiceInventoryNotAvailable     = errors.New("inventory not available")
	ErrServiceDoorIsClosed              = errors.New("door is closed")
	ErrServiceItemAlreadyInTheInventory = errors.New("item already in the inventory")
	ErrServiceItemIsNotInTheInventory   = errors.New("item is not the inventory")
	ErrServiceItemAlreadyPutOn          = errors.New("item already put on")
	ErrServiceItemCantBeCarried         = errors.New("item can't be carried")
	ErrServiceItemCantBePutOn           = errors.New("item can't be put on")
)

type PutOnUsecase struct {
	World *World
}

func (u *PutOnUsecase) PutOn(itemName string) error {
	items := u.World.GetItems()
	item, ok := items[itemName]
	if !ok {
		return ErrServiceItemDoesNotExists
	}
	if !item.GetPutOn() {
		return ErrServiceItemCantBePutOn
	}

	roomName := u.World.Player.GetPosition()
	rooms := u.World.GetRooms()
	room, ok := rooms[roomName]
	if !ok {
		return ErrServiceRoomDoesNotExists
	}

	clothes := u.World.Player.GetClothes()
	_, ok = clothes[itemName]
	if ok {
		return ErrServiceItemAlreadyPutOn
	}

	err := room.RemoveItem(itemName)
	if err != nil {
		return ErrServiceItemIsNotInTheInventory
	}
	u.World.Player.StoreWear(itemName)
	u.World.Player.ChangeInventoryAvailability(true)

	return nil
}

type TakeUsecase struct {
	World *World
}

func (u *TakeUsecase) Take(itemName string) error {
	items := u.World.GetItems()
	item, ok := items[itemName]
	if !ok {
		return ErrServiceItemDoesNotExists
	}
	if !item.GetCarry() {
		return ErrServiceItemCantBeCarried
	}

	roomName := u.World.Player.GetPosition()
	rooms := u.World.GetRooms()
	room, ok := rooms[roomName]
	if !ok {
		return ErrServiceRoomDoesNotExists
	}

	if !u.World.Player.GetInventoryAvailable() {
		return ErrServiceInventoryNotAvailable
	}

	err := room.RemoveItem(itemName)
	if err != nil {
		return ErrServiceItemIsNotInTheInventory
	}

	inventory := u.World.Player.GetInventory()
	_, ok = inventory[itemName]
	if ok {
		return ErrServiceItemAlreadyInTheInventory
	}

	u.World.Player.StoreItem(itemName)

	return nil
}

type WalkUsecase struct {
	World *World
}

func (u *WalkUsecase) Walk(direction string) (string, string, error) {
	roomName := u.World.Player.GetPosition()

	rooms := u.World.GetRooms()
	currentRoom, ok := rooms[roomName]
	if !ok {
		return "", "", ErrServiceRoomDoesNotExists
	}
	gateways := currentRoom.GetGateways()
	exists := false
	for _, gateway := range gateways {
		if gateway == direction {
			exists = true
		}
	}
	if !exists {
		return "", "", ErrServiceNoGateway
	}

	nextRoom, ok := rooms[direction]
	if !ok {
		return "", "", ErrServiceRoomDoesNotExists
	}
	doorName := nextRoom.Door
	doors := u.World.GetDoors()
	door, ok := doors[doorName]
	if ok && !door.GetDoorStatus() {
		return "", "", ErrServiceDoorIsClosed
	}

	u.World.Player.ChangePosition(nextRoom.Name)

	gateways = nextRoom.GetGateways()
	gatewaysStr := strings.Join(gateways, ", ")

	return nextRoom.EntryWelcome, gatewaysStr, nil
}

/*
 * WORLD LAYER
 *
 * Acts as database for world
 * resources.
 */

var world *World

type ConfigWorld struct {
	World
}

type World struct {
	Player *Player

	Items map[string]Item
	Doors map[string]*Door
	Rooms map[string]*Room
}

type Player struct {
	Name      string
	Inventory map[string]struct{}
	Clothes   map[string]struct{}
	Position  string

	InventoryAvailable bool
}

func (p *Player) GetName() string {
	return p.Name
}

func (p *Player) GetInventory() map[string]struct{} {
	return p.Inventory
}

func (p *Player) StoreItem(itemName string) {
	p.Inventory[itemName] = struct{}{}
}

func (p *Player) GetClothes() map[string]struct{} {
	return p.Clothes
}

func (p *Player) StoreWear(itemName string) {
	p.Clothes[itemName] = struct{}{}
}

func (p *Player) GetPosition() string {
	return p.Position
}

func (p *Player) ChangePosition(roomName string) {
	p.Position = roomName
}

func (p *Player) GetInventoryAvailable() bool {
	return p.InventoryAvailable
}

func (p *Player) ChangeInventoryAvailability(value bool) {
	p.InventoryAvailable = value
}

type Item struct {
	Name string

	OpenDoors bool
	PutOn     bool
	Carry     bool
}

func (i Item) GetName() string {
	return i.Name
}

func (i Item) GetOpenDoors() bool {
	return i.OpenDoors
}

func (i Item) GetPutOn() bool {
	return i.PutOn
}

func (i Item) GetCarry() bool {
	return i.Carry
}

type Door struct {
	Name   string
	Status bool

	OpenedWith string
}

func (d *Door) GetDoorStatus() bool {
	return d.Status
}

func (d *Door) GetOpenedWith() string {
	return d.OpenedWith
}

type Room struct {
	Name      string
	Inventory map[string]map[string]struct{}
	Door      string
	Gateways  []string

	EntryWelcome string
}

func (r *Room) GetName() string {
	return r.Name
}

func (r *Room) RemoveItem(itemName string) error {
	for _, furniture := range r.Inventory {
		_, ok := furniture[itemName]
		if ok {
			delete(furniture, itemName)
			return nil
		}
	}

	return fmt.Errorf("item is not the inventory")
}

func (r *Room) GetDoor() string {
	return r.Door
}

func (r *Room) GetGateways() []string {
	return r.Gateways
}

func (r *Room) GetEntryWelcome() string {
	return r.EntryWelcome
}

func NewWorld(c ConfigWorld) *World {
	return &World{
		Player: c.Player,

		Items: c.Items,
		Doors: c.Doors,
		Rooms: c.Rooms,
	}
}

func (w *World) GetItems() map[string]Item {
	return w.Items
}

func (w *World) GetDoors() map[string]*Door {
	return w.Doors
}

func (w *World) GetRooms() map[string]*Room {
	return w.Rooms
}

/*
 * CLI PACKAGE
 *
 * Backend backbone.
 */

/* CLI THE TRANSPORT MASTER */

var (
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
		return ErrCLINoCommand.Error()
	}

	h, err := cli.Mux.Process(c[0])
	if err != nil {
		return ErrCLICommandDoesNotExists.Error()
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
		return nil, fmt.Errorf("command does not exists")
	}

	return handler, nil
}

/* CODEC THE INPUT PARSER */

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
		return nil, fmt.Errorf("no command")
	}

	return command, nil
}
