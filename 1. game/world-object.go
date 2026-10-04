package main

type Object struct {
	Inventory  map[string]*Item
	Status     string
	AffectedBy map[string]*Item
	CanHold    map[string]*Item
}
