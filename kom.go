package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"go.riyazali.net/sqlite"
)

type Value = sqlite.Value

type KomPlugin interface {
	Init(KomPluginApi, PluginArguments) error
	ColumnNames() []string
	GetParts(*Value) ([]map[string]string, error)
	CanFilter(string) bool
}

type KomPluginApi interface {
	ReadSetting(string) (string, error)
	WriteSetting(string, string) error
	DeleteSetting(string) error
}

type PluginApi struct {
	settingsTableName string
	sqliteApi         *sqlite.ExtensionApi
}

func (api *PluginApi) Init(sqliteApi *sqlite.ExtensionApi, settingsTableName string) error {
	api.sqliteApi = sqliteApi
	api.settingsTableName = settingsTableName

	return nil
}

func (api *PluginApi) ReadSetting(key string) (string, error) {
	conn := api.sqliteApi.Connection()

	stmt, _, err := conn.Prepare(fmt.Sprintf("SELECT value FROM %s WHERE key=$1 LIMIT 2", api.settingsTableName))
	if err != nil {
		return "", err
	}
	defer stmt.Finalize()

	stmt.BindText(1, key)
	rowReturned, err := stmt.Step()
	if err != nil {
		return "", err
	}
	if !rowReturned {
		return "", sqlite.Error(sqlite.SQLITE_ERROR, fmt.Sprintf("no setting found in %s for %s", api.settingsTableName, key))
	}
	value := stmt.GetText("value")

	rowReturned, _ = stmt.Step()
	if rowReturned {
		return "", sqlite.Error(sqlite.SQLITE_ERROR, fmt.Sprintf("multiple setting found in %s for %s", api.settingsTableName, key))

	}
	return value, nil
}

func (api *PluginApi) WriteSetting(key string, value string) error {
	conn := api.sqliteApi.Connection()

	stmt, _, err := conn.Prepare(fmt.Sprintf("INSERT INTO %s VALUES ($1, $2)", api.settingsTableName))

	if err != nil {
		return err
	}
	defer stmt.Finalize()

	stmt.BindText(1, key)
	stmt.BindText(2, value)
	_, err = stmt.Step()
	if err != nil {
		return err
	}

	return nil
}

func (api *PluginApi) DeleteSetting(key string) error {
	conn := api.sqliteApi.Connection()

	stmt, _, err := conn.Prepare(fmt.Sprintf("DELETE FROM %s WHERE key=$1", api.settingsTableName))

	if err != nil {
		return err
	}
	defer stmt.Finalize()

	stmt.BindText(1, key)
	_, err = stmt.Step()
	if err != nil {
		return err
	}

	return nil
}

func (api *PluginApi) Destroy() {}

type PluginArguments map[string]string

type KomModule struct {
	sqliteApi *sqlite.ExtensionApi
}

func (m *KomModule) Init(sqliteApi *sqlite.ExtensionApi) {
	m.sqliteApi = sqliteApi
}

func maybeUnquote(literal string) string {
	literal = strings.TrimSpace(literal)

	if len(literal) < 2 {
		return literal
	}

	if literal[0] == '\'' && literal[len(literal)-1] == '\'' {
		return strings.ReplaceAll(literal[1:len(literal)-1], "''", "'")
	}

	if literal[0] == '"' && literal[len(literal)-1] == '"' {
		return strings.ReplaceAll(literal[1:len(literal)-1], "\"\"", "\"\"")
	}

	return literal
}

