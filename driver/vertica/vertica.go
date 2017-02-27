// Package vertica implements the Driver interface.
package vertica

import (
	"database/sql"
	"fmt"
	neturl "net/url" // alias to allow `url string` func signature in createDSN
	"strings"

	_ "github.com/alexbrainman/odbc"
	"github.com/mattes/migrate/driver"
	"github.com/mattes/migrate/file"
	"github.com/mattes/migrate/migrate/direction"
)

type Driver struct {
	db *sql.DB
}

var (
	dsn = "Driver=%s;Servername=%s;Port=%s;Database=%s;uid=%s;pwd=%s;"
)

const tableName = "schema_migrations"

func createDSN(url string) (string, error) {
	u, err := neturl.Parse(url)
	if err != nil {
		return "", err
	}
	// host and port
	hostElems := strings.Split(u.Host, ":")
	host := hostElems[0]
	port := ""
	if len(hostElems) > 1 {
		port = hostElems[1]
	}
	// database
	database := strings.TrimLeft(u.Path, "/")
	// username and password
	username := ""
	password := ""
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}

	return fmt.Sprintf(dsn, "Vertica", host, port, database, username, password), nil
}

func (driver *Driver) Initialize(url string) error {
	dsn, err := createDSN(url)
	if err != nil {
		return err
	}
	db, err := sql.Open("odbc", dsn)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}
	driver.db = db

	if err := driver.ensureVersionTableExists(); err != nil {
		return err
	}
	return nil
}

func (driver *Driver) Close() error {
	if err := driver.db.Close(); err != nil {
		return err
	}
	return nil
}

func (driver *Driver) ensureVersionTableExists() error {
	r := driver.db.QueryRow("SELECT count(*) FROM tables WHERE table_name = ?", tableName)
	c := 0
	if err := r.Scan(&c); err != nil {
		return err
	}
	if c > 0 {
		return nil
	}
	if _, err := driver.db.Exec("CREATE TABLE IF NOT EXISTS " + tableName + " (version bigint not null primary key)"); err != nil {
		return err
	}
	return nil
}

func (driver *Driver) FilenameExtension() string {
	return "sql"
}

func (driver *Driver) Migrate(f file.File, pipe chan interface{}) {
	defer close(pipe)
	pipe <- f

	tx, err := driver.db.Begin()
	if err != nil {
		pipe <- err
		return
	}

	if f.Direction == direction.Up {
		if _, err := tx.Exec("INSERT INTO "+tableName+" (version) VALUES (?)", f.Version); err != nil {
			pipe <- err
			if err := tx.Rollback(); err != nil {
				pipe <- err
			}
			return
		}
	} else if f.Direction == direction.Down {
		if _, err := tx.Exec("DELETE FROM "+tableName+" WHERE version=?", f.Version); err != nil {
			pipe <- err
			if err := tx.Rollback(); err != nil {
				pipe <- err
			}
			return
		}
	}

	if err := f.ReadContent(); err != nil {
		pipe <- err
		return
	}

	// ODBC makes a prepared statement so one can only do one command per Exec
	commands := strings.Split(string(f.Content), ";")
	for _, command := range commands {
		command = strings.Trim(command, "\n")
		if command == "" || strings.HasPrefix(command, "--") {
			continue
		}
		if _, err := tx.Exec(command); err != nil {
			pipe <- err
			if err := tx.Rollback(); err != nil {
				pipe <- err
			}
			return
		}
	}

	if err := tx.Commit(); err != nil {
		pipe <- err
		return
	}
}

func (driver *Driver) Version() (uint64, error) {
	var version uint64
	err := driver.db.QueryRow("SELECT version FROM " + tableName + " ORDER BY version DESC LIMIT 1").Scan(&version)
	switch {
	case err == sql.ErrNoRows:
		return 0, nil
	case err != nil:
		return 0, err
	default:
		return version, nil
	}
}

func init() {
	driver.RegisterDriver("vertica", &Driver{})
}
