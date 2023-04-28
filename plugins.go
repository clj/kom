package main

var plugins = map[string]func() KomPlugin{
	"inventree": func() KomPlugin { return &InventreePlugin{} },
}