func (m *KomModule) Connect(conn *sqlite.Conn, args []string, declare func(string) error) (_ sqlite.VirtualTable, err error) {
	table := &KomVirtualTable{}
	pluginArgs := args[3:]
	settingsTableName := ""
	pluginArguments := make(PluginArguments)

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
		s[1] = maybeUnquote(s[1])
		switch s[0] {
		case "plugin":
			if pluginMaker, ok := plugins[s[1]]; ok {
				table.plugin = pluginMaker()
			}
		case "settings":
			settingsTableName = s[1]
		default:
			pluginArguments[s[0]] = s[1]
		}
	}

	if table.plugin == nil {
		return nil, sqlite.Error(sqlite.SQLITE_ERROR, "plugin is a required argument")
	}
	if settingsTableName == "" {
		return nil, sqlite.Error(sqlite.SQLITE_ERROR, "settings is a required argument")
	}

	if err := conn.Exec(fmt.Sprintf("SELECT key, value FROM %s LIMIT 1", settingsTableName), nil); err != nil {
		return nil, sqlite.Error(sqlite.SQLITE_ERROR, fmt.Sprintf("settings table %s does not exist or has wrong schema", settingsTableName))
	}

	api := &PluginApi{}
	if err := api.Init(m.sqliteApi, settingsTableName); err != nil {
		return nil, err
	}
	if err = table.plugin.Init(api, pluginArguments); err != nil {
		return nil, err
	}

	columnNames := strings.Join(table.plugin.ColumnNames(), ",")

	return table, declare(fmt.Sprintf("CREATE TABLE x(%s)", columnNames))
}

type KomVirtualTable struct {
	api    PluginApi
	plugin KomPlugin
}

func (vt *KomVirtualTable) BestIndex(info *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	output := &sqlite.IndexInfoOutput{EstimatedCost: 1000000, IndexNumber: -1}
	columns := vt.plugin.ColumnNames()

	for _, constraint := range info.Constraints {
		if !constraint.Usable || constraint.Op != sqlite.INDEX_CONSTRAINT_EQ {
			output.ConstraintUsage = append(output.ConstraintUsage, &sqlite.ConstraintUsage{
				ArgvIndex: 0,
				Omit:      false,
			})

			continue
		}

		if vt.plugin.CanFilter(columns[constraint.ColumnIndex]) {
			output.EstimatedCost = 100
			output.EstimatedRows = 1
			output.IndexNumber = constraint.ColumnIndex
			output.ConstraintUsage = append(output.ConstraintUsage, &sqlite.ConstraintUsage{
				ArgvIndex: 1,
				Omit:      false,
			})

			return output, nil
		}

	}
	return output, nil
}

func (vt *KomVirtualTable) Open() (_ sqlite.VirtualCursor, err error) {

	return &KomCursor{
		plugin: vt.plugin,
	}, nil
}

func (vt *KomVirtualTable) Disconnect() error {
	return nil
}

func (vt *KomVirtualTable) Destroy() error {

	vt.api.Destroy()
	return vt.Disconnect()
}

type KomCursor struct {
	plugin KomPlugin
	parts  []map[string]string
	rowId  int64
}

func (c *KomCursor) Next() error {

	c.rowId += 1

	return sqlite.SQLITE_OK
}

func (c *KomCursor) Column(ctx *sqlite.VirtualTableContext, i int) error {

	columns := c.plugin.ColumnNames()

	ctx.ResultText(c.parts[c.rowId][columns[i]])

	return nil
}

func (c *KomCursor) Filter(indexNumber int, indexString string, values ...sqlite.Value) error {
	var pkValue *sqlite.Value = nil
	if len(values) != 0 {
		pkValue = &values[0]
	}
	parts, err := c.plugin.GetParts((*Value)(pkValue))
	if err != nil {
		return err
	}
	c.parts = parts
	c.rowId = 0

	return sqlite.SQLITE_OK
}

func (c *KomCursor) Rowid() (int64, error) {
	return c.rowId, nil
}
func (c *KomCursor) Eof() bool {
	return c.rowId >= int64(len(c.parts))
}
func (c *KomCursor) Close() error {
	return nil
}

func init() {
	sqlite.Register(func(api *sqlite.ExtensionApi) (sqlite.ErrorCode, error) {
		module := &KomModule{}
		module.Init(api)
		if err := api.CreateModule("kom", module); err != nil {
			return sqlite.SQLITE_ERROR, err
		}
		return sqlite.SQLITE_OK, nil
	})
}

func main() {}
