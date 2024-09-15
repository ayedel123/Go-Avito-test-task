package tenders

import (
	"database/sql"
	"encoding/json"
	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"go_server/m/common/helpers"
	"net/http"
)

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
