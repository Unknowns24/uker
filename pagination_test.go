package uker_test

import (
	"encoding/json"
	"testing"

	"github.com/unknowns24/uker"
)

func TestPaginate(t *testing.T) {
	type TestProduct struct {
		Id    uint   `json:"id" gorm:"primary_key"`
		State uint   `json:"state"`
		Name  string `json:"name" gorm:"unique"`
		Desc  string `json:"description"`
	}

	mock := uker.MockDB{
		UsedInterfaces: []interface{}{
			&TestProduct{},
		},
		Objects: []interface{}{
			&TestProduct{Name: "tp1", State: 1, Desc: "ssa"},
			&TestProduct{Name: "tp2", State: 0},
			&TestProduct{Name: "tp3ss", State: 1},
			&TestProduct{Name: "ss", State: 2},
		},
	}

	db, err := mock.Create()
	if err != nil {
		t.Fatal(err)
	}

	var result []TestProduct

	noPaginateParams := uker.Pagination{}
	paginateParams := uker.Pagination{
		Page:    "1",
		Search:  "ss",
		PerPage: "1",
		Sort:    "id",
		SortDir: uker.PAGINATION_ORDER_DESC,
	}

	pagOneParams := uker.PaginationOpts{
		DB:         db,
		Where:      "state != 0",
		Result:     &result,
		TableModel: &TestProduct{},
	}

	// Call the Paginate function
	paginationResult := noPaginateParams.Paginate(pagOneParams)

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
	pagTwoParams := uker.PaginationOpts{
		DB:         db,
		Where:      "state != 2",
		Result:     &result2,
		TableModel: &TestProduct{},
	}
	paginationResult2 := paginateParams.Paginate(pagTwoParams)

	if paginationResult2["last_page"] != 2 {
		t.Error("Per page not working")
	}

	if paginationResult2["total"] != int64(2) {
		t.Errorf("Some clause is not working, expected total 2 -> %d received", paginationResult2["total"])
	}

	err = mock.Close()
	if err != nil {
		t.Error(err)
	}
}
