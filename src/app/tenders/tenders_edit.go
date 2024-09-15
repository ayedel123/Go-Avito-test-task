package tenders

import (
	"database/sql"
	"encoding/json"
	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"go_server/m/common/helpers"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

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
	if helpers.IsOkServiceType(req_body.ServiceType) {
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
