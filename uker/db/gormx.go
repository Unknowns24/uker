package db

import (
	"fmt"

	mysqlDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// MySQLConnData holds the connection data used to connect with MySQL.
type MySQLConnData struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
}

// Connector opens a MySQL connection using the provided configuration.
type Connector struct {
	conn MySQLConnData
}

// NewMySQL creates a new Connector instance.
func NewMySQL(conn MySQLConnData) Connector {
	return Connector{conn: conn}
}

// Open establishes a connection to MySQL and optionally runs migrations.
func (c Connector) Open(migrate ...any) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", c.conn.User, c.conn.Password, c.conn.Host, c.conn.Port, c.conn.Database)
	db, err := gorm.Open(mysqlDriver.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if len(migrate) > 0 {
		if err := db.AutoMigrate(migrate...); err != nil {
			return nil, err
		}
	}

	return db, nil
}
