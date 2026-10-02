package main

import "errors"

var (
	ErrAlreadyExists = errors.New("already exists")
	ErrNotFound      = errors.New("not found")
)

// OBJECTS

type Object struct {
	name string
}

func NewObject(name string) Object {
	return Object{name: name}
}

// ITEMS

type Item struct {
	name string
}

func NewItem(name string) Item {
	return Item{name: name}
}

// INVENTORY

type Inventory struct {
	store map[Item]struct{}
}

func NewInventory() *Inventory {
	return &Inventory{store: make(map[Item]struct{})}
}

func (i *Inventory) Put(item Item) error {
	_, ok := i.store[item]
	if !ok {
		i.store[item] = struct{}{}
	} else {
		return ErrAlreadyExists
	}

	return nil
}

func (i *Inventory) Take(item Item) error {
	_, ok := i.store[item]
	if !ok {
		return ErrNotFound
	} else {
		delete(i.store, item)
	}

	return nil
}

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
	/*
		эта функция инициализирует игровой мир - все комнаты
		если что-то было - оно корректно перезатирается
	*/
}

func handleCommand(command string) string {
	/*
		данная функция принимает команду от "пользователя"
		и наверняка вызывает какой-то другой метод или функцию у "мира" - списка комнат
	*/
	return "not implemented"
}
