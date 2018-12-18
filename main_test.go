package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkSelect(b *testing.B) {
	server := NewAPIServer(&cfg)

	for i := 0; i < 10000; i++ {
		//創建一個請求
		req, err := http.NewRequest("GET", "/API/Select/ETH", nil)
		if err != nil {
			b.Fatal(err)
		}

		//創建一個ResponseRecorder
		r := httptest.NewRecorder()

		server.selectCurrency(r, req)

		if status := r.Code; status != http.StatusOK {
		}

		expected := `{"alive": true}`
		if r.Body.String() != expected {
		}
	}

	log.Printf("BenchmarkSelect")
}
