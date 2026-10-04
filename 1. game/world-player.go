package main

// Player Object

type Player struct {
	Inventory          map[string]*Item
	InventoryAvailable bool
	Position           *Room
	Goals              []map[string][]string
}

func (p *Player) CreateInventory() error {
	if p.InventoryAvailable {
		return ErrInventoryAvailable
	}

	p.InventoryAvailable = true

	return nil
}

func (p *Player) RemoveInventory() error {
	if !p.InventoryAvailable {
		return ErrInventoryNotAvailable
	}

	p.InventoryAvailable = false

	return nil
}

func (p *Player) TakeItem(name string) (*Item, error) {
	if !p.InventoryAvailable {
		return nil, ErrInventoryNotFound
	}

	item, ok := p.Inventory[name]
	if !ok {
		return nil, ErrInventoryItemNotFound
	}

	delete(p.Inventory, name)

	return item, nil
}

func (p *Player) PutItem(i *Item) error {
	if !p.InventoryAvailable {
		return ErrInventoryNotFound
	}

	_, ok := p.Inventory[i.Name]
	if ok {
		return ErrInventoryAlreadyHasItem
	}

	p.Inventory[i.Name] = i

	return nil
}
