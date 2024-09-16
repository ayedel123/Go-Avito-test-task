package tenders

import (
	"database/sql"
	"encoding/json"
	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"net/http"
	"time"
)

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
