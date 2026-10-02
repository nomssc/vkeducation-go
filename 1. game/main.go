package main

import "errors"

var (
	ErrAlreadyExists  = errors.New("already exists")
	ErrNotFound       = errors.New("not found")
	ErrStatusMismatch = errors.New("status mismatch")
)

// ITEMS

type Item struct {
	name string
}

func NewItem(name string) Item {
	return Item{name: name}
}

// OBJECTS

type Object struct {
	name      string
	status    int
	inventory map[string]Item
}

func NewObject(name string) *Object {
	return &Object{
		name:      name,
		status:    0,
		inventory: make(map[string]Item),
	}
}

func (o *Object) CheckStatus(status int) error {
	if o.status != status {
		return ErrStatusMismatch
	} else {
		return nil
	}
}

func (o *Object) TakeItem(i string) (Item, error) {
	item, ok := o.inventory[i]
	if !ok {
		return Item{}, ErrNotFound
	} else {
		delete(o.inventory, i)
		return item, nil
	}
}

func (o *Object) PutItem(i Item) error {
	_, ok := o.inventory[i.name]
	if !ok {
		o.inventory[i.name] = i
		return nil
	} else {
		return ErrAlreadyExists
	}
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
