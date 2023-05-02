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

## Getting Rid of SQLite3

It would be possible to get rid of SQLite3 by directly implementing an ODBC driver which translates the two types of query currently issued by KiCad to API calls into Inventree or other apps. Finding resources and examples of implementing a simple ODBC driver is considerably harder than what I have accomplished here.

The following is a list of resources that might help should somebody actually want to do this:

* [Developing an ODBC Driver](https://learn.microsoft.com/en-us/sql/odbc/reference/develop-driver/developing-an-odbc-driver?view=sql-server-ver16)
* [SQLiteODBC source code](https://ch-werner.homepage.t-online.de/sqliteodbc/html/sqlite3odbc_8c-source.html)
* [A old Windows based ODBC driver](https://github.com/LukeMauldin/lodbc)
