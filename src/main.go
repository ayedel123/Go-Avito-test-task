package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	_ "github.com/lib/pq"
)

func getUsers(db *sql.DB) ([]Employee, error) {
	rows, err := db.Query("SELECT id, username, first_name, last_name, created_at, updated_at FROM employee")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []Employee
	for rows.Next() {
		var user Employee
		if err := rows.Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func usersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := getUsers(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func httpSetHandlers(db *sql.DB) {
	r := mux.NewRouter()

	r.HandleFunc("/api/ping", pingHandler).Methods("GET")
	r.HandleFunc("/api/users", usersHandler(db)).Methods("GET")
	r.HandleFunc("/api/tenders", tendersHandler(db)).Methods("GET")
	r.HandleFunc("/api/archived_tenders", tendersArchiveHandler(db)).Methods("GET")

	r.HandleFunc("/api/tenders/new", newTenderHandler(db)).Methods("POST")
	r.HandleFunc("/api/tenders/my", myTendersHandler(db)).Methods("GET")

	r.HandleFunc("/api/tenders/{tenderId}/status", statusTendersHandler(db)).Methods("GET", "PUT")
	r.HandleFunc("/api/tenders/{tenderId}/edit", editTendersHandler(db)).Methods("PATCH")
	r.HandleFunc("/api/tenders/{tenderId}/rollback/{version}", rollbackTendersHandler(db)).Methods("PUT")

	r.HandleFunc("/api/bids", bidsHandler(db)).Methods("GET")
	r.HandleFunc("/api/bids/new", newBidHandler(db)).Methods("POST")
	r.HandleFunc("/api/bids/my", myBidsHandler(db)).Methods("GET")

	r.HandleFunc("/api/bids/{tenderId}/list", listBidsHandler(db)).Methods("GET")
	r.HandleFunc("/api/bids/{bidId}/status", statusBidsHandler(db)).Methods("GET", "PUT")
	r.HandleFunc("/api/bids/{bidId}/edit", editBidsHandler(db)).Methods("PATCH")
	r.HandleFunc("/api/bids/{bidId}/rollback/{version}", editBidsHandler(db)).Methods("PUT")

	http.Handle("/", r)
}

func main() {
	connStr := os.Getenv("POSTGRES_CONN")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	httpSetHandlers(db)
	log.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
