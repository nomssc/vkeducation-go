package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

func main() {}

func initGame() {
	var (
		itemKeys = Item{
			Name: "ключи",

			OpenDoors: true,
			Carry:     true,
		}
		itemWriting = Item{
			Name: "конспекты",

			Carry: true,
		}
		itemBackpack = Item{
			Name: "рюкзак",

			PutOn:          true,
			GivesInventory: true,
		}
		itemTea = Item{
			Name: "чай",
		}
	)

	doorStreet := &Door{
		Name:   "дверь",
		Status: false,
	}

	const (
		roomNameKitchen = "кухня"
		roomNameHallway = "коридор"
		roomNameBedroom = "комната"
		roomNameStreet  = "улица"
	)

	// Gateway{Label, Room}: the label is what the player types and sees,
	// the room is where it actually leads. They differ for "домой".
	kitchen := &Room{
		Name: roomNameKitchen,
		Furniture: []Furniture{
			{Place: "на столе", Items: []string{itemTea.Name}},
		},
		Gateways: []Gateway{{Label: roomNameHallway, Room: roomNameHallway}},

		EntryWelcome:   "кухня, ничего интересного",
		LookoutWelcome: "ты находишься на кухне",
	}
	hallway := &Room{
		Name: roomNameHallway,
		Gateways: []Gateway{
			{Label: roomNameKitchen, Room: roomNameKitchen},
			{Label: roomNameBedroom, Room: roomNameBedroom},
			{Label: roomNameStreet, Room: roomNameStreet},
		},

		EntryWelcome: "ничего интересного",
	}
	bedroom := &Room{
		Name: roomNameBedroom,
		Furniture: []Furniture{
			{Place: "на столе", Items: []string{itemKeys.Name, itemWriting.Name}},
			{Place: "на стуле", Items: []string{itemBackpack.Name}},
		},
		Gateways: []Gateway{{Label: roomNameHallway, Room: roomNameHallway}},

		EntryWelcome: "ты в своей комнате",
	}
	street := &Room{
		Name:     roomNameStreet,
		Door:     doorStreet.Name,
		Gateways: []Gateway{{Label: "домой", Room: roomNameKitchen}},

		EntryWelcome: "на улице весна",
	}

	player := &Player{
		Name:      "@nomssc",
		Inventory: map[string]struct{}{},
		Clothes:   map[string]struct{}{},
		Position:  roomNameKitchen,
		Visited:   map[string]struct{}{roomNameKitchen: {}},
		Goals: []Goal{
			{Name: "собрать рюкзак", Items: []string{itemKeys.Name, itemWriting.Name}},
			{Name: "идти в универ", Rooms: []string{roomNameStreet}},
		},
	}

	// Kitchen-specific behavior lives here, not in the usecase.
	kitchen.LookoutExtra = func(p *Player) string {
		pending := p.PendingGoals()
		if len(pending) == 0 {
			return ""
		}

		return "надо " + strings.Join(pending, " и ")
	}

	world = NewWorld(ConfigWorld{
		World{
			Player: player,
			Items: map[string]Item{
				itemKeys.Name:     itemKeys,
				itemWriting.Name:  itemWriting,
				itemBackpack.Name: itemBackpack,
				itemTea.Name:      itemTea,
			},
			Doors: map[string]*Door{
				doorStreet.Name: doorStreet,
			},
			Rooms: map[string]*Room{
				kitchen.Name: kitchen,
				hallway.Name: hallway,
				bedroom.Name: bedroom,
				street.Name:  street,
			},
		},
	})

	cli = NewCLI(ConfigCLI{
		CLI{
			Mux:   NewMux(),
			Codec: NewCodec(),
		},
	})

	lookout := &LookoutTransport{Usecase: &LookoutUsecase{World: world}, ArgumentsCount: 0}
	walk := &WalkTransport{Usecase: &WalkUsecase{World: world}, ArgumentsCount: 1}
	open := &OpenTransport{Usecase: &OpenUsecase{World: world}, ArgumentsCount: 2}
	take := &TakeTransport{Usecase: &TakeUsecase{World: world}, ArgumentsCount: 1}
	putOn := &PutOnTransport{Usecase: &PutOnUsecase{World: world}, ArgumentsCount: 1}

	cli.Handle("осмотреться", lookout.Handler)
	cli.Handle("идти", walk.Handler)
	cli.Handle("применить", open.Handler)
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
	ErrTransportItemIsNotInTheInventory    = errors.New("нет предмета в инвентаре")
	ErrTransportItemIsNotInTheRoom         = errors.New("нет такого")
	ErrTransportNoGateway                  = errors.New("нет пути")
	ErrTransportDoorIsClosed               = errors.New("дверь закрыта")
	ErrTransportRoomDoesNotExists          = errors.New("комната не существует")
	ErrTransportDoorDoesNotExists          = errors.New("дверь не существует")
	ErrTransportItemDoesNotExists          = errors.New("нет такого")
	ErrTransportItemCantBeCarried          = errors.New("предмет нельзя положить в инвентарь")
	ErrTransportItemAlreadyInTheInventory  = errors.New("предмент уже есть в инвентаре")
	ErrTransportItemCantBePutOn            = errors.New("предмет нельзя надеть")
	ErrTransportItemAlreadyPutOn           = errors.New("предмет уже надет")
	ErrTransportItemCantBeUsed             = errors.New("не к чему применить")
)

