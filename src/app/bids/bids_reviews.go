package bids

import (
	"database/sql"
	"encoding/json"
	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"go_server/m/common/helpers"
	"go_server/m/tenders"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func getReviews(db *sql.DB, tender_id uuid.UUID, user_id, limit, offset int) ([]BidReview, errinfo.ErrorInfo) {
	var err_info errinfo.ErrorInfo
	query := `
		SELECT br.id, br.bid_id, br.author_name, br.description, br.created_at 
		FROM bids_reviews br
		JOIN (SELECT id, tender_id FROM bids WHERE tender_id = $3 AND author_id = $4) AS b ON b.id = br.bid_id
		LIMIT $1
		OFFSET $2
	`
	var reviews []BidReview
	rows, err := db.Query(query, limit, offset, tender_id, user_id)
	err_info = dbhelp.SqlErrToErrInfo(err, 404, "Reviews not found.")
	if err_info.Status != 200 {
		return nil, err_info
	}
	defer rows.Close()

	for rows.Next() {
		var review BidReview
		if err := rows.Scan(&review.Id, &review.BidId, &review.AuthorName, &review.Description, &review.CreatedAt); err != nil {
			err_info.Status = dbhelp.SqlErrToStatus(err, 500)
			return nil, err_info
		}
		reviews = append(reviews, review)
	}
	err_info.Status = dbhelp.SqlErrToStatus(rows.Err(), 500)

	return reviews, err_info

}

func checkReviewParams(db *sql.DB, r *http.Request) (author_id int, tender_id uuid.UUID, err_info errinfo.ErrorInfo) {
	err_info.Init(200, "Ok")
	vars := mux.Vars(r)
	s_tender_id := vars["tenderId"]
	author_name := r.URL.Query().Get("authorUsername")
	requester_name := r.URL.Query().Get("requesterUsername")
	author_id = 0

	if len(s_tender_id) > 100 || len(s_tender_id) == 0 {
		err_info.Status = 400
		err_info.Reason = errinfo.ErrMessageWrongRequest
	}

	author_id, err_info = dbhelp.GetUserId(db, author_name)
	if err_info.Status != 200 {
		return
	}

	tender_id, err_info = helpers.ParseUUID(s_tender_id)
	if err_info.Status != 200 {
		return
	}
	tender, err_info := tenders.GetTender(db, tender_id)
	if err_info.Status != 200 {
		return
	}

	requester_id, err_info := dbhelp.IsUserExistAndResponsible(db, requester_name, tender.OrganizationID)
	if err_info.Status != 200 {
		return
	}

	if tender.AuthorID != requester_id {
		err_info.Init(403, "User is not tender author.")
	}
	return

}

func ReviewsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err_info errinfo.ErrorInfo
		err_info.Init(200, "Ok")
		limit, offset, err_info := helpers.GetLimitOffsetFromRequest(r)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		author_id, tender_id, err_info := checkReviewParams(db, r)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		reviews, err_info := getReviews(db, tender_id, author_id, limit, offset)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reviews)
	}
}
