package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func createTenderDataToTender(req CreateTenderData, user_id, version int, created_at time.Time) *Tender {
	return &Tender{
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
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	ServiceType string `json:"serviceType,omitempty"`
}

func atoi(s string) (num int, err_info ErrorInfo) {
	num = 0
	num, err := strconv.Atoi(s)
	if err == nil && num >= 0 {
		err_info.status = 200
		return
	}
	err_info.status = 400
	err_info.reason = "Parametr must be positive number."
	return
}

func sqlErrToErrInfo(err error, err_status int, message string) ErrorInfo {
	var err_info ErrorInfo
	if message != "" {
		err_info.reason = message
	}
	err_info.status = sqlErrToStatus(err, err_status)
	return err_info
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

func getLimitOffsetFromRequest(r *http.Request) (limit int, offset int, err_info ErrorInfo) {
	s_limit := r.URL.Query().Get("limit")
	s_offset := r.URL.Query().Get("offset")

	//var limit, offset int

	if s_limit == "" {
		s_limit = "5"
	}

	if s_offset == "" {
		s_offset = "0"
	}
	err_info.status = 200
	limit, err_info = atoi(s_limit)
	if err_info.status == 200 {
		offset, err_info = atoi(s_offset)
	}
	return

}

func getIntFromRequest(r *http.Request, default_val int, param_name string) (num int, err_info ErrorInfo) {
	s_param := r.URL.Query().Get(param_name)
	if default_val != -1 && s_param == "" {
		s_param = strconv.Itoa(default_val)
	}
	num, err_info = atoi(s_param)

	return
}

func getUserNameFromRequest(r *http.Request) string {
	username := r.URL.Query().Get("username")
	return username
}

func isUserInOrganization(db *sql.DB, user_id, organization_id int) ErrorInfo {

	query := `
        SELECT orgr.user_id 
        FROM organization_responsible orgr 
        WHERE orgr.user_id = $1 AND orgr.organization_id = $2
		LIMIT 1
    `
	err := db.QueryRow(query, user_id, organization_id).Scan(&user_id)
	var err_info ErrorInfo
	err_info.status = sqlErrToStatus(err, 403)
	err_info.reason = "User does not have permission."
	return err_info
}

func getUserId(db *sql.DB, user_name string) (user_id int, err_info ErrorInfo) {
	query := `
        SELECT e.id 
        FROM  employee e WHERE e.username = $1
		LIMIT 1
    `
	err := db.QueryRow(query, user_name).Scan(&user_id)

	err_info.status = sqlErrToStatus(err, 401)
	if err_info.status != 200 {
		err_info.reason = "User does not exist."
	}

	return
}

func getTenders(db *sql.DB, limit, offset int, service_type string) ([]Tender, ErrorInfo) {
	var query string
	var args []interface{}
	var error_info ErrorInfo
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
	error_info.status = sqlErrToStatus(err, 500)
	if error_info.status != 200 {
		error_info.reason = ErrMessageServer
		return nil, error_info
	}
	defer rows.Close()
	var tenders []Tender
	for rows.Next() {
		var tender Tender
		if err := rows.Scan(&tender.ID, &tender.Name, &tender.Description, &tender.Status, &tender.ServiceType, &tender.AuthorID, &tender.Version, &tender.CreatedAt); err != nil {
			error_info.reason = ErrMessageServer
			error_info.status = sqlErrToStatus(err, 500)
			return nil, error_info
		}
		tenders = append(tenders, tender)
	}
	if err := rows.Err(); err != nil {
		error_info.reason = ErrMessageServer
		error_info.status = sqlErrToStatus(err, 500)
		return nil, error_info
	}

	return tenders, error_info
}

func getArchivedTenders(db *sql.DB, limit, offset int, service_type string) ([]Tender, error) {
	var query string
	var args []interface{}
	if service_type != "" {
		query = `
		SELECT id, name, description, status, service_type, version 
		FROM tenders_archive
		WHERE service_type = $1
		ORDER BY name
		LIMIT $2
		OFFSET $3
		`
		args = []interface{}{service_type, limit, offset}
	} else {
		query = `
		SELECT id, name, description, status, service_type, version 
		FROM tenders_archive
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
		if err := rows.Scan(&tender.ID, &tender.Name, &tender.Description, &tender.Status, &tender.ServiceType, &tender.Version); err != nil {
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

func tendersArchiveHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info ErrorInfo
		if r.Method != http.MethodGet {
			err_info.reason = ErrMessageMethodNotAllowed
			err_info.status = 405
			sendHttpErr(w, err_info)
			return
		}
		service_type := r.URL.Query().Get("service_type")
		limit, offset, err_info := getLimitOffsetFromRequest(r)
		if err_info.status != 200 || !isOkServiceType(service_type) {
			sendHttpErr(w, err_info)
			return
		}

		tenders, err := getArchivedTenders(db, limit, offset, service_type)
		if err != nil {
			sendHttpErr(w, err_info)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tenders)
	}
}

func tendersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err_info ErrorInfo
		service_type := r.URL.Query().Get("service_type")
		limit, offset, err_info := getLimitOffsetFromRequest(r)
		if err_info.status != 200 || !isOkServiceType(service_type) {
			sendHttpErr(w, err_info)
			return
		}

		tenders, err_info := getTenders(db, limit, offset, service_type)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tenders)
	}
}

func isUserExistAndResponsible(db *sql.DB, user_name string, organization_id int) (user_id int, err_info ErrorInfo) {
	user_id, err_info = getUserId(db, user_name)
	if err_info.status != 200 {
		return
	}
	err_info = isUserInOrganization(db, user_id, organization_id)
	if err_info.status != 200 {
		err_info.status = http.StatusForbidden
		err_info.reason = ErrMessageNoPermission
		return
	}
	return
}

func createTender(db *sql.DB, creator_username string, tender *Tender) ErrorInfo {
	var err_info ErrorInfo
	query := `
		INSERT INTO tenders (name, description, status, service_type, author_id,organization_id, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7,$8)
		RETURNING id`

	err := db.QueryRow(query, tender.Name, tender.Description, tender.Status, tender.ServiceType, tender.AuthorID, tender.OrganizationID, tender.Version, tender.CreatedAt).Scan(&tender.ID)
	err_info.status = sqlErrToStatus(err, http.StatusInternalServerError)
	if err_info.status != 200 {
		err_info.reason = ErrMessageServer
	}
	return err_info

}

func archiveTender(db *sql.DB, tender *Tender) ErrorInfo {
	var err_info ErrorInfo
	query := `
		INSERT INTO tenders_archive (id,name, description, status, service_type, version)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	err := db.QueryRow(query, tender.ID, tender.Name, tender.Description, tender.Status, tender.ServiceType, tender.Version).Scan(&tender.ID)
	err_info.status = sqlErrToStatus(err, http.StatusInternalServerError)
	if err_info.status != 200 {
		err_info.reason = ErrMessageServer
	}
	return err_info

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
		var err_info ErrorInfo
		err_info.status = 200
		var req CreateTenderData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validateNewTender(&req) {
			err_info.reason = ErrMessageWrongRequest
			err_info.status = 400
			sendHttpErr(w, err_info)
			return
		}
		user_id, err_info := isUserExistAndResponsible(db, req.CreatorUsername, req.OrganizationID)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}
		tender := createTenderDataToTender(req, user_id, 1, time.Now())
		tender.OrganizationID = req.OrganizationID
		err_info = createTender(db, req.CreatorUsername, tender)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tender)

	}

}