/* LOOKOUT COMMAND TRANSPORT LOGIC */

type LookoutTransport struct {
	Usecase *LookoutUsecase

	ArgumentsCount int
}

func (t *LookoutTransport) Handler(r Request, w *ResponseWriter) {
	if len(r.Args()) != t.ArgumentsCount {
		w.Error(ErrTransportIncorrectNumberOfArguments)
		return
	}

	message, err := t.Usecase.Lookout()
	if err != nil {
		if errors.Is(err, ErrServiceRoomDoesNotExists) {
			w.Error(ErrTransportRoomDoesNotExists)
			return
		}
		w.Error(ErrTransportUnexpectedStuff)
		return
	}

	w.Write(message)
}

/* OPEN COMMAND TRANSPORT LOGIC */

type OpenTransport struct {
	Usecase *OpenUsecase

	ArgumentsCount int
}

func (t *OpenTransport) Handler(r Request, w *ResponseWriter) {
	if len(r.Args()) != t.ArgumentsCount {
		w.Error(ErrTransportIncorrectNumberOfArguments)
		return
	}

	err := t.Usecase.Open(r.Args()[0], r.Args()[1])
	if err != nil {
		switch {
		case errors.Is(err, ErrServiceItemDoesNotExists):
			w.Error(ErrTransportItemDoesNotExists)
		case errors.Is(err, ErrServiceItemIsNotInTheInventory):
			w.Error(fmt.Errorf("%w - %s", ErrTransportItemIsNotInTheInventory, r.Args()[0]))
		case errors.Is(err, ErrServiceItemCantOpenDoors) || errors.Is(err, ErrServiceObjectIsNotDoor):
			w.Error(ErrTransportItemCantBeUsed)
		case errors.Is(err, ErrServiceRoomDoesNotExists):
			w.Error(ErrTransportRoomDoesNotExists)
		default:
			w.Error(ErrTransportUnexpectedStuff)
		}

		return
	}

	w.Write("дверь открыта")
}

/* PUT ON COMMAND TRANSPORT LOGIC */

type PutOnTransport struct {
	Usecase *PutOnUsecase

	ArgumentsCount int
}

func (t *PutOnTransport) Handler(r Request, w *ResponseWriter) {
	if len(r.Args()) != t.ArgumentsCount {
		w.Error(ErrTransportIncorrectNumberOfArguments)
		return
	}

	err := t.Usecase.PutOn(r.Args()[0])
	if err != nil {
		switch {
		case errors.Is(err, ErrServiceItemDoesNotExists):
			w.Error(ErrTransportItemDoesNotExists)
		case errors.Is(err, ErrServiceItemCantBePutOn):
			w.Error(ErrTransportItemCantBePutOn)
		case errors.Is(err, ErrServiceRoomDoesNotExists):
			w.Error(ErrTransportRoomDoesNotExists)
		case errors.Is(err, ErrServiceItemAlreadyPutOn):
			w.Error(ErrTransportItemAlreadyPutOn)
		case errors.Is(err, ErrServiceItemIsNotInTheRoom):
			w.Error(ErrTransportItemIsNotInTheRoom)
		default:
			w.Error(ErrTransportUnexpectedStuff)
		}

		return
	}

	w.Write(fmt.Sprintf("вы надели: %s", r.Args()[0]))
}

