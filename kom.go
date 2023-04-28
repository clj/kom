package main

import (
	"strings"

	"go.riyazali.net/sqlite"
)

type KomPlugin interface {
	Init()
}

type KomModule struct {
	plugin KomPlugin
}

func (m *KomModule) Connect(_ *sqlite.Conn, args []string, declare func(string) error) (_ sqlite.VirtualTable, err error) {
	var table = &KomVirtualTable{}
	pluginArgs := args[3:]

	// Values in args:
	//
	// The first string, argv[0], is the name of the module being invoked. The module name
	// is the name provided as the second argument to sqlite3_create_module() and as the
	// argument to the USING clause of the CREATE VIRTUAL TABLE statement that is running.
	// The second, argv[1], is the name of the database in which the new virtual table is
	// being created. The database name is "main" for the primary database, or "temp" for
	// TEMP database, or the name given at the end of the ATTACH statement for attached
	// databases. The third element of the array, argv[2], is the name of the new virtual
	// table, as specified following the TABLE keyword in the CREATE VIRTUAL TABLE statement.
	// If present, the fourth and subsequent strings in the argv[] array report the arguments
	// to the module name in the CREATE VIRTUAL TABLE statement.
	// From xCreate docs: https://www.sqlite.org/vtab.html#xcreate

	for i := range pluginArgs {
		s := strings.SplitN(pluginArgs[i], "=", 2)
		switch s[0] {
		case "plugin":
			if pluginMaker, ok := plugins[s[1]]; ok {
				m.plugin = pluginMaker()
				m.plugin.Init()
			}
		}
	}

	if m.plugin == nil {
		return nil, sqlite.Error(sqlite.SQLITE_ERROR, "plugin is a required argument")
	}
	return table, declare("CREATE TABLE x(c1)")
}

type KomVirtualTable struct {
}

func (vt *KomVirtualTable) BestIndex(_ *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	return &sqlite.IndexInfoOutput{EstimatedCost: 1000000}, nil
}
func (vt *KomVirtualTable) Open() (_ sqlite.VirtualCursor, err error) { return &KomCursor{}, nil }
func (vt *KomVirtualTable) Disconnect() error                         { return nil }
func (vt *KomVirtualTable) Destroy() error                            { return vt.Disconnect() }

type KomCursor struct {
}

func (c *KomCursor) Next() error                                         { return sqlite.SQLITE_OK }
func (c *KomCursor) Column(ctx *sqlite.VirtualTableContext, i int) error { return nil }
func (c *KomCursor) Filter(int, string, ...sqlite.Value) error           { return sqlite.SQLITE_OK }
func (c *KomCursor) Rowid() (int64, error)                               { return -1, sqlite.SQLITE_EMPTY }
func (c *KomCursor) Eof() bool                                           { return true }
func (c *KomCursor) Close() error                                        { return nil }

func init() {
	sqlite.Register(func(api *sqlite.ExtensionApi) (sqlite.ErrorCode, error) {
		if err := api.CreateModule("kom", &KomModule{}); err != nil {
			return sqlite.SQLITE_ERROR, err
		}
		return sqlite.SQLITE_OK, nil
	})
}

func main() {}
