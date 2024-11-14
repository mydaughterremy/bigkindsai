package db

// create mysql dsn string
import (
	"fmt"
)

func CreateMySQLDSN(host string, port string, user string, password string, dbname string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, dbname)
}