/* TAKE COMMAND TRANSPORT LOGIC */

type TakeTransport struct {
	Usecase *TakeUsecase

	ArgumentsCount int
}

func (t *TakeTransport) Handler(r Request, w *ResponseWriter) {
	if len(r.Args()) != t.ArgumentsCount {
		w.Error(ErrTransportIncorrectNumberOfArguments)
		return
	}

	err := t.Usecase.Take(r.Args()[0])
	if err != nil {
		switch {
		case errors.Is(err, ErrServiceRoomDoesNotExists):
			w.Error(ErrTransportRoomDoesNotExists)
		case errors.Is(err, ErrServiceInventoryNotAvailable):
			w.Error(ErrTransportInventoryNotAvailable)
		case errors.Is(err, ErrServiceItemIsNotInTheRoom):
			w.Error(ErrTransportItemIsNotInTheRoom)
		case errors.Is(err, ErrServiceItemDoesNotExists):
			w.Error(ErrTransportItemDoesNotExists)
		case errors.Is(err, ErrServiceItemCantBeCarried):
			w.Error(ErrTransportItemCantBeCarried)
		case errors.Is(err, ErrServiceItemAlreadyInTheInventory):
			w.Error(ErrTransportItemAlreadyInTheInventory)
		default:
			w.Error(ErrTransportUnexpectedStuff)
		}

		return
	}

	w.Write(fmt.Sprintf("предмет добавлен в инвентарь: %s", r.Args()[0]))
}

/* WALK COMMAND TRANSPORT LOGIC */

type WalkTransport struct {
	Usecase *WalkUsecase

	ArgumentsCount int
}

func (t *WalkTransport) Handler(r Request, w *ResponseWriter) {
	if len(r.Args()) != t.ArgumentsCount {
		w.Error(ErrTransportIncorrectNumberOfArguments)
		return
	}

	entryWelcome, gateways, err := t.Usecase.Walk(r.Args()[0])
	if err != nil {
		switch {
		case errors.Is(err, ErrServiceRoomDoesNotExists):
			w.Error(ErrTransportRoomDoesNotExists)
		case errors.Is(err, ErrServiceNoGateway):
			w.Error(fmt.Errorf("%w в %s", ErrTransportNoGateway, r.Args()[0]))
		case errors.Is(err, ErrServiceDoorDoesNotExists):
			w.Error(ErrTransportDoorDoesNotExists)
		case errors.Is(err, ErrServiceDoorIsClosed):
			w.Error(ErrTransportDoorIsClosed)
		default:
			w.Error(ErrTransportUnexpectedStuff)
		}

		return
	}

	if gateways == "" {
		w.Write(entryWelcome)
		return
	}

	w.Write(fmt.Sprintf("%s. можно пройти - %s", entryWelcome, gateways))
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
	ErrServiceItemIsNotInTheRoom        = errors.New("item is not the room")
	ErrServiceItemAlreadyPutOn          = errors.New("item already put on")
	ErrServiceItemCantBeCarried         = errors.New("item can't be carried")
	ErrServiceItemCantBePutOn           = errors.New("item can't be put on")
	ErrServiceItemCantOpenDoors         = errors.New("item can't open doors")
	ErrServiceObjectIsNotDoor           = errors.New("object is not a door")
)

/* LOOKOUT COMMAND INTERNAL LOGIC */

type LookoutUsecase struct {
	World *World
}

func (u *LookoutUsecase) Lookout() (string, error) {
	room, ok := u.World.GetRooms()[u.World.Player.GetPosition()]
	if !ok {
		return "", ErrServiceRoomDoesNotExists
	}

	segments := make([]string, 0, 3)

	if welcome := room.GetLookoutWelcome(); welcome != "" {
		segments = append(segments, welcome)
	}

	furniture := make([]string, 0, len(room.GetFurniture()))
	for _, f := range room.GetFurniture() {
		if len(f.Items) != 0 {
			furniture = append(furniture, fmt.Sprintf("%s: %s", f.Place, strings.Join(f.Items, ", ")))
		}
	}
	if len(furniture) == 0 {
		segments = append(segments, "пустая "+room.GetName())
	} else {
		segments = append(segments, strings.Join(furniture, ", "))
	}

	if extra := room.GetLookoutExtra(); extra != nil {
		if s := extra(u.World.Player); s != "" {
			segments = append(segments, s)
		}
	}

	result := strings.Join(segments, ", ")
	if labels := room.GatewayLabels(); len(labels) != 0 {
		result += ". можно пройти - " + strings.Join(labels, ", ")
	}

	return result, nil
}

