package main

import "errors"

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

func NewWorld(c *ConfigWorld) *World {
	return &World{
		Player: c.World.Player,
		Rooms:  c.World.Rooms,
	}
}
