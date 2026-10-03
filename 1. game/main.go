package main

import (
	"errors"
	"strings"
)

var (
	ErrAlreadyExists  = errors.New("already exists")
	ErrNotFound       = errors.New("not found")
	ErrStatusMismatch = errors.New("status mismatch")

	ErrCommandNotFound = errors.New("command not found")
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

type Resources struct {
	TUI *TUI
}

type TUI struct {
	Parser Parser
}

type Parser interface {
	Unparse(command string) []string
}

// CODEC THE INPUT PARSER

type CodecConfig struct{}

type Codec struct{}

func NewCodec(c CodecConfig) Codec {
	return Codec{}
}

func (c Codec) Unparse(command string) []string {
	return strings.Split(command, " ")
}

func initGame() {
}

func handleCommand(command string) string {
	/*
		данная функция принимает команду от "пользователя"
		и наверняка вызывает какой-то другой метод или функцию у "мира" - списка комнат
	*/
	return "not implemented"
}
