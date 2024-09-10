package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

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

type CreateTenderData struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	ServiceType     string `json:"serviceType"`
	Status          string `json:"status"`
	OrganizationID  int    `json:"organizationId"`
	CreatorUsername string `json:"creatorUsername"`
}

func sqlErrToStatus(err error, err_status int) int {
	status := 200
	if err != nil {
		if err == sql.ErrNoRows {
			status = err_status
		} else {
			status = 500
		}

	}
	return status
}

func isUserInOrganization(db *sql.DB, user_id, organization_id int) int {

	query := `
        SELECT orgr.user_id 
        FROM organization_responsible orgr 
        WHERE orgr.user_id = $1 AND orgr.organization_id = $2
		LIMIT 1
    `
	err := db.QueryRow(query, user_id, organization_id).Scan(&user_id)

	return sqlErrToStatus(err, 403)
}

func isUserExists(db *sql.DB, req CreateTenderData) (int, int) {
	var id int

	query := `
        SELECT e.id 
        FROM  employee e WHERE e.username = $1
		LIMIT 1
    `
	err := db.QueryRow(query, req.CreatorUsername).Scan(&id)
	log.Println("naem ", req.CreatorUsername, " err", err)
	return id, sqlErrToStatus(err, 401)
}

func createTenderDataToTender(req CreateTenderData, id, user_id, version int, created_at time.Time) *Tender {
	return &Tender{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		ServiceType: req.ServiceType,
		AuthorID:    user_id,
		Version:     version,
		CreatedAt:   created_at,
	}
}

func createTender(db *sql.DB, req CreateTenderData) (*Tender, int) {

	user_id, status := isUserExists(db, req)

	if status != 200 {
		return new(Tender), status
	}

	if status = isUserInOrganization(db, user_id, req.OrganizationID); status != 200 {
		return new(Tender), status
	}

	id := 0
	created_at := time.Now()

	version := 1
	query := `
		INSERT INTO tenders (id, name, description, status, service_type,  author_id, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7,$8)`

	_, err := db.Exec(query, id, req.Name, req.Description, req.Status, req.ServiceType, user_id, version, created_at)
	status = sqlErrToStatus(err, 500)
	tender := createTenderDataToTender(req, id, user_id, version, created_at)

	return tender, status

}

func newTenderHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateTenderData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tender, status := createTender(db, req)
		log.Println("Status", status)
		if status != 200 {
			switch status {
			case http.StatusUnauthorized:
				http.Error(w, "Incorrect username or user does not exist.", status)
			case http.StatusForbidden:
				http.Error(w, "User does not have permission for this organization.", status)
			default:
				http.Error(w, "Something went wrong. Please try again.", status)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tender)

	}

}
