package bids

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"go_server/m/common/helpers"
	"go_server/m/tenders"
)

type Bid struct {
	ID          int       `json:"id"`
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"-"`
	Status      string    `json:"status" binding:"required"`
	AuthorType  string    `json:"author_type" binding:"required"`
	AuthorID    int       `json:"author_id" binding:"required"`
	TenderID    int       `json:"-"`
	Version     int       `json:"version" gorm:"default:1"`
	AproveCount int       `json:"-"`
	CreatedAt   time.Time `json:"created_at" gorm:"default:current_timestamp"`
}

type editBidRequestBody struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

func getBids(db *sql.DB, limit, offset int) ([]Bid, errinfo.ErrorInfo) {
	var query string
	var args []interface{}
	var err_info errinfo.ErrorInfo

	query = `
		SELECT id, name, description, status, author_type,author_id,tender_id, version, created_at 
		FROM bids
		ORDER BY name
		LIMIT $1
		OFFSET $2
		`
	args = []interface{}{limit, offset}

	rows, err := db.Query(query, args...)
	err_info.Status = dbhelp.SqlErrToStatus(err, 500)
	if err_info.Status != 200 {
		err_info.Reason = errinfo.ErrMessageServer
		log.Println(err)
		return nil, err_info
	}
	defer rows.Close()
	var bids []Bid
	for rows.Next() {
		var bid Bid
		if err := rows.Scan(&bid.ID, &bid.Name, &bid.Description, &bid.Status, &bid.AuthorType, &bid.AuthorID, &bid.TenderID, &bid.Version, &bid.CreatedAt); err != nil {
			err_info.Reason = errinfo.ErrMessageServer
			err_info.Status = dbhelp.SqlErrToStatus(err, 500)
			log.Println(err)
			return nil, err_info
		}
		bids = append(bids, bid)
	}
	if err := rows.Err(); err != nil {
		err_info.Reason = errinfo.ErrMessageServer
		err_info.Status = dbhelp.SqlErrToStatus(err, 500)
		log.Println(err)
		return nil, err_info
	}

	return bids, err_info
}

func getBid(db *sql.DB, tender_id int) (*Bid, errinfo.ErrorInfo) {
	var err_info errinfo.ErrorInfo
	err_info.Status = 200

	query := `
		SELECT id, name, description, status, author_type, author_id, tender_id, version, approve_count,created_at 
		FROM bids
		WHERE id = $1
		LIMIT 1
		`
	rows, err := db.Query(query, tender_id)
	if err != nil {
		err_info = dbhelp.SqlErrToErrInfo(err, 500, errinfo.ErrMessageServer)
		return nil, err_info
	}
	defer rows.Close()

	var bid Bid
	if rows.Next() {
		if err := rows.Scan(&bid.ID, &bid.Name, &bid.Description, &bid.Status, &bid.AuthorType, &bid.AuthorID, &bid.TenderID, &bid.Version, &bid.AproveCount, &bid.CreatedAt); err != nil {
			err_info = dbhelp.SqlErrToErrInfo(err, 500, errinfo.ErrMessageServer)
			return nil, err_info
		}
		return &bid, err_info
	}
	err_info = dbhelp.SqlErrToErrInfo(sql.ErrNoRows, 404, errinfo.ErrMessageBidNotFound)

	return nil, err_info
}

func BidsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err_info errinfo.ErrorInfo
		log.Println("GettingBids")
		limit, offset, err_info := helpers.GetLimitOffsetFromRequest(r)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}

		bids, err_info := getBids(db, limit, offset)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bids)
	}
}

func createBidDataToBid(req CreateBidData, version int, created_at time.Time) *Bid {
	return &Bid{
		Name:       req.Name,
		Status:     "Created",
		AuthorType: req.AuthorType,
		AuthorID:   req.AuthorId,
		TenderID:   req.TenderID,
		Version:    version,
		CreatedAt:  created_at,
	}
}

type CreateBidData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TenderID    int    `json:"tenderId"`
	AuthorType  string `json:"authorType"`
	AuthorId    int    `json:"authorId"`
}

func hasUserAccesstoTender(db *sql.DB, user_name string, tender_id int) errinfo.ErrorInfo {

	user_id, err_info := dbhelp.GetUserId(db, user_name)
	if err_info.Status != 200 {
		return err_info
	}
	tender, err_info := tenders.GetTender(db, tender_id)
	if err_info.Status != 200 {
		return err_info
	}
	err_info = dbhelp.IsUserInOrganization(db, user_id, tender.OrganizationID)
	if err_info.Status != 200 {
		return err_info
	}
	return err_info
}
