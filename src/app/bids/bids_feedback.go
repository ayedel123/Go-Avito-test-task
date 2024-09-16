package bids

import (
	"database/sql"
	"encoding/json"
	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"go_server/m/common/helpers"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type BidReview struct {
	Id          uuid.UUID `json:"id"`
	Description string    `json:"description" binding:"required"`
	BidId       uuid.UUID `json:"-"`
	AuthorName  string    `json:"-"`
	CreatedAt   time.Time `json:"created_at" gorm:"default:current_timestamp"`
}

func createReview(db *sql.DB, bid_review *BidReview) errinfo.ErrorInfo {
	var err_info errinfo.ErrorInfo
	query := `
		INSERT INTO bids_reviews (bid_id, author_name ,description)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err := db.QueryRow(query, bid_review.BidId, bid_review.Description, bid_review.Description).Scan(&bid_review.Id)
	err_info.Status = dbhelp.SqlErrToStatus(err, http.StatusInternalServerError)
	if err_info.Status != 200 {
		err_info.Reason = errinfo.ErrMessageServer
	}

	return err_info

}

func checkFeedbackParams(db *sql.DB, r *http.Request) (bid_review BidReview, err_info errinfo.ErrorInfo) {
	err_info.Init(200, "Ok")
	vars := mux.Vars(r)
	s_bid_id := vars["bidId"]
	user_name := r.URL.Query().Get("username")
	review := r.URL.Query().Get("bidFeedback")

	if len(review) > 1000 || len(review) == 0 {
		log.Println("Review LEN ", len(review))
		err_info.Status = 400
		err_info.Reason = errinfo.ErrMessageWrongRequest
		return
	}

	bid_id, err_info := helpers.ParseUUID(s_bid_id)
	if err_info.Status != 200 {
		log.Println("ID not numb")
		return
	}
	bid, err_info := getBid(db, bid_id)
	if err_info.Status != 200 {
		log.Println("NO BID", bid_id)
		return
	}

	err_info = hasUserAccesstoTender(db, user_name, bid.TenderID)

	if err_info.Status != 200 {
		return
	}
	bid_review.Description = review
	bid_review.AuthorName = user_name
	bid_review.BidId = bid_id
	return
}

func FeedbackHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		bid_review, err_info := checkFeedbackParams(db, r)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		createReview(db, &bid_review)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bid_review)

	}
}
