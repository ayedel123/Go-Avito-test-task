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

func handlePutBidStatus(db *sql.DB, w http.ResponseWriter, r *http.Request, bid_id int) {
	var err_info errinfo.ErrorInfo
	user_name := r.URL.Query().Get("username")
	new_status := r.URL.Query().Get("status")
	if user_name == "" || !helpers.IsNewStatusOk(new_status) {
		err_info.Status = 400
		err_info.Reason = errinfo.ErrMessageWrongRequest
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

	new_status, err_info = updateBidStatus(db, bid_id, new_status)
	if err_info.Status != 200 {
		errinfo.SendHttpErr(w, err_info)
		return
	}
	bid.Status = new_status
	json.NewEncoder(w).Encode(bid)
}

func handleGetBidStatus(db *sql.DB, w http.ResponseWriter, r *http.Request, bid_id int) {
	var err_info errinfo.ErrorInfo
	user_name := r.URL.Query().Get("username")

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

	w.Write([]byte(bid.Status))
}

func updateBidStatus(db *sql.DB, bid_id int, status string) (string, errinfo.ErrorInfo) {
	var err_info errinfo.ErrorInfo
	err_info.Status = 200
	query := `
		UPDATE bids
		SET status = $1
		WHERE id = $2
		RETURNING status
	`

	var updatedStatus string
	err := db.QueryRow(query, status, bid_id).Scan(&updatedStatus)
	if err != nil {
		err_info.Status = dbhelp.SqlErrToStatus(err, 500)
		err_info.Reason = errinfo.ErrMessageServer
		return "", err_info
	}

	return updatedStatus, err_info
}

func StatusBidsHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info errinfo.ErrorInfo
		err_info.Status = 200
		vars := mux.Vars(r)
		s_bid_id := vars["bidId"]
		w.Header().Set("Content-Type", "application/json")
		bid_id, err_info := helpers.Atoi(s_bid_id)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		log.Println("status handling ", s_bid_id)

		if r.Method == http.MethodGet {
			handleGetBidStatus(db, w, r, bid_id)

		} else if r.Method == http.MethodPut {
			handlePutBidStatus(db, w, r, bid_id)
		}
	}
}
