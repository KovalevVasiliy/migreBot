package main

import (
    "fmt"
    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/postgres"
    "os"
)

var db *gorm.DB = nil

func initPostgres() {
    var err error
    PostgresPassword := os.Getenv("POSTGRES_PASSWORD")
    PostgresUser := os.Getenv("POSTGRES_USER")
    PostgresDb := os.Getenv("POSTGRES_DB")
    PostgresPort := os.Getenv("POSTGRES_PORT")
    PostgresHost := os.Getenv("POSTGRES_HOST")

    connectionString := fmt.Sprintf("host=%s port=%s password=%s user=%s dbname=%s sslmode=disable",
        PostgresHost, PostgresPort, PostgresPassword, PostgresUser, PostgresDb)
    db, err = gorm.Open("postgres", connectionString)
    if err != nil {
        panic(err)
    }
}

func getPostgres() *gorm.DB {
    return db
}