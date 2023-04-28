package main

import (
	"go.riyazali.net/sqlite"
)

type KomModule struct{}

func (m *KomModule) Connect(_ *sqlite.Conn, args []string, declare func(string) error) (_ sqlite.VirtualTable, err error) {
	var table = &KomVirtualTable{}

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
