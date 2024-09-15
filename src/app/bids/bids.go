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

	"github.com/gorilla/mux"
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
		SELECT id, name,  description, status, author_type,author_id,tender_id, version, created_at 
		FROM bids
		WHERE tender_id = $1
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
		if err := rows.Scan(&bid.ID, &bid.Name, &bid.Description, &bid.Status, &bid.AuthorType, &bid.AuthorID, &bid.TenderID, &bid.Version, &bid.CreatedAt); err != nil {
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

func createBid(db *sql.DB, bid *Bid) errinfo.ErrorInfo {

	var err_info errinfo.ErrorInfo
	query := `
		INSERT INTO bids (name, description,status, author_type, author_id, tender_id, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`
	err := db.QueryRow(query, bid.Name, &bid.Description, bid.Status, bid.AuthorType, bid.AuthorID, bid.TenderID, bid.Version, bid.CreatedAt).Scan(&bid.ID)
	err_info.Status = dbhelp.SqlErrToStatus(err, http.StatusInternalServerError)
	if err_info.Status != 200 {
		err_info.Reason = errinfo.ErrMessageServer
	}

	return err_info

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

func getUserBids(db *sql.DB, user_id, limit, offset int) ([]Bid, errinfo.ErrorInfo) {
	var err_info errinfo.ErrorInfo
	err_info.Reason = errinfo.ErrMessageServer
	query := `
	SELECT id, name, description,, status,author_type , author_id, tender_id,version, created_at
	FROM bids
	WHERE author_id = $1
	ORDER BY name
	LIMIT $2 OFFSET $3
	`
	rows, err := db.Query(query, user_id, limit, offset)
	err_info.Status = dbhelp.SqlErrToStatus(err, 200)
	if err_info.Status != 200 {
		return nil, err_info
	}
	defer rows.Close()
	var bids []Bid
	for rows.Next() {
		var bid Bid
		if err := rows.Scan(&bid.ID, &bid.Name, &bid.Description, &bid.Status, &bid.AuthorType, &bid.AuthorID, &bid.TenderID, &bid.Version, &bid.CreatedAt); err != nil {
			err_info.Status = dbhelp.SqlErrToStatus(err, 500)
			return nil, err_info
		}
		bids = append(bids, bid)
	}
	err_info.Status = dbhelp.SqlErrToStatus(rows.Err(), 500)
	return bids, err_info

}

func MyBidsHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		var err_info errinfo.ErrorInfo

		w.Header().Set("Content-Type", "application/json")

		limit, offset, err_info := helpers.GetLimitOffsetFromRequest(r)
		var bids []Bid
		var user_id int
		if err_info.Status == 200 {
			user_name := r.URL.Query().Get("username")
			user_id, err_info = dbhelp.GetUserId(db, user_name)

		}
		if err_info.Status == 200 {
			bids, err_info = getUserBids(db, user_id, limit, offset)
		}

		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)

		} else {

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bids)
		}
	}
}

func GetTenderBids(db *sql.DB, tender_id, limit, offset int) ([]Bid, errinfo.ErrorInfo) {
	var err_info errinfo.ErrorInfo
	err_info.Reason = errinfo.ErrMessageServer
	query := `
	SELECT id, name,description, status,author_type , author_id, tender_id,version, created_at
	FROM bids
	WHERE tender_id = $1
	ORDER BY name
	LIMIT $2 OFFSET $3
	`
	rows, err := db.Query(query, tender_id, limit, offset)
	err_info.Status = dbhelp.SqlErrToStatus(err, 200)
	if err_info.Status != 200 {
		return nil, err_info
	}
	defer rows.Close()
	var bids []Bid
	for rows.Next() {
		var bid Bid
		if err := rows.Scan(&bid.ID, &bid.Name, &bid.Description, &bid.Status, &bid.AuthorType, &bid.AuthorID, &bid.TenderID, &bid.Version, &bid.CreatedAt); err != nil {
			err_info.Status = dbhelp.SqlErrToStatus(err, 500)
			return nil, err_info
		}
		bids = append(bids, bid)
	}
	err_info.Status = dbhelp.SqlErrToStatus(rows.Err(), 500)
	return bids, err_info

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

func ListBidsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err_info errinfo.ErrorInfo
		vars := mux.Vars(r)
		s_tender_id := vars["tenderId"]
		user_name := r.URL.Query().Get("username")
		tender_id, err_info := helpers.Atoi(s_tender_id)
		limit, offset, tmp_err_info := helpers.GetLimitOffsetFromRequest(r)
		if err_info.Status != 200 || tmp_err_info.Status != 200 || user_name == "" {
			err_info.Reason = errinfo.ErrMessageWrongRequest
			err_info.Status = 400
			errinfo.SendHttpErr(w, err_info)
			return
		}

		err_info = hasUserAccesstoTender(db, user_name, tender_id)
		log.Println("status ", err_info.Status)
		log.Println("reason", err_info.Reason)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		bids, err_info := GetTenderBids(db, tender_id, limit, offset)
		if err_info.Status != 200 {
			errinfo.SendHttpErr(w, err_info)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bids)

	}
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

func editBid(db *sql.DB, username string, bid *Bid, req_body *editBidRequestBody) errinfo.ErrorInfo {
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
		bid_id, err_info := helpers.Atoi(s_bid_id)
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
		err_info = editBid(db, user_name, bid, &req_body)
		if err_info.Status != 200 {

			errinfo.SendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bid)
	}
}

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
		bid_id, tmp_err_info := helpers.Atoi(s_bid_id)

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
