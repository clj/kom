# Development notes

## ODBC Logging

ODBC logging can be turned on by creating ~/.odbcinst.ini, containing the following section:

    [ODBC]
    Trace        = Yes
    TraceFile    = /tmp/sql.log
    ForceTrace   = Yes

## Resources

### KiCad Database Libraries

#### Examples

Examples of how to set up KiCad database libraries:

* https://forum.kicad.info/t/a-database-case-study/39631/7
* https://github.com/SumantKhalate/KiCad-libdb/

#### Internals

* https://gitlab.com/kicad/code/kicad/-/blob/master/common/database/database_connection.cpp
    * shows the (two) queries that are used:
        * select all: "SELECT * FROM {}{}{}"
        * select one: "SELECT * FROM {}{}{} WHERE {}{}{} = ?"

### SQLite3

#### BestIndex and Filter

Figuring out exactly how BestIndex and Filter interact with each other was slightly tricky. Here are some useful examples:

* https://www.sqlite.org/src/file?name=ext/misc/series.c&ci=trunk
    * encodes the constraints as bits in idxNum
* https://www.sqlite.org/src/file?name=ext/rtree/rtree.c&ci=trunk
    * more complex constraints than above
* https://www.sqlite.org/src/file?name=ext/fts5/fts5_main.c&ci=trunk
    * a well documented example

#### Non-C Language Implementations

* Go
    * https://github.com/riyaz-ali/sqlite
        * Only focused on building extensions
    * https://github.com/mattn/go-sqlite3
        * SQLite3 driver that also has support for building extensions