package uker_test

import (
	"encoding/json"
	"testing"

	"github.com/unknowns24/uker"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPaginate(t *testing.T) {
	type TestProduct struct {
		Id    uint   `json:"id" gorm:"primary_key"`
		State uint   `json:"state"`
		Name  string `json:"name" gorm:"unique"`
		Desc  string `json:"description"`
	}

	// Create a GORM DB instance with a SQLite driver
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		t.Fatalf("Error creating GORM DB: %v", err)
	}

	// Import table
	db.AutoMigrate(&TestProduct{})

	// create test products
	tp1 := TestProduct{Name: "tp1", State: 1, Desc: "ssa"}
	tp2 := TestProduct{Name: "tp2", State: 0}
	tp3 := TestProduct{Name: "tp3ss", State: 1}
	tp4 := TestProduct{Name: "ss", State: 2}

	db.Create(&tp1)
	db.Create(&tp2)
	db.Create(&tp3)
	db.Create(&tp4)

	var result []TestProduct

	noPaginateParams := uker.Pagination{}
	paginateParams := uker.Pagination{
		Page:    "1",
		Search:  "ss",
		PerPage: "1",
		Sort:    "id",
		SortDir: uker.PAGINATION_ORDER_DESC,
	}

	// Call the Paginate function
	paginationResult := noPaginateParams.Paginate(db, "test_products", "state != 0", &result)

	// Check if the pagination result is not nil
	if paginationResult == nil {
		t.Errorf("Pagination result is nil")
	}

	// Check if the pagination result has the correct keys
	expectedKeys := []string{"page", "total", "per_page", "last_page", "data"}
	for _, key := range expectedKeys {
		if _, ok := paginationResult[key]; !ok {
			t.Errorf("Pagination result does not have key: %s", key)
		}
	}

	// Get all products inside data
	jsonData, _ := json.Marshal(paginationResult["data"])

	var resProducts []TestProduct
	json.Unmarshal(jsonData, &resProducts)

	for _, product := range resProducts {
		if product.State == 0 {
			t.Error("Pagination result where codition does not work, unexpected product state returned")
		}
	}

	// test if params are working
	var result2 []TestProduct

	// Call the Paginate function
	paginationResult2 := paginateParams.Paginate(db, "test_products", "state != 2", &result2)

	if paginationResult2["last_page"] != 2 {
		t.Error("Per page not working")
	}

	if paginationResult2["total"] != int64(2) {
		t.Errorf("Some clause is not working, expected total 2 -> %s received", paginationResult2["total"])
	}
}
