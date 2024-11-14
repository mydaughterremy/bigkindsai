package db

// create postgres dsn string
import (
	"fmt"
)

func CreatePostgresDSN(host string, port string, user string, password string, dbname string, certPath string) string {
	if certPath == "" {
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	} else {
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=verify-full sslrootcert=%s", host, port, user, password, dbname, certPath)
	}
}
