package uker

import (
	"fmt"

	mysqlDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Config interface
type DBConnection struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
}

// Global interface
type MySQL interface {
	StablishConnection(conn DBConnection, migrate ...interface{}) (db *gorm.DB, err error)
}

// Local struct to be implmented
type mysql struct{}

// External contructor
func NewMySQL() MySQL {
	return &mysql{}
}

// Stablish connection function implementation
func (sql *mysql) StablishConnection(conn DBConnection, migrate ...interface{}) (db *gorm.DB, err error) {
	// create string connection
	connString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", conn.User, conn.Password, conn.Host, conn.Port, conn.Database)

	// open connection to mysql server
	db, err = gorm.Open(mysqlDriver.Open(connString), &gorm.Config{})
	if err != nil {
		return
	}

	// migrate tables
	db.AutoMigrate(&migrate)
	return
}
