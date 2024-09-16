package main

import (
	"database/sql"
	"encoding/json"
	"go_server/m/bids"
	"go_server/m/common/dbhelp"
	_ "go_server/m/common/errinfo"
	"go_server/m/tenders"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	_ "github.com/lib/pq"
)

func getUsers(db *sql.DB) ([]dbhelp.Employee, error) {
	rows, err := db.Query("SELECT id, username, first_name, last_name, created_at, updated_at FROM employee")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []dbhelp.Employee
	for rows.Next() {
		var user dbhelp.Employee
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
	r.HandleFunc("/api/tenders", tenders.TendersHandler(db)).Methods("GET")
	r.HandleFunc("/api/archived_tenders", tenders.TendersArchiveHandler(db)).Methods("GET")

	r.HandleFunc("/api/tenders/new", tenders.NewTenderHandler(db)).Methods("POST")
	r.HandleFunc("/api/tenders/my", tenders.MyTendersHandler(db)).Methods("GET")

	r.HandleFunc("/api/tenders/{tenderId}/status", tenders.StatusTendersHandler(db)).Methods("GET", "PUT")
	r.HandleFunc("/api/tenders/{tenderId}/edit", tenders.EditTendersHandler(db)).Methods("PATCH")
	r.HandleFunc("/api/tenders/{tenderId}/rollback/{version}", tenders.RollbackTendersHandler(db)).Methods("PUT")

	r.HandleFunc("/api/bids", bids.BidsHandler(db)).Methods("GET")
	r.HandleFunc("/api/bids/new", bids.NewBidHandler(db)).Methods("POST")
	r.HandleFunc("/api/bids/my", bids.MyBidsHandler(db)).Methods("GET")

	r.HandleFunc("/api/bids/{tenderId}/list", bids.ListBidsHandler(db)).Methods("GET")
	r.HandleFunc("/api/bids/{bidId}/status", bids.StatusBidsHandler(db)).Methods("GET", "PUT")
	r.HandleFunc("/api/bids/{bidId}/edit", bids.EditBidsHandler(db)).Methods("PATCH")
	r.HandleFunc("/api/bids/{bidId}/rollback/{version}", bids.RollbackBidsHandler(db)).Methods("PUT")
	r.HandleFunc("/api/bids/{bidId}/feedback", bids.FeedbackHandler(db)).Methods("PUT")
	r.HandleFunc("/api/bids/{tenderId}/reviews", bids.ReviewsHandler(db)).Methods("GET")

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
