package bids

import (
	"database/sql"
	"encoding/json"
	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"go_server/m/tenders"
	"net/http"
	"time"
)

func validateNewBid(new_bid *CreateBidData) bool {
	if len(new_bid.Name) > 100 {
		return false
	}
	if len(new_bid.Description) > 100 {
		return false
	}
	if new_bid.TenderID <= 0 || new_bid.AuthorId <= 0 {
		return false
	}
	if new_bid.AuthorType != "User" && new_bid.AuthorType != "Organization" {
		return false
	}
	return true
}

func createBid(db *sql.DB, bid *Bid) errinfo.ErrorInfo {

	var err_info errinfo.ErrorInfo
	query := `
		INSERT INTO bids (name, description,status, author_type, author_id, tender_id, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`
	err := db.QueryRow(query, bid.Name, bid.Description, bid.Status, bid.AuthorType, bid.AuthorID, bid.TenderID, bid.Version, bid.CreatedAt).Scan(&bid.ID)
	err_info.Status = dbhelp.SqlErrToStatus(err, http.StatusInternalServerError)
	if err_info.Status != 200 {
		err_info.Reason = errinfo.ErrMessageServer
	}

	return err_info

}

func NewBidHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info errinfo.ErrorInfo
		err_info.Status = 200
		var req CreateBidData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validateNewBid(&req) {
			err_info.Reason = errinfo.ErrMessageWrongRequest
			err_info.Status = 400
			errinfo.SendHttpErr(w, err_info)
			return
		}
		user_name, err_info := dbhelp.GetUserName(db, req.AuthorId)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		tender, err_info := tenders.GetTender(db, req.TenderID)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		_, err_info = dbhelp.IsUserExistAndResponsible(db, user_name, tender.OrganizationID)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		bid := createBidDataToBid(req, 1, time.Now())
		err_info = createBid(db, bid)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bid)

	}

}
