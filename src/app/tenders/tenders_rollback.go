package tenders

import (
	"database/sql"
	"encoding/json"
	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"go_server/m/common/helpers"
	"net/http"

	"github.com/gorilla/mux"
)

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
