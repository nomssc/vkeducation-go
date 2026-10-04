package main

type Room struct {
	Inventory map[string]*Object
	Gateways  map[string]*Room
}
