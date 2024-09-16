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

func getArchivedBid(db *sql.DB, current_bid *Bid, version int) (bid *Bid, err_info errinfo.ErrorInfo) {
	err_info.Status = 200
	err_info.Reason = errinfo.ErrMessageServer
	bid = &Bid{}
	*bid = *current_bid
	query := `
    SELECT t.name, t.description
    FROM bids_archive t
    WHERE t.id = $1 AND t.version = $2
    `

	rows, err := db.Query(query, bid.ID, version)
	if err != nil {
		err_info.Status = dbhelp.SqlErrToStatus(err, 500)
		return
	}
	defer rows.Close()

	if rows.Next() {
		err := rows.Scan(&bid.Name, &bid.Description)
		bid.Version++
		if err != nil {
			err_info.Status = dbhelp.SqlErrToStatus(err, 500)
			return
		}
		return
	}
	err_info.Status = 404
	err_info.Reason = "This version of bid does not exist."
	return
}

func rollbackBid(db *sql.DB, current_bid, old_bid *Bid) errinfo.ErrorInfo {
	err_info := archiveBid(db, current_bid)
	if err_info.Status != 200 {
		return err_info
	}
	old_bid.Version++
	err_info = updateBid(db, old_bid)
	return err_info
}

func RollbackBidsHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info errinfo.ErrorInfo
		err_info.Status = 200
		vars := mux.Vars(r)
		s_bid_id := vars["bidId"]
		s_version := vars["version"]
		version, err_info := helpers.Atoi(s_version)
		bid_id, tmp_err_info := helpers.ParseUUID(s_bid_id)

		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			log.Println("errr1")
			return
		}
		if tmp_err_info.Status != 200 {
			errinfo.SendHttpErr(w, tmp_err_info)
			log.Println("errr1")
			return
		}
		current_bid, err_info := getBid(db, bid_id)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			log.Println("errr1")
			return
		}

		user_name := r.URL.Query().Get("username")
		err_info = hasUserAccesstoTender(db, user_name, current_bid.TenderID)
		old_bid := &Bid{}
		*old_bid = *current_bid
		old_bid, err_info = getArchivedBid(db, current_bid, version)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			log.Println("errr1")
			return
		}
		err_info = rollbackBid(db, current_bid, old_bid)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			log.Println("errr1")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(old_bid)
	}
}
