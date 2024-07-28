package uker

import (
	"fmt"
	"os"
	"reflect"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var mockdbfile string = "test.db"

type MockDB struct {
	database       *gorm.DB
	Objects        []interface{}
	UsedInterfaces []interface{}
}

// MockDB for testing
//
// @return *gorm.DB: Gorm db instance
func (m *MockDB) Create() (*gorm.DB, error) {
	// Create a GORM DB instance with a SQLite driver
	db, err := gorm.Open(sqlite.Open(mockdbfile), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	// Validate if gorm instance does not fail
	if err != nil {
		return nil, fmt.Errorf("error creating GORM DB: %v", err)
	}

	m.database = db

	// Import used interfaces
	db.AutoMigrate(m.UsedInterfaces...)

	for _, inter := range m.UsedInterfaces {
		reflect.TypeOf(inter).Elem()

		// Drop interface content
		db.Delete(inter, "1 = 1")
	}

	// Iterate objects and crete a record on db
	for _, obj := range m.Objects {
		reflect.TypeOf(obj).Elem()

		// Create record
		db.Create(obj)
	}

	return db, nil
}

func (m *MockDB) Close() error {
	sqlDB, err := m.database.DB()
	if err != nil {
		return err
	}

	sqlDB.Close()

	err = os.Remove(mockdbfile)
	if err != nil {
		return err
	}

	return nil
}
