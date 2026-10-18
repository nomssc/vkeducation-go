// Package main implements a small Multi User Dungeon style text game.
//
// The game world consists of rooms connected by gateways, items that lie on
// pieces of furniture, doors that can be opened, and a player with an
// inventory, clothes and goals. The player controls the game with text
// commands such as "осмотреться" (look around), "идти коридор" (walk to the
// hallway), "взять ключи" (take the keys), "надеть рюкзак" (put on the
// backpack) and "применить ключи дверь" (use the keys on the door).
//
// The code is split into layers:
//   - CLI: parses a command line and routes it to a handler;
//   - transport: validates arguments and turns results and errors into
//     player-facing text;
//   - service (use cases): contains the game logic of every command;
//   - world: stores the state of the player, rooms, items and doors.
//
// The whole world is described in initGame, so rooms, doors and items can be
// added or removed without touching the rest of the code.
package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// main is intentionally empty: the game is driven through handleCommand, which
// is called by the tests.
func main() {}

// initGame creates a new world with a fresh player, rooms, items and doors, and
// a CLI with all game commands attached. It replaces the global world and cli,
// so calling it again restarts the game from the beginning.
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
		EmptyText:    "ничего интересного",
	}
	bedroom := &Room{
		Name: roomNameBedroom,
		Furniture: []Furniture{
			{Place: "на столе", Items: []string{itemKeys.Name, itemWriting.Name}},
			{Place: "на стуле", Items: []string{itemBackpack.Name}},
		},
		Gateways: []Gateway{{Label: roomNameHallway, Room: roomNameHallway}},

		EntryWelcome: "ты в своей комнате",
		EmptyText:    "пустая комната",
	}
	street := &Room{
		Name:     roomNameStreet,
		Door:     doorStreet.Name,
		Gateways: []Gateway{{Label: "домой", Room: roomNameHallway}},

		EntryWelcome: "на улице весна",
		EmptyText:    "на улице весна",
	}

	player := &Player{
		Name:      "@nomssc",
		Inventory: map[string]struct{}{},
		Clothes:   map[string]struct{}{},
		Position:  roomNameKitchen,
		Goals: []Goal{
			{Name: "собрать рюкзак", Done: func(p *Player) bool { return p.GetInventoryAvailable() }},
			{Name: "идти в универ"},
		},
	}

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

// Errors of the transport layer. Their text is shown to the player as is.
var (
	ErrTransportUnexpectedStuff            = errors.New("неожиданная ошибка")
	ErrTransportIncorrectNumberOfArguments = errors.New("не хватает параметров")
	ErrTransportInventoryNotAvailable      = errors.New("некуда класть")
	ErrTransportItemIsNotInTheInventory    = errors.New("нет предмета в инвентаре")
	ErrTransportItemIsNotInTheRoom         = errors.New("нет такого")
	ErrTransportNoGateway                  = errors.New("нет пути")
	ErrTransportDoorIsClosed               = errors.New("дверь закрыта")
	ErrTransportRoomDoesNotExists          = errors.New("комната не существует")
	ErrTransportDoorDoesNotExists          = errors.New("дверь не существует")
	ErrTransportItemDoesNotExists          = errors.New("нет такого")
	ErrTransportItemCantBeCarried          = errors.New("нельзя взять")
	ErrTransportItemCantBePutOn            = errors.New("нельзя надеть")
	ErrTransportItemCantBeUsed             = errors.New("не к чему применить")
)

// LookoutTransport is the transport of the "осмотреться" command. It validates
// the request and passes it to LookoutUsecase.
type LookoutTransport struct {
	Usecase *LookoutUsecase

	ArgumentsCount int
}