/* OPEN COMMAND INTERNAL LOGIC */

type OpenUsecase struct {
	World *World
}

func (u *OpenUsecase) Open(itemName string, object string) error {
	if _, ok := u.World.Player.GetInventory()[itemName]; !ok {
		return ErrServiceItemIsNotInTheInventory
	}

	item, ok := u.World.GetItems()[itemName]
	if !ok {
		return ErrServiceItemDoesNotExists
	}
	if !item.GetOpenDoors() {
		return ErrServiceItemCantOpenDoors
	}

	door, ok := u.World.GetDoors()[object]
	if !ok {
		return ErrServiceObjectIsNotDoor
	}

	rooms := u.World.GetRooms()
	room, ok := rooms[u.World.Player.GetPosition()]
	if !ok {
		return ErrServiceRoomDoesNotExists
	}

	// The door must belong to a room reachable from where the player stands.
	reachable := false
	for _, gateway := range room.GetGateways() {
		next, ok := rooms[gateway.Room]
		if ok && next.GetDoor() == object {
			reachable = true
			break
		}
	}
	if !reachable {
		return ErrServiceObjectIsNotDoor
	}

	door.ChangeStatus(true)

	return nil
}

/* PUT ON COMMAND INTERNAL LOGIC */

type PutOnUsecase struct {
	World *World
}

func (u *PutOnUsecase) PutOn(itemName string) error {
	item, ok := u.World.GetItems()[itemName]
	if !ok {
		return ErrServiceItemDoesNotExists
	}
	if !item.GetPutOn() {
		return ErrServiceItemCantBePutOn
	}

	room, ok := u.World.GetRooms()[u.World.Player.GetPosition()]
	if !ok {
		return ErrServiceRoomDoesNotExists
	}

	if _, ok := u.World.Player.GetClothes()[itemName]; ok {
		return ErrServiceItemAlreadyPutOn
	}

	if err := room.RemoveItem(itemName); err != nil {
		return ErrServiceItemIsNotInTheRoom
	}

	u.World.Player.StoreWear(itemName)
	if item.GetGivesInventory() {
		u.World.Player.ChangeInventoryAvailability(true)
	}

	return nil
}

/* TAKE COMMAND INTERNAL LOGIC */

type TakeUsecase struct {
	World *World
}

func (u *TakeUsecase) Take(itemName string) error {
	item, ok := u.World.GetItems()[itemName]
	if !ok {
		return ErrServiceItemDoesNotExists
	}
	if !item.GetCarry() {
		return ErrServiceItemCantBeCarried
	}

	room, ok := u.World.GetRooms()[u.World.Player.GetPosition()]
	if !ok {
		return ErrServiceRoomDoesNotExists
	}

	if !u.World.Player.GetInventoryAvailable() {
		return ErrServiceInventoryNotAvailable
	}

	// An item the player already took is no longer in the room,
	// so it is reported as "нет такого" here.
	if err := room.RemoveItem(itemName); err != nil {
		return ErrServiceItemIsNotInTheRoom
	}

	u.World.Player.StoreItem(itemName)

	return nil
}

/* WALK COMMAND INTERNAL LOGIC */

type WalkUsecase struct {
	World *World
}

func (u *WalkUsecase) Walk(direction string) (string, string, error) {
	rooms := u.World.GetRooms()

	currentRoom, ok := rooms[u.World.Player.GetPosition()]
	if !ok {
		return "", "", ErrServiceRoomDoesNotExists
	}

	target := ""
	for _, gateway := range currentRoom.GetGateways() {
		if gateway.Label == direction {
			target = gateway.Room
			break
		}
	}
	if target == "" {
		return "", "", ErrServiceNoGateway
	}

	nextRoom, ok := rooms[target]
	if !ok {
		return "", "", ErrServiceRoomDoesNotExists
	}

	if doorName := nextRoom.GetDoor(); doorName != "" {
		door, ok := u.World.GetDoors()[doorName]
		if !ok {
			return "", "", ErrServiceDoorDoesNotExists
		}
		if !door.GetDoorStatus() {
			return "", "", ErrServiceDoorIsClosed
		}
	}

	u.World.Player.ChangePosition(nextRoom.GetName())

	return nextRoom.GetEntryWelcome(), strings.Join(nextRoom.GatewayLabels(), ", "), nil
}

