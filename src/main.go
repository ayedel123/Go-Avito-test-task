package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

func getTenders(db *sql.DB) ([]Tender, error) {

	rows, err := db.Query("SELECT id, name, description, status, service_type, author_id, version, created_at FROM tenders")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tenders []Tender
	for rows.Next() {
		var tender Tender
		if err := rows.Scan(&tender.ID, &tender.Name, &tender.Description, &tender.Status, &tender.ServiceType, &tender.AuthorID, &tender.Version, &tender.CreatedAt); err != nil {
			return nil, err
		}
		tenders = append(tenders, tender)

	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tenders, nil

}

func tendersHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		tenders, err := getTenders(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tenders)

	}

}

func pingHandler(w http.ResponseWriter, r *http.Request) {

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func newTenderHandler(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()

	fmt.Fprintf(w, "Method: %s\n", r.Method)
	fmt.Fprintf(w, "Headers: %v\n", r.Header)
	fmt.Fprintf(w, "Body: %s\n", body)

}

func main() {

	/*dbUser := os.Getenv("POSTGRES_USERNAME")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DATABASE")
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	*/
	connStr := os.Getenv("POSTGRES_CONN")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/api/ping", pingHandler)
	http.HandleFunc("/api/users", usersHandler(db))
	http.HandleFunc("/api/tenders", tendersHandler(db))
	http.HandleFunc("/api/tenders/new", newTenderHandler)
	log.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}
