package main

import (
	"errors"
	"strings"
)

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