func getUserTenders(db *sql.DB, user_id, limit, offset int) ([]Tender, ErrorInfo) {
	var err_info ErrorInfo
	err_info.reason = ErrMessageServer
	query := `
	SELECT id, name, description, status, service_type, author_id, version, created_at
	FROM tenders
	WHERE author_id = $1
	ORDER BY name
	LIMIT $2 OFFSET $3
	`
	rows, err := db.Query(query, user_id, limit, offset)
	err_info.status = sqlErrToStatus(err, 200)
	if err_info.status != 200 {
		return nil, err_info
	}
	defer rows.Close()
	var tenders []Tender
	for rows.Next() {
		var tender Tender
		if err := rows.Scan(&tender.ID, &tender.Name, &tender.Description, &tender.Status, &tender.ServiceType, &tender.AuthorID, &tender.Version, &tender.CreatedAt); err != nil {
			err_info.status = sqlErrToStatus(err, 500)
			return nil, err_info
		}
		tenders = append(tenders, tender)
	}
	err_info.status = sqlErrToStatus(rows.Err(), 500)
	return tenders, err_info

}

func myTendersHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		var err_info ErrorInfo

		w.Header().Set("Content-Type", "application/json")

		limit, offset, err_info := getLimitOffsetFromRequest(r)
		var tenders []Tender
		var user_id int
		if err_info.status == 200 {
			user_name := getUserNameFromRequest(r)
			user_id, err_info = getUserId(db, user_name)

		}
		if err_info.status == 200 {
			tenders, err_info = getUserTenders(db, user_id, limit, offset)
		}

		if err_info.status != 200 {
			sendHttpErr(w, err_info)

		} else {

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tenders)
		}
	}
}

