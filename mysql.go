package uker

import (
	"fmt"

	mysqlDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Config interface
type MySQLConnData struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
}

// Global interface
type Mysql interface {
	// Satablish connection with MySQL server
	//
	// @param conn MySQLConnData: struct with necessary data to stablish connection with database.
	//
	// @param migrate ...interface{}: all interfaces to import to database.
	//
	// @return (db *gorm.DB, err error): database connection & error if exists
	StablishConnection(migrate ...interface{}) (db *gorm.DB, err error)
}

// Local struct to be implmented
type mysql_implementation struct {
	conn MySQLConnData
}

// External contructor
func NewMySQL(connData MySQLConnData) Mysql {
	return &mysql_implementation{
		conn: connData,
	}
}

func (sql *mysql_implementation) StablishConnection(migrate ...interface{}) (db *gorm.DB, err error) {
	// create string connection
	connString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", sql.conn.User, sql.conn.Password, sql.conn.Host, sql.conn.Port, sql.conn.Database)

	// open connection to mysql server
	db, err = gorm.Open(mysqlDriver.Open(connString), &gorm.Config{})
	if err != nil {
		return
	}

	// migrate tables
	db.AutoMigrate(&migrate)
	return
}
