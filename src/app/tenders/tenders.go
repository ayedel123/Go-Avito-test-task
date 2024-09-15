package tenders

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"go_server/m/common/helpers"

	"github.com/gorilla/mux"
)

type Tender struct {
	ID             int       `json:"id"`
	Name           string    `json:"name" binding:"required"`
	Description    string    `json:"description" binding:"required"`
	Status         string    `json:"status" binding:"required"`
	ServiceType    string    `json:"service_type"`
	AuthorID       int       `json:"-"`
	OrganizationID int       `json:"-"`
	Version        int       `json:"version" gorm:"default:1"`
	CreatedAt      time.Time `json:"created_at" gorm:"default:current_timestamp"`
}

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

// func getIntFromRequest(r *http.Request, default_val int, param_name string) (num int, err_info errinfo.ErrorInfo) {
// 	s_param := r.URL.Query().Get(param_name)
// 	if default_val != -1 && s_param == "" {
// 		s_param = strconv.Itoa(default_val)
// 	}
// 	num, err_info = helpers.Atoi(s_param)

// 	return
// }

// func getUserName(db *sql.DB, user_id int) (user_name string, err_info errinfo.ErrorInfo) {
// 	user_name = ""
// 	query := `
//         SELECT e.username
//         FROM  employee e WHERE e.id = $1
// 		LIMIT 1
//     `
// 	err := db.QueryRow(query, user_id).Scan(&user_name)

// 	err_info.Status = dbhelp.SqlErrToStatus(err, 401)
// 	if err_info.Status != 200 {
// 		err_info.Reason = "User does not exist."
// 	}

// 	return
// }

func GetTenders(db *sql.DB, limit, offset int, service_type string) ([]Tender, errinfo.ErrorInfo) {
	var query string
	var args []interface{}
	var err_info errinfo.ErrorInfo
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
	err_info.Status = dbhelp.SqlErrToStatus(err, 500)
	if err_info.Status != 200 {
		err_info.Reason = errinfo.ErrMessageServer
		return nil, err_info
	}
	defer rows.Close()
	var tenders []Tender
	for rows.Next() {
		var tender Tender
		if err := rows.Scan(&tender.ID, &tender.Name, &tender.Description, &tender.Status, &tender.ServiceType, &tender.AuthorID, &tender.Version, &tender.CreatedAt); err != nil {
			err_info.Reason = errinfo.ErrMessageServer
			err_info.Status = dbhelp.SqlErrToStatus(err, 500)
			return nil, err_info
		}
		tenders = append(tenders, tender)
	}
	if err := rows.Err(); err != nil {
		err_info.Reason = errinfo.ErrMessageServer
		err_info.Status = dbhelp.SqlErrToStatus(err, 500)
		return nil, err_info
	}

	return tenders, err_info
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

func TendersArchiveHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info errinfo.ErrorInfo
		if r.Method != http.MethodGet {
			err_info.Reason = errinfo.ErrMessageMethodNotAllowed
			err_info.Status = 405
			errinfo.SendHttpErr(w, err_info)
			return
		}
		service_type := r.URL.Query().Get("service_type")
		limit, offset, err_info := helpers.GetLimitOffsetFromRequest(r)
		if err_info.Status != 200 || !isOkServiceType(service_type) {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		tenders, err := getArchivedTenders(db, limit, offset, service_type)
		if err != nil {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tenders)
	}
}

func TendersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err_info errinfo.ErrorInfo
		service_type := r.URL.Query().Get("service_type")
		limit, offset, err_info := helpers.GetLimitOffsetFromRequest(r)
		if err_info.Status != 200 || !isOkServiceType(service_type) {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		tenders, err_info := GetTenders(db, limit, offset, service_type)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tenders)
	}
}

func createTender(db *sql.DB, tender *Tender) errinfo.ErrorInfo {
	var err_info errinfo.ErrorInfo
	query := `
		INSERT INTO tenders (name, description, status, service_type, author_id,organization_id, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7,$8)
		RETURNING id`

	err := db.QueryRow(query, tender.Name, tender.Description, tender.Status, tender.ServiceType, tender.AuthorID, tender.OrganizationID, tender.Version, tender.CreatedAt).Scan(&tender.ID)
	err_info.Status = dbhelp.SqlErrToStatus(err, http.StatusInternalServerError)
	if err_info.Status != 200 {
		err_info.Reason = errinfo.ErrMessageServer
	}
	return err_info

}