func getTender(db *sql.DB, tender_id int) (*Tender, ErrorInfo) {
	var err_info ErrorInfo
	err_info.status = 200

	query := `
    SELECT t.id, t.name, t.description, t.status, t.service_type, 
           t.author_id, t.organization_id, t.version, t.created_at
    FROM tenders t
    WHERE t.id = $1
    `
	rows, err := db.Query(query, tender_id)
	if err != nil {
		err_info = sqlErrToErrInfo(err, 500, ErrMessageServer)
		return nil, err_info
	}
	defer rows.Close()

	var tender Tender
	if rows.Next() {
		if err := rows.Scan(&tender.ID, &tender.Name, &tender.Description,
			&tender.Status, &tender.ServiceType, &tender.AuthorID,
			&tender.OrganizationID, &tender.Version, &tender.CreatedAt); err != nil {
			err_info = sqlErrToErrInfo(err, 500, ErrMessageServer)
			return nil, err_info
		}
		return &tender, err_info
	}
	err_info = sqlErrToErrInfo(sql.ErrNoRows, 404, ErrMessageTenderNotFound)

	return nil, err_info
}

func updateTenderStatus(db *sql.DB, tender_uid int, status string) (string, ErrorInfo) {
	var err_info ErrorInfo
	err_info.status = 200
	query := `
		UPDATE tenders
		SET status = $1
		WHERE id = $2
		RETURNING status
	`

	var updatedStatus string
	err := db.QueryRow(query, status, tender_uid).Scan(&updatedStatus)
	if err != nil {
		err_info.status = sqlErrToStatus(err, 500)
		err_info.reason = ErrMessageServer
		return "", err_info
	}

	return updatedStatus, err_info
}

func handleGetTenderStatus(db *sql.DB, w http.ResponseWriter, r *http.Request, tender_id int) {
	var err_info ErrorInfo
	user_name := getUserNameFromRequest(r)
	if user_name != "" {
		_, err_info = getUserId(db, user_name)
		if err_info.status != 200 {
			err_info.reason = ErrMessageWrongUser
			sendHttpErr(w, err_info)
			return
		}
	}
	tender, err_info := getTender(db, tender_id)
	if err_info.status != 200 {
		sendHttpErr(w, err_info)
		return
	}
	log.Println("err_info.status:", err_info.status)
	log.Println("istender nil", tender == nil)
	w.Write([]byte(tender.Status))
}

func isNewStatusOk(status string) bool {
	return (status != "" && (status == "Created" || status == "Published" || status == "Closed"))
}

func handlePutTenderStatus(db *sql.DB, w http.ResponseWriter, r *http.Request, tender_id int) {
	var err_info ErrorInfo
	user_name := r.URL.Query().Get("username")
	new_status := r.URL.Query().Get("status")
	if user_name == "" || !isNewStatusOk(new_status) {
		err_info.status = 400
		err_info.reason = ErrMessageWrongRequest
		sendHttpErr(w, err_info)
		return
	}
	user_id, err_info := getUserId(db, user_name)
	if err_info.status != 200 {
		sendHttpErr(w, err_info)
		return
	}

	tender, err_info := getTender(db, tender_id)
	if err_info.status != 200 {
		sendHttpErr(w, err_info)
		return
	}
	err_info = isUserInOrganization(db, user_id, tender.OrganizationID)
	if err_info.status != 200 {
		sendHttpErr(w, err_info)
		return
	}

	new_status, err_info = updateTenderStatus(db, tender.ID, new_status)
	if err_info.status != 200 {
		sendHttpErr(w, err_info)
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
		}
	}
}

