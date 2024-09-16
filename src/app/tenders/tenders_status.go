package tenders

import (
	"database/sql"
	"encoding/json"
	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"go_server/m/common/helpers"
	"log"
	"net/http"

	"github.com/google/uuid"
)

func updateTenderStatus(db *sql.DB, tender_uid uuid.UUID, status string) (string, errinfo.ErrorInfo) {
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

func handleGetTenderStatus(db *sql.DB, w http.ResponseWriter, r *http.Request, tender_id uuid.UUID) {
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

func handlePutTenderStatus(db *sql.DB, w http.ResponseWriter, r *http.Request, tender_id uuid.UUID) {
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
		tender_id, _ := helpers.ParseUUID(s_tender_id)
		log.Println("status handling ", s_tender_id)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			handleGetTenderStatus(db, w, r, tender_id)

		} else if r.Method == http.MethodPut {
			handlePutTenderStatus(db, w, r, tender_id)
		}
	}
}
