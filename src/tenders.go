package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

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

type CreateTenderData struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	ServiceType     string `json:"serviceType"`
	Status          string `json:"status"`
	OrganizationID  int    `json:"organizationId"`
	CreatorUsername string `json:"creatorUsername"`
}

func atoi(s string) (int, int) {

	if num, err := strconv.Atoi(s); err == nil && num >= 0 {
		return num, 200
	}
	return 0, 400
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

func getLimitOffsetFromRequest(r *http.Request) (int, int, int) {
	s_limit := r.URL.Query().Get("limit")
	s_offset := r.URL.Query().Get("offset")

	var limit, offset int

	if s_limit == "" {
		s_limit = "5"
	}

	if s_offset == "" {
		s_offset = "0"
	}
	status := 200
	limit, status = atoi(s_limit)
	if status == 200 {
		offset, status = atoi(s_offset)
	}
	return limit, offset, status

}

func getIntFromRequest(r *http.Request, default_val int, param_name string) (int, int) {
	s_param := r.URL.Query().Get(param_name)
	if default_val != -1 && s_param == "" {
		s_param = strconv.Itoa(default_val)
	}
	param, status := atoi(s_param)
	return param, status

}

func getUserNameFromRequest(r *http.Request) string {
	username := r.URL.Query().Get("username")
	return username
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

func getUserId(db *sql.DB, user_name string) (int, int) {
	var id int

	query := `
        SELECT e.id 
        FROM  employee e WHERE e.username = $1
		LIMIT 1
    `
	err := db.QueryRow(query, user_name).Scan(&id)
	log.Println("name ", user_name, " err", err)
	return id, sqlErrToStatus(err, 401)
}

func getTenders(db *sql.DB, limit, offset int) ([]Tender, error) {
	query := `
	SELECT id, name, description, status, service_type, author_id, version, created_at 
	FROM tenders
	ORDER BY name
	LIMIT $1
	OFFSET $2
	`
	rows, err := db.Query(query, limit, offset)
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
		if r.Method != http.MethodGet {
			sendHttpErr(w, http.StatusMethodNotAllowed)
			return
		}
		limit, offset, res_status := getLimitOffsetFromRequest(r)
		if res_status != 200 {
			return
		}
		tenders, err := getTenders(db, limit, offset)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tenders)
	}
}

func createTender(db *sql.DB, req CreateTenderData) (*Tender, int) {

	user_id, status := getUserId(db, req.CreatorUsername)

	if status != 200 {
		return new(Tender), status
	}

	if status = isUserInOrganization(db, user_id, req.OrganizationID); status != 200 {
		return new(Tender), status
	}

	created_at := time.Now()
	version := 1

	query := `
		INSERT INTO tenders (name, description, status, service_type, author_id, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id int
	err := db.QueryRow(query, req.Name, req.Description, req.Status, req.ServiceType, user_id, version, created_at).Scan(&id)
	status = sqlErrToStatus(err, http.StatusInternalServerError)

	if status != http.StatusOK {
		return nil, status
	}
	tender := createTenderDataToTender(req, id, user_id, version, created_at)

	return tender, status

}

func newTenderHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, ErrMessageWrongRequest, http.StatusMethodNotAllowed)
			return
		}

		var req CreateTenderData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, ErrMessageWrongRequest, http.StatusBadRequest)
			return
		}

		tender, res_status := createTender(db, req)
		log.Println("Status", res_status)
		if res_status != 200 {
			sendHttpErr(w, res_status)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tender)

	}

}

func getUserTenders(db *sql.DB, user_id, limit, offset int) ([]Tender, int) {
	query := `
	SELECT id, name, description, status, service_type, author_id, version, created_at
	FROM tenders
	WHERE author_id = $1
	ORDER BY name
	LIMIT $2 OFFSET $3
	`
	rows, err := db.Query(query, user_id, limit, offset)
	res_status := sqlErrToStatus(err, 200)
	if res_status != 200 {
		return nil, res_status
	}
	defer rows.Close()
	var tenders []Tender
	for rows.Next() {
		var tender Tender
		if err := rows.Scan(&tender.ID, &tender.Name, &tender.Description, &tender.Status, &tender.ServiceType, &tender.AuthorID, &tender.Version, &tender.CreatedAt); err != nil {
			return nil, sqlErrToStatus(err, 500)
		}
		tenders = append(tenders, tender)
	}

	return tenders, sqlErrToStatus(rows.Err(), 500)

}

func myTendersHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			sendHttpErr(w, http.StatusMethodNotAllowed)
			return
		}
		limit, offset, res_status := getLimitOffsetFromRequest(r)
		var tenders []Tender
		var user_id int
		if res_status == 200 {
			user_name := getUserNameFromRequest(r)
			user_id, res_status = getUserId(db, user_name)

		}
		if res_status == 200 {
			tenders, res_status = getUserTenders(db, user_id, limit, offset)
		}

		if res_status != 200 {
			sendHttpErr(w, res_status)

		} else {

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tenders)
		}
	}
}

func getTenderStatus(tender_id, user_id int) {

}

func handleGetTenderStatus(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	tender_i, res_status := getIntFromRequest(r, -1, "tenderId")
	if res_status != 200 {
		sendHttpErr(w, res_status)
		return
	}
	user_name := getUserNameFromRequest(r)
	user_id, res_status := getUserId(db, user_name)
	if res_status != 200 {
		sendHttpErr(w, res_status)
		return
	}

	getTenderStatus(tender_i, user_id)

}

func statusTendersHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			sendHttpErr(w, http.StatusMethodNotAllowed)
			return
		} else if r.Method == http.MethodGet {
			handleGetTenderStatus(db, w, r)

		} else if r.Method == http.MethodPut {

		}

	}
}