func updateTender(db *sql.DB, tender *Tender) ErrorInfo {
	var err_info ErrorInfo
	query := `UPDATE tenders 
	SET name = $1, description = $2, service_type = $3, version = $4
	WHERE id = $5
	`
	_, err := db.Exec(query, tender.Name, tender.Description, tender.ServiceType, tender.Version, tender.ID)
	if err != nil {
		log.Println(err)
	}
	err_info.status = sqlErrToStatus(err, 500)
	err_info.reason = ErrMessageServer
	return err_info

}

func editTender(db *sql.DB, username string, tender *Tender, req_body *editTenderRequestBody) ErrorInfo {
	var err_info ErrorInfo
	err_info.status = 200

	err_info = archiveTender(db, tender)
	if err_info.status != 200 {
		return err_info
	}

	tender.Version++
	if req_body.Name != "" {
		tender.Name = req_body.Name
	}
	if req_body.Description != "" {
		tender.Description = req_body.Description
	}
	if req_body.ServiceType != "" {
		tender.ServiceType = req_body.ServiceType
	}

	err_info = updateTender(db, tender)
	log.Println("CantUpdate", tender.OrganizationID)
	return err_info
}

func validateEditTenderParams(req_body *editTenderRequestBody) bool {

	if req_body.Description != "" && len(req_body.Description) > 100 {
		return false
	}
	if req_body.Name != "" && len(req_body.Name) > 100 {
		return false
	}
	if isOkServiceType(req_body.ServiceType) {
		return true
	}
	return false

}

func editTendersHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info ErrorInfo
		vars := mux.Vars(r)
		s_tender_id := vars["tenderId"]
		user_name := r.URL.Query().Get("username")

		var req_body editTenderRequestBody
		tender_id, err_info := atoi(s_tender_id)
		if err := json.NewDecoder(r.Body).Decode(&req_body); err != nil || err_info.status != 200 || !validateEditTenderParams(&req_body) {
			err_info.reason = ErrMessageWrongRequest
			err_info.status = 400
			sendHttpErr(w, err_info)
			return
		}

		tender, err_info := getTender(db, tender_id)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}

		user_id, err_info := getUserId(db, user_name)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}
		err_info = isUserInOrganization(db, user_id, tender.OrganizationID)

		if err_info.status != 200 {

			sendHttpErr(w, err_info)
			return
		}
		err_info = editTender(db, user_name, tender, &req_body)
		if err_info.status != 200 {

			sendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tender)
	}
}

func getArchivedTender(db *sql.DB, tender_id, version int) (tender *Tender, err_info ErrorInfo) {
	err_info.status = 200
	err_info.reason = ErrMessageServer
	tender = &Tender{}
	query := `
    SELECT t.name, t.description, t.status, t.service_type
    FROM tenders_archive t
    WHERE t.id = $1 AND t.version = $2
    `
	rows, err := db.Query(query, tender_id, version)
	if err != nil {
		err_info.status = sqlErrToStatus(err, 500)
		return
	}
	defer rows.Close()

	if rows.Next() {
		err := rows.Scan(&tender.Name, &tender.Description, &tender.Status, &tender.ServiceType)
		tender.Version++
		if err != nil {
			err_info.status = sqlErrToStatus(err, 500)
			return
		}
		return
	}
	err_info.status = 404
	err_info.reason = "This version of tender does not exist."
	return
}

func rollbackTender(db *sql.DB, current_tender, old_tender *Tender) ErrorInfo {
	err_info := archiveTender(db, current_tender)
	if err_info.status != 200 {
		return err_info
	}
	old_tender.Version = current_tender.Version + 1
	old_tender.ID = current_tender.ID
	err_info = updateTender(db, old_tender)
	return err_info
}

func rollbackTendersHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info ErrorInfo
		err_info.status = 200
		vars := mux.Vars(r)
		s_tender_id := vars["tenderId"]
		s_version := vars["version"]
		version, err_info := atoi(s_version)
		tender_id, tmp_err_info := atoi(s_tender_id)

		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}
		if tmp_err_info.status != 200 {
			sendHttpErr(w, tmp_err_info)
			return
		}
		current_tender, err_info := getTender(db, tender_id)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}

		user_name := r.URL.Query().Get("username")
		_, err_info = isUserExistAndResponsible(db, user_name, current_tender.OrganizationID)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}

		old_tender, err_info := getArchivedTender(db, tender_id, version)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}
		err_info = rollbackTender(db, current_tender, old_tender)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(old_tender)
	}
}
