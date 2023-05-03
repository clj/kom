package main

import (
	"fmt"
	"strconv"
	"strings"

	"go.riyazali.net/sqlite"
)

var (
	Version   string = "dev"
	Commit    string = "?"
	BuildDate string = "?"
)

type Part map[string]any
type Parts []Part

type KomPlugin interface {
	Init(KomPluginApi, PluginArguments) error
	ColumnNames() []string
	GetParts(any) (Parts, error)
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

func GetValue(value sqlite.Value) any {
	switch value.Type() {
	case sqlite.SQLITE_INTEGER:
		return value.Int64()
	case sqlite.SQLITE_FLOAT:
		return value.Float()
	case sqlite.SQLITE_TEXT:
		return value.Text()
	default:
		panic(fmt.Sprintf("unknown type %s", value.Type().String()))
	}
}

func Convert(value any, t string) (any, error) {

	switch t {
	case "int":
	case "float":
	case "string":
	case "":
		return value, nil
	default:
		return nil, fmt.Errorf("invalid destination type %s", t)

	}

	switch v := value.(type) {
	case int64:
		switch t {
		case "int":
			return v, nil
		case "float":
			return float64(v), nil
		case "string":
			return strconv.FormatInt(v, 10), nil
		}
	case float64:
		switch t {
		case "int":
			return int64(v), nil
		case "float":
			return v, nil
		case "string":
			return strconv.FormatFloat(v, 'G', -1, 64), nil
		}
	case string:
		switch t {
		case "int":
			return strconv.ParseInt(v, 0, 64)
		case "float":
			return strconv.ParseFloat(v, 64)
		case "string":
			return v, nil
		}
	case bool:
		switch t {
		case "int":
			if v {
				return 1, nil
			}
			return 0, nil
		case "float":
			if v {
				return 1.0, nil
			}
			return 0.0, nil
		case "string":
			if v {
				return "1", nil
			}
			return "0", nil
		}
	}

	return nil, fmt.Errorf("unknown source type %T for value %v", value, value)
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
	parts  Parts
	rowId  int64
}

func (c *KomCursor) Next() error {

	c.rowId += 1

	return sqlite.SQLITE_OK
}

func (c *KomCursor) Column(ctx *sqlite.VirtualTableContext, i int) error {

	columns := c.plugin.ColumnNames()

	switch value := c.parts[c.rowId][columns[i]].(type) {
	case int:
		ctx.ResultInt(value)
	case int64:
		ctx.ResultInt64(value)
	case float64:
		ctx.ResultFloat(value)
	case string:
		ctx.ResultText(value)
	case nil:
		ctx.ResultNull()
	default:
		return fmt.Errorf("unknown type: %T", value)
	}

	return nil
}

func (c *KomCursor) Filter(indexNumber int, indexString string, values ...sqlite.Value) error {
	var pkValue any = nil
	if len(values) != 0 {
		pkValue = GetValue(values[0])
	}
	parts, err := c.plugin.GetParts(pkValue)
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

type VersionFn struct{}

func (m *VersionFn) Args() int           { return -1 }
func (m *VersionFn) Deterministic() bool { return true }
func (m *VersionFn) Apply(ctx *sqlite.Context, values ...sqlite.Value) {
	if len(values) == 1 {
		switch values[0].Text() {
		case "version":
			ctx.ResultText(Version)
			return
		case "sha":
			ctx.ResultText(Commit)
			return
		case "build_date":
			ctx.ResultText(BuildDate)
			return
		}
	}

	ctx.ResultText(fmt.Sprintf("version: %s sha: %s build_date: %s", Version, Commit, BuildDate))
}

func init() {
	sqlite.Register(func(api *sqlite.ExtensionApi) (sqlite.ErrorCode, error) {
		module := &KomModule{}
		module.Init(api)
		if err := api.CreateModule("kom", module); err != nil {
			return sqlite.SQLITE_ERROR, err
		}
		if err := api.CreateFunction("kom_version", &VersionFn{}); err != nil {
			return sqlite.SQLITE_ERROR, err
		}
		return sqlite.SQLITE_OK, nil
	})
}

func main() {}