/*
 * WORLD LAYER
 *
 * Acts as database for world
 * resources.
 */

/* WORLD THE STATE STORAGE */

var world *World

// Used to configurate world on
// inititalization.
type ConfigWorld struct {
	World
}

// Responsible for storing the
// whole game world state.
type World struct {
	Player *Player

	Items map[string]Item
	Doors map[string]*Door
	Rooms map[string]*Room
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

/* PLAYER THE STATE STORAGE */

// A goal is complete when all of its items are in the
// inventory and all of its rooms have been visited.
type Goal struct {
	Name  string
	Items []string
	Rooms []string
}

// Responsible for storing player
// state.
type Player struct {
	Name      string
	Inventory map[string]struct{}
	Clothes   map[string]struct{}
	Position  string
	Visited   map[string]struct{}
	Goals     []Goal

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
	p.Visited[roomName] = struct{}{}
}

func (p *Player) GetInventoryAvailable() bool {
	return p.InventoryAvailable
}

func (p *Player) ChangeInventoryAvailability(value bool) {
	p.InventoryAvailable = value
}

// Names of the goals that are not complete yet, in declaration order.
func (p *Player) PendingGoals() []string {
	pending := make([]string, 0, len(p.Goals))

	for _, goal := range p.Goals {
		done := true

		for _, item := range goal.Items {
			if _, ok := p.Inventory[item]; !ok {
				done = false
			}
		}
		for _, room := range goal.Rooms {
			if _, ok := p.Visited[room]; !ok {
				done = false
			}
		}

		if !done {
			pending = append(pending, goal.Name)
		}
	}

	return pending
}

/* ITEM THE STORAGE DEFINITION */

// Defines parameters.
type Item struct {
	Name string

	OpenDoors      bool
	PutOn          bool
	Carry          bool
	GivesInventory bool
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

func (i Item) GetGivesInventory() bool {
	return i.GivesInventory
}

/* DOOR THE STORAGE DEFINITION */

// Defines parameters.
type Door struct {
	Name   string
	Status bool
}

func (d *Door) GetDoorStatus() bool {
	return d.Status
}

func (d *Door) ChangeStatus(value bool) {
	d.Status = value
}

/* ROOM THE STATE STORAGE */

var errRoomItemNotFound = errors.New("item is not in the room")

// A piece of furniture and the items lying on it,
// kept in order so the output is deterministic.
type Furniture struct {
	Place string // full phrase, e.g. "на столе"
	Items []string
}

// A way out of a room: Label is what the player types,
// Room is the room it leads to.
type Gateway struct {
	Label string
	Room  string
}

// Responsible for storing room
// state.
type Room struct {
	Name      string
	Furniture []Furniture
	Door      string
	Gateways  []Gateway

	EntryWelcome   string
	LookoutWelcome string

	// Optional room-specific addition to the lookout text.
	LookoutExtra func(p *Player) string
}

func (r *Room) GetName() string {
	return r.Name
}

func (r *Room) GetFurniture() []Furniture {
	return r.Furniture
}

func (r *Room) RemoveItem(itemName string) error {
	for i := range r.Furniture {
		items := r.Furniture[i].Items
		if idx := slices.Index(items, itemName); idx >= 0 {
			r.Furniture[i].Items = slices.Delete(items, idx, idx+1)
			return nil
		}
	}

	return errRoomItemNotFound
}

func (r *Room) GetDoor() string {
	return r.Door
}

func (r *Room) GetGateways() []Gateway {
	return r.Gateways
}

func (r *Room) GatewayLabels() []string {
	labels := make([]string, 0, len(r.Gateways))
	for _, gateway := range r.Gateways {
		labels = append(labels, gateway.Label)
	}

	return labels
}

func (r *Room) GetEntryWelcome() string {
	return r.EntryWelcome
}

func (r *Room) GetLookoutWelcome() string {
	return r.LookoutWelcome
}

func (r *Room) GetLookoutExtra() func(p *Player) string {
	return r.LookoutExtra
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
	}

	return w.message
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
		return nil, errors.New("command does not exists")
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
	command := strings.Fields(s)
	if len(command) < 1 {
		return nil, errors.New("no command")
	}

	return command, nil
}