func archiveTender(db *sql.DB, tender *Tender) errinfo.ErrorInfo {
	var err_info errinfo.ErrorInfo
	query := `
		INSERT INTO tenders_archive (id,name, description, status, service_type, version)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	err := db.QueryRow(query, tender.ID, tender.Name, tender.Description, tender.Status, tender.ServiceType, tender.Version).Scan(&tender.ID)
	err_info.Status = dbhelp.SqlErrToStatus(err, http.StatusInternalServerError)
	if err_info.Status != 200 {
		err_info.Reason = errinfo.ErrMessageServer
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

func NewTenderHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info errinfo.ErrorInfo
		err_info.Status = 200
		var req CreateTenderData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validateNewTender(&req) {
			err_info.Reason = errinfo.ErrMessageWrongRequest
			err_info.Status = 400
			errinfo.SendHttpErr(w, err_info)
			return
		}
		user_id, err_info := dbhelp.IsUserExistAndResponsible(db, req.CreatorUsername, req.OrganizationID)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		tender := createTenderDataToTender(req, user_id, 1, time.Now())
		tender.OrganizationID = req.OrganizationID
		err_info = createTender(db, tender)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tender)

	}

}

func getUserTenders(db *sql.DB, user_id, limit, offset int) ([]Tender, errinfo.ErrorInfo) {
	var err_info errinfo.ErrorInfo
	err_info.Reason = errinfo.ErrMessageServer
	query := `
	SELECT id, name, description, status, service_type, author_id, version, created_at
	FROM tenders
	WHERE author_id = $1
	ORDER BY name
	LIMIT $2 OFFSET $3
	`
	rows, err := db.Query(query, user_id, limit, offset)
	err_info.Status = dbhelp.SqlErrToStatus(err, 200)
	if err_info.Status != 200 {
		return nil, err_info
	}
	defer rows.Close()
	var tenders []Tender
	for rows.Next() {
		var tender Tender
		if err := rows.Scan(&tender.ID, &tender.Name, &tender.Description, &tender.Status, &tender.ServiceType, &tender.AuthorID, &tender.Version, &tender.CreatedAt); err != nil {
			err_info.Status = dbhelp.SqlErrToStatus(err, 500)
			return nil, err_info
		}
		tenders = append(tenders, tender)
	}
	err_info.Status = dbhelp.SqlErrToStatus(rows.Err(), 500)
	return tenders, err_info

}

func MyTendersHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		var err_info errinfo.ErrorInfo

		w.Header().Set("Content-Type", "application/json")

		limit, offset, err_info := helpers.GetLimitOffsetFromRequest(r)
		var tenders []Tender
		var user_id int
		if err_info.Status == 200 {
			user_name := r.URL.Query().Get("username")
			user_id, err_info = dbhelp.GetUserId(db, user_name)

		}
		if err_info.Status == 200 {
			tenders, err_info = getUserTenders(db, user_id, limit, offset)
		}

		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tenders)
		}
	}
}

func GetTender(db *sql.DB, tender_id int) (*Tender, errinfo.ErrorInfo) {
	var err_info errinfo.ErrorInfo
	err_info.Status = 200

	query := `
    SELECT t.id, t.name, t.description, t.status, t.service_type, 
           t.author_id, t.organization_id, t.version, t.created_at
    FROM tenders t
    WHERE t.id = $1
    `
	rows, err := db.Query(query, tender_id)
	if err != nil {
		err_info = dbhelp.SqlErrToErrInfo(err, 500, errinfo.ErrMessageServer)
		return nil, err_info
	}
	defer rows.Close()

	var tender Tender
	if rows.Next() {
		if err := rows.Scan(&tender.ID, &tender.Name, &tender.Description,
			&tender.Status, &tender.ServiceType, &tender.AuthorID,
			&tender.OrganizationID, &tender.Version, &tender.CreatedAt); err != nil {
			err_info = dbhelp.SqlErrToErrInfo(err, 500, errinfo.ErrMessageServer)
			return nil, err_info
		}
		return &tender, err_info
	}
	err_info = dbhelp.SqlErrToErrInfo(sql.ErrNoRows, 404, errinfo.ErrMessageTenderNotFound)

	return nil, err_info
}

func updateTenderStatus(db *sql.DB, tender_uid int, status string) (string, errinfo.ErrorInfo) {
	var err_info errinfo.ErrorInfo
	err_info.Status = 200
	query := `
		UPDATE tenders
		SET status = $1
		WHERE id = $2
		RETURNING status
	`

	var updatedStatus string
	err := db.QueryRow(query, status, tender_uid).Scan(&updatedStatus)
	if err != nil {
		err_info.Status = dbhelp.SqlErrToStatus(err, 500)
		err_info.Reason = errinfo.ErrMessageServer
		return "", err_info
	}

	return updatedStatus, err_info
}

func handleGetTenderStatus(db *sql.DB, w http.ResponseWriter, r *http.Request, tender_id int) {
	var err_info errinfo.ErrorInfo
	user_name := r.URL.Query().Get("username")
	if user_name != "" {
		_, err_info = dbhelp.GetUserId(db, user_name)
		if err_info.Status != 200 {
			err_info.Reason = errinfo.ErrMessageWrongUser
			errinfo.SendHttpErr(w, err_info)
			return
		}
	}
	tender, err_info := GetTender(db, tender_id)
	if err_info.Status != 200 {
		errinfo.SendHttpErr(w, err_info)
		return
	}

	w.Write([]byte(tender.Status))
}

func handlePutTenderStatus(db *sql.DB, w http.ResponseWriter, r *http.Request, tender_id int) {
	var err_info errinfo.ErrorInfo
	user_name := r.URL.Query().Get("username")
	new_status := r.URL.Query().Get("status")
	if user_name == "" || !helpers.IsNewStatusOk(new_status) {
		err_info.Status = 400
		err_info.Reason = errinfo.ErrMessageWrongRequest
		errinfo.SendHttpErr(w, err_info)
		return
	}

	tender, err_info := GetTender(db, tender_id)
	if err_info.Status != 200 {
		errinfo.SendHttpErr(w, err_info)
		return
	}

	_, err_info = dbhelp.IsUserExistAndResponsible(db, user_name, tender.OrganizationID)
	if err_info.Status != 200 {
		errinfo.SendHttpErr(w, err_info)
		return
	}

	new_status, err_info = updateTenderStatus(db, tender.ID, new_status)
	if err_info.Status != 200 {
		errinfo.SendHttpErr(w, err_info)
		return
	}
	tender.Status = new_status
	json.NewEncoder(w).Encode(tender)
}

func StatusTendersHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		s_tender_id := r.URL.Path[len("/api/tenders/") : len(r.URL.Path)-len("/status")]
		tender_id, _ := helpers.Atoi(s_tender_id)
		log.Println("status handling ", s_tender_id)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			handleGetTenderStatus(db, w, r, tender_id)

		} else if r.Method == http.MethodPut {
			handlePutTenderStatus(db, w, r, tender_id)
		}
	}
}

func updateTender(db *sql.DB, tender *Tender) errinfo.ErrorInfo {
	var err_info errinfo.ErrorInfo
	query := `UPDATE tenders 
	SET name = $1, description = $2, service_type = $3, version = $4
	WHERE id = $5
	`
	_, err := db.Exec(query, tender.Name, tender.Description, tender.ServiceType, tender.Version, tender.ID)
	if err != nil {
		log.Println(err)
	}
	err_info.Status = dbhelp.SqlErrToStatus(err, 500)
	err_info.Reason = errinfo.ErrMessageServer
	return err_info

}

func editTender(db *sql.DB, tender *Tender, req_body *editTenderRequestBody) errinfo.ErrorInfo {
	var err_info errinfo.ErrorInfo
	err_info.Status = 200

	err_info = archiveTender(db, tender)
	if err_info.Status != 200 {
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

func EditTendersHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info errinfo.ErrorInfo
		vars := mux.Vars(r)
		s_tender_id := vars["tenderId"]
		user_name := r.URL.Query().Get("username")

		var req_body editTenderRequestBody
		tender_id, err_info := helpers.Atoi(s_tender_id)
		if err := json.NewDecoder(r.Body).Decode(&req_body); err != nil || err_info.Status != 200 || !validateEditTenderParams(&req_body) {
			err_info.Reason = errinfo.ErrMessageWrongRequest
			err_info.Status = 400
			errinfo.SendHttpErr(w, err_info)
			return
		}

		tender, err_info := GetTender(db, tender_id)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		_, err_info = dbhelp.IsUserExistAndResponsible(db, user_name, tender.OrganizationID)

		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		err_info = editTender(db, tender, &req_body)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tender)
	}
}

func getArchivedTender(db *sql.DB, tender_id, version int) (tender *Tender, err_info errinfo.ErrorInfo) {
	err_info.Status = 200
	err_info.Reason = errinfo.ErrMessageServer
	tender = &Tender{}
	query := `
    SELECT t.name, t.description, t.status, t.service_type
    FROM tenders_archive t
    WHERE t.id = $1 AND t.version = $2
    `
	rows, err := db.Query(query, tender_id, version)
	if err != nil {
		err_info.Status = dbhelp.SqlErrToStatus(err, 500)
		return
	}
	defer rows.Close()

	if rows.Next() {
		err := rows.Scan(&tender.Name, &tender.Description, &tender.Status, &tender.ServiceType)
		tender.Version++
		if err != nil {
			err_info.Status = dbhelp.SqlErrToStatus(err, 500)
			return
		}
		return
	}
	err_info.Status = 404
	err_info.Reason = "This version of tender does not exist."
	return
}

func rollbackTender(db *sql.DB, current_tender, old_tender *Tender) errinfo.ErrorInfo {
	err_info := archiveTender(db, current_tender)
	if err_info.Status != 200 {
		return err_info
	}
	old_tender.Version = current_tender.Version + 1
	old_tender.ID = current_tender.ID
	err_info = updateTender(db, old_tender)
	return err_info
}

func RollbackTendersHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info errinfo.ErrorInfo
		err_info.Status = 200
		vars := mux.Vars(r)
		s_tender_id := vars["tenderId"]
		s_version := vars["version"]
		version, err_info := helpers.Atoi(s_version)
		tender_id, tmp_err_info := helpers.Atoi(s_tender_id)

		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		if tmp_err_info.Status != 200 {
			errinfo.SendHttpErr(w, tmp_err_info)
			return
		}
		current_tender, err_info := GetTender(db, tender_id)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		user_name := r.URL.Query().Get("username")
		_, err_info = dbhelp.IsUserExistAndResponsible(db, user_name, current_tender.OrganizationID)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		old_tender, err_info := getArchivedTender(db, tender_id, version)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		err_info = rollbackTender(db, current_tender, old_tender)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(old_tender)
	}
}
