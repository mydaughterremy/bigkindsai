package main

import (
	"fmt"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func GetGORMDB() (*gorm.DB, error) {
	host := os.Getenv("UPSTAGE_DATABASE_HOST")
	port := os.Getenv("UPSTAGE_DATABASE_PORT")
	user := os.Getenv("UPSTAGE_DATABASE_USER")
	password := os.Getenv("UPSTAGE_DATABASE_PASSWORD")
	dbname := os.Getenv("UPSTAGE_DATABASE_NAME")
	sslmode := os.Getenv("UPSTAGE_DATABASE_SSLMODE")
	sslrootcert := os.Getenv("UPSTAGE_DATABASE_SSLROOTCERT")

	switch os.Getenv("UPSTAGE_DATABASE_ENGINE") {
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s sslrootcert=%s",
			host, port, user, password, dbname, sslmode, sslrootcert)
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Asia%%2FSeoul",
			user, password, host, port, dbname)
		return gorm.Open(mysql.Open(dsn), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unknown database engine")
	}
}
