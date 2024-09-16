package bids

import (
	"database/sql"
	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"go_server/m/common/helpers"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func checkSubmitDecisionParams(db *sql.DB, r *http.Request) (bid *Bid, err_info errinfo.ErrorInfo) {
	err_info.Init(200, "Ok")
	bid = &Bid{}
	vars := mux.Vars(r)
	s_bid_id := vars["bidId"]
	user_name := r.URL.Query().Get("username")
	decision := r.URL.Query().Get("decision")
	log.Println("user", user_name)
	if decision != "Approved" && decision != "Rejected" {
		log.Println("Wrong decision")
		err_info.Status = 400
		err_info.Reason = errinfo.ErrMessageWrongRequest
		return
	}

	bid_id, err_info := helpers.ParseUUID(s_bid_id)
	if err_info.Status != 200 {
		log.Println("ID not numb")
		return
	}
	bid, err_info = getBid(db, bid_id)
	if err_info.Status != 200 {
		log.Println("NO BID", bid_id)
		return
	}

	err_info = hasUserAccesstoTender(db, user_name, bid.TenderID)
	log.Println("user", user_name)
	if err_info.Status != 200 {
		return
	}

	return
}

func getResponsibleCount(db *sql.DB, tender_id uuid.UUID) (count int, err_info errinfo.ErrorInfo) {

	err_info.Init(200, "Ok")
	count = 0
	query := `
		SELECT COUNT(*) 
		FROM tenders t
		JOIN organization_responsible or ON t.organization_id = or.organization_id
		WHERE t.id = $1

	`

	err := db.QueryRow(query, tender_id).Scan(&count)
	if err != nil {
		err_info.Status = dbhelp.SqlErrToStatus(err, http.StatusInternalServerError)
		if err_info.Status != 200 {
			err_info.Reason = errinfo.ErrMessageServer
		}
		return count, err_info
	}

	return count, err_info

}

func updateAproveCount(db *sql.DB, bid_id uuid.UUID, count int) (int, errinfo.ErrorInfo) {
	var err_info errinfo.ErrorInfo
	err_info.Status = 200
	query := `
		UPDATE bids
		SET approve_count = $1
		WHERE id = $2
		RETURNING status
	`

	err := db.QueryRow(query, count, bid_id).Scan(&count)
	if err != nil {
		err_info.Status = dbhelp.SqlErrToStatus(err, 500)
		err_info.Reason = errinfo.ErrMessageServer
		return count, err_info
	}

	return count, err_info
}

func SubmitDecisionHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decision := r.URL.Query().Get("decision")
		bid, err_info := checkSubmitDecisionParams(db, r)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		if decision == "Rejected" {
			updateBidStatus(db, bid.ID, "Closed")
		} else if bid.Status != "Closed" {
			resp_count, err_info := getResponsibleCount(db, bid.TenderID)
			if err_info.Status != 200 {
				errinfo.SendHttpErr(w, err_info)
				return
			}
			bid.AproveCount++
			_, err_info = updateAproveCount(db, bid.ID, bid.AproveCount)
			if err_info.Status != 200 {
				errinfo.SendHttpErr(w, err_info)
				return
			}
			if bid.AproveCount >= min(3, resp_count) {
				updateBidStatus(db, bid.ID, "Published")
			}
			if err_info.Status != 200 {
				errinfo.SendHttpErr(w, err_info)
				return
			}
		}

	}
}
