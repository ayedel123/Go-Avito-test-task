package tenders

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"go_server/m/common/helpers"

	"github.com/google/uuid"
)

type Tender struct {
	ID             uuid.UUID `json:"id"`
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
		if err_info.Status != 200 || !helpers.IsOkServiceType(service_type) {
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
		if err_info.Status != 200 || !helpers.IsOkServiceType(service_type) {
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

func GetTender(db *sql.DB, tender_id uuid.UUID) (*Tender, errinfo.ErrorInfo) {
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
