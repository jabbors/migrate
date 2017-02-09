# Vertica Driver

* Runs migrations in transactions.
  That means that if a migration fails, it will be safely rolled back.
* Tries to return helpful error messages.
* Stores migration version details in table ``schema_migrations``.
  This table will be auto-generated.


## Usage

```bash
migrate -url vertica://user@host:port/database -path ./db/migrations create add_field_to_table
migrate -url vertica://user@host:port/database -path ./db/migrations up
migrate help # for more info
```

## Note

This driver is based on https://github.com/alexbrainman/odbc/ and requires, besides unixODBC that
Vertica drivers are installed and configured on the client system. For more information see
https://github.com/alexbrainman/odbc/wiki and https://my.vertica.com/docs/8.0.x/HTML/index.htm#Authoring/ConnectingToHPVertica/ClientDriverMisc/InstallingTheHPVerticaClientDrivers.htm%3FTocPath%3DConnecting%2520to%2520Vertica%7CClient%2520Libraries%7CClient%2520Drivers%7C_____0

## Authors

* Johan Abbors, https://github.com/jabbors