// Handler handles the "осмотреться" command: it writes the description of the
// current room, or a player-facing error.
func (t *LookoutTransport) Handler(r Request, w *ResponseWriter) {
	if len(r.Args()) < t.ArgumentsCount {
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

// OpenTransport is the transport of the "применить" command. It validates the
// request and passes it to OpenUsecase.
type OpenTransport struct {
	Usecase *OpenUsecase

	ArgumentsCount int
}

// Handler handles the "применить <item> <object>" command: it writes a
// confirmation when the door gets opened, or a player-facing error.
func (t *OpenTransport) Handler(r Request, w *ResponseWriter) {
	if len(r.Args()) < t.ArgumentsCount {
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

// PutOnTransport is the transport of the "надеть" command. It validates the
// request and passes it to PutOnUsecase.
type PutOnTransport struct {
	Usecase *PutOnUsecase

	ArgumentsCount int
}

// Handler handles the "надеть <item>" command: it writes a confirmation when
// the item has been put on, or a player-facing error.
func (t *PutOnTransport) Handler(r Request, w *ResponseWriter) {
	if len(r.Args()) < t.ArgumentsCount {
		w.Error(ErrTransportIncorrectNumberOfArguments)
		return
	}

	err := t.Usecase.PutOn(r.Args()[0])
	if err != nil {
		switch {
		case errors.Is(err, ErrServiceItemDoesNotExists):
			w.Error(ErrTransportItemDoesNotExists)
		case errors.Is(err, ErrServiceItemCantBePutOn):
			w.Error(fmt.Errorf("%w: %s", ErrTransportItemCantBePutOn, r.Args()[0]))
		case errors.Is(err, ErrServiceRoomDoesNotExists):
			w.Error(ErrTransportRoomDoesNotExists)
		case errors.Is(err, ErrServiceItemIsNotInTheRoom):
			w.Error(ErrTransportItemIsNotInTheRoom)
		default:
			w.Error(ErrTransportUnexpectedStuff)
		}

		return
	}

	w.Write(fmt.Sprintf("вы надели: %s", r.Args()[0]))
}

// TakeTransport is the transport of the "взять" command. It validates the
// request and passes it to TakeUsecase.
type TakeTransport struct {
	Usecase *TakeUsecase

	ArgumentsCount int
}

// Handler handles the "взять <item>" command: it writes a confirmation when the
// item has been added to the inventory, or a player-facing error.
func (t *TakeTransport) Handler(r Request, w *ResponseWriter) {
	if len(r.Args()) < t.ArgumentsCount {
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
			w.Error(fmt.Errorf("%w: %s", ErrTransportItemCantBeCarried, r.Args()[0]))
		default:
			w.Error(ErrTransportUnexpectedStuff)
		}

		return
	}

	w.Write(fmt.Sprintf("предмет добавлен в инвентарь: %s", r.Args()[0]))
}

// WalkTransport is the transport of the "идти" command. It validates the
// request and passes it to WalkUsecase.
type WalkTransport struct {
	Usecase *WalkUsecase

	ArgumentsCount int
}

// Handler handles the "идти <direction>" command: it writes the welcome text of
// the new room followed by the available gateways, or a player-facing error.
func (t *WalkTransport) Handler(r Request, w *ResponseWriter) {
	if len(r.Args()) < t.ArgumentsCount {
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

// Errors of the service layer. The transport layer translates them into
// player-facing errors.
var (
	ErrServiceNoGateway               = errors.New("no gateway")
	ErrServiceDoorDoesNotExists       = errors.New("door does not exists")
	ErrServiceRoomDoesNotExists       = errors.New("room does not exists")
	ErrServiceItemDoesNotExists       = errors.New("item does not exists")
	ErrServiceInventoryNotAvailable   = errors.New("inventory not available")
	ErrServiceDoorIsClosed            = errors.New("door is closed")
	ErrServiceItemIsNotInTheInventory = errors.New("item is not the inventory")
	ErrServiceItemIsNotInTheRoom      = errors.New("item is not the room")
	ErrServiceItemCantBeCarried       = errors.New("item can't be carried")
	ErrServiceItemCantBePutOn         = errors.New("item can't be put on")
	ErrServiceItemCantOpenDoors       = errors.New("item can't open doors")
	ErrServiceObjectIsNotDoor         = errors.New("object is not a door")
)

// LookoutUsecase implements the logic of the "осмотреться" command.
type LookoutUsecase struct {
	World *World
}

// Lookout describes the room the player is in: the room's welcome text, the
// items lying on its furniture (or a text for an empty room), an optional
// room-specific addition, and the available gateways.
//
// It returns ErrServiceRoomDoesNotExists if the player's position is unknown.
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
		empty := room.GetEmptyText()
		if empty == "" {
			empty = "пустая " + room.GetName()
		}
		segments = append(segments, empty)
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

// OpenUsecase implements the logic of the "применить" command.
type OpenUsecase struct {
	World *World
}

// Open uses the item itemName from the player's inventory to open the door
// object. The item must be able to open doors, and the door must belong to a
// room reachable from the player's current room.
//
// It returns one of ErrServiceItemIsNotInTheInventory,
// ErrServiceItemDoesNotExists, ErrServiceItemCantOpenDoors,
// ErrServiceObjectIsNotDoor or ErrServiceRoomDoesNotExists on failure.
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

// PutOnUsecase implements the logic of the "надеть" command.
type PutOnUsecase struct {
	World *World
}

// PutOn takes the item itemName from the current room and puts it on the
// player. If the item gives an inventory (like a backpack), the player can
// carry items afterwards.
//
// It returns ErrServiceItemDoesNotExists, ErrServiceRoomDoesNotExists,
// ErrServiceItemIsNotInTheRoom or ErrServiceItemCantBePutOn on failure.
func (u *PutOnUsecase) PutOn(itemName string) error {
	item, ok := u.World.GetItems()[itemName]
	if !ok {
		return ErrServiceItemDoesNotExists
	}
	room, ok := u.World.GetRooms()[u.World.Player.GetPosition()]
	if !ok {
		return ErrServiceRoomDoesNotExists
	}

	if !room.HasItem(itemName) {
		return ErrServiceItemIsNotInTheRoom
	}
	if !item.GetPutOn() {
		return ErrServiceItemCantBePutOn
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

// TakeUsecase implements the logic of the "взять" command.
type TakeUsecase struct {
	World *World
}

// Take moves the item itemName from the current room to the player's
// inventory. The item must lie in the room, be carriable, and the player must
// already have an inventory.
//
// It returns ErrServiceItemDoesNotExists, ErrServiceRoomDoesNotExists,
// ErrServiceItemIsNotInTheRoom, ErrServiceItemCantBeCarried or
// ErrServiceInventoryNotAvailable on failure.
func (u *TakeUsecase) Take(itemName string) error {
	item, ok := u.World.GetItems()[itemName]
	if !ok {
		return ErrServiceItemDoesNotExists
	}
	room, ok := u.World.GetRooms()[u.World.Player.GetPosition()]
	if !ok {
		return ErrServiceRoomDoesNotExists
	}

	if !room.HasItem(itemName) {
		return ErrServiceItemIsNotInTheRoom
	}
	if !item.GetCarry() {
		return ErrServiceItemCantBeCarried
	}

	if !u.World.Player.GetInventoryAvailable() {
		return ErrServiceInventoryNotAvailable
	}

	if err := room.RemoveItem(itemName); err != nil {
		return ErrServiceItemIsNotInTheRoom
	}

	u.World.Player.StoreItem(itemName)

	return nil
}

// WalkUsecase implements the logic of the "идти" command.
type WalkUsecase struct {
	World *World
}

// Walk moves the player through the gateway of the current room whose label is
// direction. It returns the welcome text of the new room and a comma-separated
// list of the gateway labels leading out of it.
//
// It returns ErrServiceRoomDoesNotExists, ErrServiceNoGateway,
// ErrServiceDoorDoesNotExists or ErrServiceDoorIsClosed on failure; in that
// case the player does not move.
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

// world is the global game world, created by initGame.
var world *World

// ConfigWorld is used to configure the world on initialization.
type ConfigWorld struct {
	World
}

// World stores the whole game world state: the player and all items, doors and
// rooms, indexed by name.
type World struct {
	Player *Player

	Items map[string]Item
	Doors map[string]*Door
	Rooms map[string]*Room
}

// NewWorld creates a World from the given configuration.
func NewWorld(c ConfigWorld) *World {
	return &World{
		Player: c.Player,

		Items: c.Items,
		Doors: c.Doors,
		Rooms: c.Rooms,
	}
}

// GetItems returns all items of the world indexed by name.
func (w *World) GetItems() map[string]Item {
	return w.Items
}

// GetDoors returns all doors of the world indexed by name.
func (w *World) GetDoors() map[string]*Door {
	return w.Doors
}

// GetRooms returns all rooms of the world indexed by name.
func (w *World) GetRooms() map[string]*Room {
	return w.Rooms
}

// Goal is something the player has to do. A goal with a nil Done function is
// never completed.
type Goal struct {
	Name string
	Done func(p *Player) bool
}

// Player stores the state of the player.
type Player struct {
	Name string
	// Inventory is the set of names of the items the player carries.
	Inventory map[string]struct{}
	// Clothes is the set of names of the items the player wears.
	Clothes map[string]struct{}
	// Position is the name of the room the player is in.
	Position string
	// Goals are the things the player has to do.
	Goals []Goal

	// InventoryAvailable reports whether the player can carry items, that is,
	// whether they are wearing something that gives an inventory.
	InventoryAvailable bool
}

// GetName returns the player's name.
func (p *Player) GetName() string {
	return p.Name
}

// GetInventory returns the set of names of the items the player carries.
func (p *Player) GetInventory() map[string]struct{} {
	return p.Inventory
}

// StoreItem puts the item itemName into the player's inventory.
func (p *Player) StoreItem(itemName string) {
	p.Inventory[itemName] = struct{}{}
}

// GetClothes returns the set of names of the items the player wears.
func (p *Player) GetClothes() map[string]struct{} {
	return p.Clothes
}

// StoreWear marks the item itemName as worn by the player.
func (p *Player) StoreWear(itemName string) {
	p.Clothes[itemName] = struct{}{}
}

// GetPosition returns the name of the room the player is in.
func (p *Player) GetPosition() string {
	return p.Position
}

// ChangePosition moves the player to the room roomName.
func (p *Player) ChangePosition(roomName string) {
	p.Position = roomName
}

// GetInventoryAvailable reports whether the player can carry items.
func (p *Player) GetInventoryAvailable() bool {
	return p.InventoryAvailable
}

// ChangeInventoryAvailability sets whether the player can carry items.
func (p *Player) ChangeInventoryAvailability(value bool) {
	p.InventoryAvailable = value
}

// PendingGoals returns the names of the goals that are not complete yet, in
// declaration order.
func (p *Player) PendingGoals() []string {
	pending := make([]string, 0, len(p.Goals))

	for _, goal := range p.Goals {
		if goal.Done == nil || !goal.Done(p) {
			pending = append(pending, goal.Name)
		}
	}

	return pending
}

// Item defines the parameters of an item: what the player can do with it.
type Item struct {
	Name string

	// OpenDoors tells that the item can open doors.
	OpenDoors bool
	// PutOn tells that the item can be put on.
	PutOn bool
	// Carry tells that the item can be put into the inventory.
	Carry bool
	// GivesInventory tells that wearing the item gives the player an inventory.
	GivesInventory bool
}

// GetName returns the item's name.
func (i Item) GetName() string {
	return i.Name
}

// GetOpenDoors reports whether the item can open doors.
func (i Item) GetOpenDoors() bool {
	return i.OpenDoors
}

// GetPutOn reports whether the item can be put on.
func (i Item) GetPutOn() bool {
	return i.PutOn
}

// GetCarry reports whether the item can be put into the inventory.
func (i Item) GetCarry() bool {
	return i.Carry
}

// GetGivesInventory reports whether wearing the item gives an inventory.
func (i Item) GetGivesInventory() bool {
	return i.GivesInventory
}

// Door defines the parameters of a door. Status is true when the door is open.
type Door struct {
	Name   string
	Status bool
}

// GetDoorStatus reports whether the door is open.
func (d *Door) GetDoorStatus() bool {
	return d.Status
}

// ChangeStatus opens (true) or closes (false) the door.
func (d *Door) ChangeStatus(value bool) {
	d.Status = value
}

// errRoomItemNotFound is returned by Room.RemoveItem when the room does not
// contain the requested item.
var errRoomItemNotFound = errors.New("item is not in the room")

// Furniture is a piece of furniture and the items lying on it, kept in order so
// the output is deterministic.
type Furniture struct {
	Place string
	Items []string
}

// Gateway is a way out of a room: Label is what the player types, Room is the
// room it leads to.
type Gateway struct {
	Label string
	Room  string
}

// Room stores the state of a room.
type Room struct {
	Name      string
	Furniture []Furniture
	// Door is the name of the door that has to be open to enter the room.
	// It is empty for rooms without a door.
	Door     string
	Gateways []Gateway

	// EntryWelcome is the text shown when the player enters the room.
	EntryWelcome string
	// LookoutWelcome is the text that starts the room description on lookout.
	LookoutWelcome string
	EmptyText      string // what the lookout says when nothing lies in the room

	// LookoutExtra is an optional room-specific addition to the lookout text.
	LookoutExtra func(p *Player) string
}

// GetName returns the room's name.
func (r *Room) GetName() string {
	return r.Name
}

// GetFurniture returns the furniture of the room.
func (r *Room) GetFurniture() []Furniture {
	return r.Furniture
}

// RemoveItem removes the item itemName from the furniture of the room. It
// returns errRoomItemNotFound if there is no such item in the room.
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

// GetDoor returns the name of the door of the room, or an empty string if the
// room has no door.
func (r *Room) GetDoor() string {
	return r.Door
}

// GetGateways returns the gateways leading out of the room.
func (r *Room) GetGateways() []Gateway {
	return r.Gateways
}

// GatewayLabels returns the labels of all gateways of the room, in order.
func (r *Room) GatewayLabels() []string {
	labels := make([]string, 0, len(r.Gateways))
	for _, gateway := range r.Gateways {
		labels = append(labels, gateway.Label)
	}

	return labels
}

// GetEntryWelcome returns the text shown when the player enters the room.
func (r *Room) GetEntryWelcome() string {
	return r.EntryWelcome
}

// GetLookoutWelcome returns the text that starts the lookout description.
func (r *Room) GetLookoutWelcome() string {
	return r.LookoutWelcome
}

// GetEmptyText returns the text shown by lookout when nothing lies in the room.
func (r *Room) GetEmptyText() string {
	return r.EmptyText
}

// HasItem reports whether the item itemName lies on any furniture of the room.
func (r *Room) HasItem(itemName string) bool {
	for _, f := range r.Furniture {
		if slices.Contains(f.Items, itemName) {
			return true
		}
	}

	return false
}

// GetLookoutExtra returns the optional room-specific lookout addition, or nil.
func (r *Room) GetLookoutExtra() func(p *Player) string {
	return r.LookoutExtra
}

// Errors of the CLI layer. Their text is shown to the player as is.
var (
	ErrCLINoCommand            = errors.New("команда не предоставлена")
	ErrCLICommandDoesNotExists = errors.New("неизвестная команда")
)

// cli is the global command line interface, created by initGame.
var cli *CLI

// Request stores the state of a request throughout handling the command.
type Request struct {
	command string
	args    []string
}

// Command returns the name of the command.
func (c Request) Command() string {
	return c.command
}

// Args returns the arguments of the command.
func (c Request) Args() []string {
	return c.args
}

// ResponseWriter stores the response of a handler: either a message or an error.
type ResponseWriter struct {
	message string
	error   error
}

// Write sets the response message.
func (r *ResponseWriter) Write(s string) {
	r.message = s
}

// Error sets the response error. If an error is set, it takes precedence over
// the message.
func (r *ResponseWriter) Error(e error) {
	r.error = e
}

// Handler is a function that handles a command.
type Handler func(r Request, w *ResponseWriter)

// ConfigCLI is used to configure the CLI on initialization.
type ConfigCLI struct {
	CLI
}

// Router attaches handlers to commands and finds a handler by command name.
type Router interface {
	Attach(command string, h Handler)
	Process(command string) (Handler, error)
}

// Interpreter splits a raw command string into its parts.
type Interpreter interface {
	UnparseCommandString(s string) ([]string, error)
}

// CLI controls the program life and manages all its operations: it parses
// commands with Codec and routes them with Mux.
type CLI struct {
	Mux   Router
	Codec Interpreter
}

// NewCLI creates a CLI from the given configuration.
func NewCLI(c ConfigCLI) *CLI {
	return &CLI{
		Mux:   c.CLI.Mux,
		Codec: c.CLI.Codec,
	}
}

// handleCommand executes one line typed by the player, in the form
// "<command> <arg1> <arg2> ...", and returns the text to show to the player.
// An empty line and an unknown command produce ErrCLINoCommand and
// ErrCLICommandDoesNotExists respectively.
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

// Handle attaches the handler h to the command.
func (c *CLI) Handle(command string, h Handler) {
	c.Mux.Attach(command, h)
}

// ConfigMux is used to configure the Mux on initialization.
type ConfigMux struct {
	Mux
}

// Mux manages command handlers.
type Mux struct {
	handlers map[string]Handler
}

// NewMux creates an empty Mux.
func NewMux(c ...ConfigMux) *Mux {
	return &Mux{
		handlers: make(map[string]Handler),
	}
}

// Attach registers the handler h for the command, replacing any previous one.
func (m *Mux) Attach(command string, h Handler) {
	m.handlers[command] = h
}

// Process returns the handler registered for the command, or an error if there
// is none.
func (m *Mux) Process(command string) (Handler, error) {
	handler, ok := m.handlers[command]
	if !ok {
		return nil, errors.New("command does not exists")
	}

	return handler, nil
}

// ConfigCodec is used to configure the Codec on initialization.
type ConfigCodec struct {
	Codec
}

// Codec parses and unparses inputs.
type Codec struct{}

// NewCodec creates a Codec.
func NewCodec(c ...ConfigCodec) Codec {
	return Codec{}
}

// UnparseCommandString splits s into whitespace-separated fields: the command
// followed by its arguments. It returns an error if s has no fields.
func (c Codec) UnparseCommandString(s string) ([]string, error) {
	command := strings.Fields(s)
	if len(command) < 1 {
		return nil, errors.New("no command")
	}

	return command, nil
}
