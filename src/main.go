package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

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
	http.HandleFunc("/api/ping", pingHandler)
	http.HandleFunc("/api/users", usersHandler(db))
	http.HandleFunc("/api/tenders", tendersHandler(db))
	http.HandleFunc("/api/tenders/new", newTenderHandler(db))
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
