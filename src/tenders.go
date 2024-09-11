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

type editTenderRequestBody struct {
	Name        string `json:"name,omitempty"`        // Поле не обязательно
	Description string `json:"description,omitempty"` // Поле не обязательно
	ServiceType string `json:"serviceType,omitempty"` // Поле не обязательно
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

func getTenders(db *sql.DB, limit, offset int, service_type string) ([]Tender, error) {
	var query string
	var args []interface{}
	if service_type != "" {
		query = `
		SELECT id, name, description, status, service_type, author_id, version, created_at 
		FROM tenders
		WHERE service_type = $1
		ORDER BY name
		LIMIT $2
		OFFSET $3
		`
		args = []interface{}{service_type, limit, offset}
	} else {
		query = `
		SELECT id, name, description, status, service_type, author_id, version, created_at 
		FROM tenders
		ORDER BY name
		LIMIT $1
		OFFSET $2
		`
		args = []interface{}{limit, offset}
	}

	rows, err := db.Query(query, args...)
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

func isOkServiceType(service_type string) bool {
	return (service_type == "" || (service_type == "Construction" || service_type == "Delivery" || service_type == "Manufacture"))
}

func tendersHandler(db *sql.DB) http.HandlerFunc {
	log.Println("Restarted")
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendHttpErr(w, http.StatusMethodNotAllowed)
			return
		}
		service_type := r.URL.Query().Get("service_type")
		limit, offset, res_status := getLimitOffsetFromRequest(r)
		if res_status != 200 || !isOkServiceType(service_type) {
			sendHttpErr(w, 400)
			return
		}

		tenders, err := getTenders(db, limit, offset, service_type)
		if err != nil {
			sendHttpErr(w, 500)
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
		INSERT INTO tenders (name, description, status, service_type, author_id,organization_id, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7,$8)
		RETURNING id`

	var id int
	err := db.QueryRow(query, req.Name, req.Description, req.Status, req.ServiceType, user_id, req.OrganizationID, version, created_at).Scan(&id)
	status = sqlErrToStatus(err, http.StatusInternalServerError)

	if status != http.StatusOK {
		return nil, status
	}
	tender := createTenderDataToTender(req, id, user_id, version, created_at)

	return tender, status

}

func validateNewTender(new_tender *CreateTenderData) bool {
	if len(new_tender.Name) > 100 {
		return false
	}
	if len(new_tender.Description) > 100 {
		return false
	}
	return true
}

func newTenderHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, ErrMessageWrongRequest, http.StatusMethodNotAllowed)
			return
		}

		var req CreateTenderData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validateNewTender(&req) {
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

func getTender(db *sql.DB, tender_id int) (*Tender, int) {
	query := `
    SELECT t.unique_id, t.id, t.name, t.description, t.status, t.service_type, t.author_id, t.organization_id, t.version, t.created_at
    FROM tenders t
    JOIN (
        SELECT id, MAX(version) AS lv
        FROM tenders
        WHERE id = $1
        GROUP BY id
    ) as lt ON t.id = lt.id AND t.version = lt.lv;
    `

	rows, err := db.Query(query, tender_id)
	if err != nil {
		return nil, sqlErrToStatus(err, 500)
	}
	defer rows.Close()

	var tender Tender
	if rows.Next() {
		err := rows.Scan(&tender.UniqueID, &tender.ID, &tender.Name, &tender.Description, &tender.Status,
			&tender.ServiceType, &tender.AuthorID, &tender.OrganizationID, &tender.Version, &tender.CreatedAt)
		if err != nil {
			return nil, sqlErrToStatus(err, 404)
		}
		return &tender, 200
	}

	return nil, 404
}

func updateTenderStatus(db *sql.DB, tender_uid int, status string) (string, int) {
	query := `
		UPDATE tenders
		SET status = $1
		WHERE unique_id = $2
		RETURNING status
	`

	var updatedStatus string
	err := db.QueryRow(query, status, tender_uid).Scan(&updatedStatus)
	if err != nil {
		return "", sqlErrToStatus(err, 404)
	}

	return updatedStatus, 200
}

func handleGetTenderStatus(db *sql.DB, w http.ResponseWriter, r *http.Request, tender_id int) {

	user_name := getUserNameFromRequest(r)
	if user_name != "" {
		_, res_status := getUserId(db, user_name)
		if res_status != 200 {
			sendHttpErr(w, res_status)
			return
		}
	}
	tender, res_status := getTender(db, tender_id)
	if res_status != 200 {
		sendHttpErr(w, res_status)
		return
	}
	w.Write([]byte(tender.Status))
}

func isNewStatusOk(status string) bool {
	return (status != "" && (status == "Created" || status == "Published" || status == "Closed"))
}

func handlePutTenderStatus(db *sql.DB, w http.ResponseWriter, r *http.Request, tender_id int) {

	user_name := r.URL.Query().Get("username")
	new_status := r.URL.Query().Get("status")
	if user_name == "" || !isNewStatusOk(new_status) {
		sendHttpErr(w, 400)
		return
	}
	user_id, res_status := getUserId(db, user_name)
	if res_status != 200 {
		sendHttpErr(w, res_status)
		return
	}

	tender, res_status := getTender(db, tender_id)
	if res_status != 200 {
		sendHttpErr(w, res_status)
		return
	}
	res_status = isUserInOrganization(db, user_id, tender.OrganizationID)
	if res_status != 200 {
		sendHttpErr(w, res_status)
		return
	}

	new_status, res_status = updateTenderStatus(db, tender.UniqueID, new_status)
	if res_status != 200 {
		sendHttpErr(w, res_status)
		return
	}
	tender.Status = new_status
	json.NewEncoder(w).Encode(tender)
}

func statusTendersHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		s_tender_id := r.URL.Path[len("/api/tenders/") : len(r.URL.Path)-len("/status")]
		tender_id, _ := atoi(s_tender_id)
		log.Println("status handling ", s_tender_id)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			handleGetTenderStatus(db, w, r, tender_id)

		} else if r.Method == http.MethodPut {
			handlePutTenderStatus(db, w, r, tender_id)
		} else {
			sendHttpErr(w, http.StatusMethodNotAllowed)
			return
		}

	}
}

func editTender() {

}

func editTendersHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		s_tender_id := r.URL.Path[len("/api/tenders/") : len(r.URL.Path)-len("/edit")]
		user_name := r.URL.Query().Get("username")

		var reqBody editTenderRequestBody
		tender_id, res_status := atoi(s_tender_id)
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil || res_status != 200 {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			log.Println("Error", s_tender_id)
			return
		}

		tender, res_status := getTender(db, tender_id)
		if res_status != 200 {
			http.Error(w, "Wrong tender id", http.StatusBadRequest)
			return
		}

		user_id, res_status := getUserId(db, user_name)
		if res_status != 200 {
			http.Error(w, "Wrong user", res_status)
			return
		}
		res_status = isUserInOrganization(db, user_id, tender.OrganizationID)

		if res_status != 200 {
			http.Error(w, ErrMessageWrongUser, res_status)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("Editing " + s_tender_id))
		w.Write([]byte("User " + user_name + " Userid " + strconv.Itoa(user_id) + "\n"))
		json.NewEncoder(w).Encode(reqBody)
	}
}
