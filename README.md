# KiCad ODBC Middleware

> **Warning**
> This repository is archived, please see:
>
> * https://github.com/clj/kob
> * https://github.com/clj/kom2 (also archived)
>
> For further development of this idea.

An SQLite3 Virtual Table that can interface KiCad's Database Libraries to component libraries that don't speak SQL.

## Supported Backends

* [Inventree](https://inventree.org)

## Installing

### macOS

Install [unixODBC](https://www.unixodbc.org), [SQLiteODBC](https://ch-werner.homepage.t-online.de/sqliteodbc/html/index.html) and [SQLite3](https://www.sqlite.org). This can be done using [homebrew](https://brew.sh):

```shell
$ brew install unixodbc sqliteodbc sqlite3
```

You most likely have to install SQLite3 from homebew, even though macOS already comes with SQLite3. This is because the  version that comes as standard with the OS does not have the ability to load modules.

The homebrew SQLite3 command won't run by default. See `brew info sqlite3` for instructions on how to make it the default command, or use the full path, e.g.: `/usr/local/opt/sqlite/bin/sqlite3` when setting up the database below.

Download the latest kicad-odbc-middleware release from [Releases](https://github.com/clj/kom/releases):

* For Intel based Macs:
    * kicad-odbc-middleware-**macos**-**amd64**-VERSION.zip
* For Apple Silicon (ARM) Macs:
    * kicad-odbc-middleware-**macos**-**arm64**-VERSION.zip

decompress it and leave the kom.dylib file somewhere convenient.

### Linux

Contributions welcome.

### Windows

Contributions welcome.

## Configuration

### ODBC Configuration

Set up a datasource, by adding the appropriate configuration to `~/.odbc.ini`:

```ini
[inventree]
Driver=/usr/local/lib/libsqlite3odbc.dylib
Description=InvenTree Datasource
Database=/Users/me/inventree/inventree.db
LoadExt=/Users/me/inventree/kom
```

assuming that you have extracted the kicad-odbc-middleware to `/Users/me/inventree/` and that you will be creating a database `inventree.db` in the same location (see the next step.)

### Middleware Configuration

Open the SQLite database which will hold the configuration:

```shell
$ sqlite3 /Users/me/inventree/inventree.db
```
Create a settings table and add the required settings:

```sql
.load /Users/me/inventree/kom
CREATE TABLE settings (key, value);
INSERT INTO settings VALUES ("server", "http://localhost:45454");
INSERT INTO settings VALUES ("username", "user123");
INSERT INTO settings VALUES ("password", "veryverysecret!");
```

You can update these values at any time, though you will have to restart KiCad to use any new settings.

Create one or more database libraries:

```sql
CREATE VIRTUAL TABLE Resistors USING
    kom(plugin="inventree", settings="settings",
        categories="Resistors", default_symbol="Device:R",
        default_footprint="Resistor_SMD:R_0805_2012Metric",
        fields="Category:(int)category, Active:(int)active=(int)0, FullName:full_name, ClampingVoltage:parameters.Clamping Voltage"));
```

Available options:

* `plugin`:
    * The name of the plugin to use
    * Available plugins: inventree
* `settings`:
    * The name of the table that stores settings

Options specific to the **inventree** plugin:
* `categories`:
    * The category to return parts from
* `default_symbol` (optional):
    * The default symbol to return if no symbol is configured for a part
* `default_footprint` (optional):
    * The default footprint to return if no symbol is configured for a part
* `fields` (optional):
    * Define one or more additional fields to expose to KiCad, or redefine the default fields
    * Default fields are: PK, IPN, Name, Keywords, Description, Symbols, Footprints
    * Both PK and IPN can be used in the `key` setting in the kicad_dbl file (see below.). However when IPN is used, it is recommended that ["Allow duplicate IPN" setting](https://docs.inventree.org/en/latest/settings/global/#main-settings) in InvenTree is set to false.

If you need to reconfigure a table, simply `DROP TABLE ____` and recreate it with the desired options.

#### Caveats

* You have to configure a default_symbol to get any sensible behaviour at the moment
* Despite the configuration option `categories` being plural, only one category can be specified at a time for a table currently


#### Field Definitions

Fields are defined as a comma separated list of field definitions matching the following format:

```
destination:(type)source=(default_type)default
```

Where:

* `destination` - field name exposed to KiCad
* `source` - field name returned by the InvenTree API
* `type` - SQL type (int, float, string) to convert the returned value to
* `default` - default value to return if the field is not present in InvenTree response
* `default_type` - the type of the default value if it should not be a string

The `source` field can access nested structures (mappings only) can be accessed using dot notation, e.g.: `metadata.kicad.footprints`. Use one of the following key prefixes to access other information:

* `metadata.` - access data from the metadata table.
* `parameters.` - access part parameters.

When none of these prefixes are used, keys are looked up in the data returned by the Parts API. Metadata and parameters are only fetched when KiCad requests data for a specific part (by using a primary key constraint).

### KiCad Configuration

Create a `inventree.kicad_dbl` file with a valid configuration (see the [KiCad documentation on Database Libraries](https://docs.kicad.org/master/en/eeschema/eeschema.html#database-libraries)), e.g.:

```json
{
    "meta": {
        "version": 0
    },
    "name": "Inventree Library",
    "description": "Components pulled from Inventree",
    "source": {
        "type": "odbc",
        "dsn": "inventree",
        "timeout_seconds": 2
    },
    "libraries": [
        {
            "name": "Resistors",
            "table": "Resistors",
            "key": "IPN",
            "symbols": "Symbols",
            "footprints": "Footprints",
            "fields": [
                {
                    "column": "IPN",
                    "name": "IPN",
                    "visible_on_add": false,
                    "visible_in_chooser": true,
                    "show_name": true,
                    "inherit_properties": true
                },
                {
                    "column": "Value",
                    "name": "Value",
                    "visible_on_add": true,
                    "visible_in_chooser": true,
                    "show_name": false
                }
            ],
            "properties": {
                "description": "Description",
                "keywords": "Keywords"
            }
        }
    ]
}
```

The above configuration would work well with a table that has been created with the following parameters:

```sql
CREATE VIRTUAL TABLE Resistors USING
    kom(plugin=inventree, settings=settings,
    categories="Resistors", default_symbol="Device:R",
    fields="Value:parameters.Value")
```

Add the library to KiCad:

* *Preferences* -> *Manage Symbol Libraries...*
* Switch to the:
    * *Global Libraries*; or
    * *Project Specific Libraries*
* Add a new library
* Give it an appropriate *Nickname*
* Set the *Library Path* to point to the `inventree.kicad_dbl` that you created earlier
* Set the *Library Format* to *Database*

You can now open the Schematic Editor and add a new component. The configured library should now be available.

## License

MIT License Copyright (c) 2023 Christian Lyder Jacobsen

Refer to [LICENSE](./LICENSE) for full text.
