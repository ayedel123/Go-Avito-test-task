package bids

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

func editBid(db *sql.DB, bid *Bid, req_body *editBidRequestBody) errinfo.ErrorInfo {
	var err_info errinfo.ErrorInfo
	err_info.Status = 200

	err_info = archiveBid(db, bid)
	if err_info.Status != 200 {
		return err_info
	}

	bid.Version++
	if req_body.Name != "" {
		bid.Name = req_body.Name
	}
	if req_body.Description != "" {
		bid.Description = req_body.Description
	}

	err_info = updateBid(db, bid)

	return err_info
}

func archiveBid(db *sql.DB, bid *Bid) errinfo.ErrorInfo {
	var err_info errinfo.ErrorInfo
	query := `
		INSERT INTO bids_archive (id,name, description, version)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	err := db.QueryRow(query, bid.ID, bid.Name, bid.Description, bid.Version).Scan(&bid.ID)
	err_info.Status = dbhelp.SqlErrToStatus(err, http.StatusInternalServerError)
	if err_info.Status != 200 {
		err_info.Reason = errinfo.ErrMessageServer
	}
	return err_info

}

func updateBid(db *sql.DB, bid *Bid) errinfo.ErrorInfo {
	var err_info errinfo.ErrorInfo
	query := `UPDATE bids 
	SET name = $1, description = $2, version = $3
	WHERE id = $4
	`
	_, err := db.Exec(query, bid.Name, bid.Description, bid.Version, bid.ID)
	if err != nil {
		log.Println(err)
	}
	err_info.Status = dbhelp.SqlErrToStatus(err, 500)
	err_info.Reason = errinfo.ErrMessageServer
	return err_info

}

func validateEditBidParams(req_body *editBidRequestBody) bool {

	if req_body.Description != "" && len(req_body.Description) > 100 {
		return false
	}
	if req_body.Name != "" && len(req_body.Name) > 100 {
		return false
	}

	return true

}

func EditBidsHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info errinfo.ErrorInfo
		vars := mux.Vars(r)
		s_bid_id := vars["bidId"]
		user_name := r.URL.Query().Get("username")

		var req_body editBidRequestBody
		bid_id, err_info := helpers.ParseUUID(s_bid_id)
		if err := json.NewDecoder(r.Body).Decode(&req_body); err != nil || err_info.Status != 200 || !validateEditBidParams(&req_body) {
			err_info.Reason = errinfo.ErrMessageWrongRequest
			err_info.Status = 400
			log.Println(err_info.Reason)
			errinfo.SendHttpErr(w, err_info)
			return
		}

		bid, err_info := getBid(db, bid_id)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		err_info = hasUserAccesstoTender(db, user_name, bid.TenderID)
		if err_info.Status != 200 {

			errinfo.SendHttpErr(w, err_info)
			return
		}
		err_info = editBid(db, bid, &req_body)
		if err_info.Status != 200 {

			errinfo.SendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bid)
	}
}
