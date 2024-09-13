package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type editBidRequestBody struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

func getBids(db *sql.DB, limit, offset int) ([]Bid, ErrorInfo) {
	var query string
	var args []interface{}
	var error_info ErrorInfo

	query = `
		SELECT id, name, description, status, author_type,author_id,tender_id, version, created_at 
		FROM bids
		ORDER BY name
		LIMIT $1
		OFFSET $2
		`
	args = []interface{}{limit, offset}

	rows, err := db.Query(query, args...)
	error_info.status = sqlErrToStatus(err, 500)
	if error_info.status != 200 {
		error_info.reason = ErrMessageServer
		log.Println(err)
		return nil, error_info
	}
	defer rows.Close()
	var bids []Bid
	for rows.Next() {
		var bid Bid
		if err := rows.Scan(&bid.ID, &bid.Name, &bid.Description, &bid.Status, &bid.AuthorType, &bid.AuthorID, &bid.TenderID, &bid.Version, &bid.CreatedAt); err != nil {
			error_info.reason = ErrMessageServer
			error_info.status = sqlErrToStatus(err, 500)
			log.Println(err)
			return nil, error_info
		}
		bids = append(bids, bid)
	}
	if err := rows.Err(); err != nil {
		error_info.reason = ErrMessageServer
		error_info.status = sqlErrToStatus(err, 500)
		log.Println(err)
		return nil, error_info
	}

	return bids, error_info
}

func getBid(db *sql.DB, tender_id int) (*Bid, ErrorInfo) {
	var err_info ErrorInfo
	err_info.status = 200

	query := `
		SELECT id, name,  description, status, author_type,author_id,tender_id, version, created_at 
		FROM bids
		WHERE tender_id = $1
		LIMIT 1
		`
	rows, err := db.Query(query, tender_id)
	if err != nil {
		err_info = sqlErrToErrInfo(err, 500, ErrMessageServer)
		return nil, err_info
	}
	defer rows.Close()

	var bid Bid
	if rows.Next() {
		if err := rows.Scan(&bid.ID, &bid.Name, &bid.Description, &bid.Status, &bid.AuthorType, &bid.AuthorID, &bid.TenderID, &bid.Version, &bid.CreatedAt); err != nil {
			err_info = sqlErrToErrInfo(err, 500, ErrMessageServer)
			return nil, err_info
		}
		return &bid, err_info
	}
	err_info = sqlErrToErrInfo(sql.ErrNoRows, 404, ErrMessageBidNotFound)

	return nil, err_info
}

func bidsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err_info ErrorInfo
		log.Println("GettingBids")
		limit, offset, err_info := getLimitOffsetFromRequest(r)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}

		bids, err_info := getBids(db, limit, offset)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bids)
	}
}

func createBid(db *sql.DB, bid *Bid) ErrorInfo {

	var errInfo ErrorInfo
	query := `
		INSERT INTO bids (name, description,, status, author_type, author_id, tender_id, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`
	err := db.QueryRow(query, bid.Name, &bid.Description, bid.Status, bid.AuthorType, bid.AuthorID, bid.TenderID, bid.Version, bid.CreatedAt).Scan(&bid.ID)
	errInfo.status = sqlErrToStatus(err, http.StatusInternalServerError)
	if errInfo.status != 200 {
		errInfo.reason = ErrMessageServer
	}

	return errInfo

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

func newBidHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info ErrorInfo
		err_info.status = 200
		var req CreateBidData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validateNewBid(&req) {
			err_info.reason = ErrMessageWrongRequest
			err_info.status = 400
			sendHttpErr(w, err_info)
			return
		}
		user_name, err_info := getUserName(db, req.AuthorId)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}
		tender, err_info := getTender(db, req.TenderID)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}

		_, err_info = isUserExistAndResponsible(db, user_name, tender.OrganizationID)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}
		bid := createBidDataToBid(req, 1, time.Now())
		err_info = createBid(db, bid)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bid)

	}

}

func getUserBids(db *sql.DB, user_id, limit, offset int) ([]Bid, ErrorInfo) {
	var err_info ErrorInfo
	err_info.reason = ErrMessageServer
	query := `
	SELECT id, name, description,, status,author_type , author_id, tender_id,version, created_at
	FROM bids
	WHERE author_id = $1
	ORDER BY name
	LIMIT $2 OFFSET $3
	`
	rows, err := db.Query(query, user_id, limit, offset)
	err_info.status = sqlErrToStatus(err, 200)
	if err_info.status != 200 {
		return nil, err_info
	}
	defer rows.Close()
	var bids []Bid
	for rows.Next() {
		var bid Bid
		if err := rows.Scan(&bid.ID, &bid.Name, &bid.Description, &bid.Status, &bid.AuthorType, &bid.AuthorID, &bid.TenderID, &bid.Version, &bid.CreatedAt); err != nil {
			err_info.status = sqlErrToStatus(err, 500)
			return nil, err_info
		}
		bids = append(bids, bid)
	}
	err_info.status = sqlErrToStatus(rows.Err(), 500)
	return bids, err_info

}

func myBidsHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		var err_info ErrorInfo

		w.Header().Set("Content-Type", "application/json")

		limit, offset, err_info := getLimitOffsetFromRequest(r)
		var bids []Bid
		var user_id int
		if err_info.status == 200 {
			user_name := getUserNameFromRequest(r)
			user_id, err_info = getUserId(db, user_name)

		}
		if err_info.status == 200 {
			bids, err_info = getUserBids(db, user_id, limit, offset)
		}

		if err_info.status != 200 {
			sendHttpErr(w, err_info)

		} else {

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bids)
		}
	}
}

func getTenderBids(db *sql.DB, tender_id, limit, offset int) ([]Bid, ErrorInfo) {
	var err_info ErrorInfo
	err_info.reason = ErrMessageServer
	query := `
	SELECT id, name,description, status,author_type , author_id, tender_id,version, created_at
	FROM bids
	WHERE tender_id = $1
	ORDER BY name
	LIMIT $2 OFFSET $3
	`
	rows, err := db.Query(query, tender_id, limit, offset)
	err_info.status = sqlErrToStatus(err, 200)
	if err_info.status != 200 {
		return nil, err_info
	}
	defer rows.Close()
	var bids []Bid
	for rows.Next() {
		var bid Bid
		if err := rows.Scan(&bid.ID, &bid.Name, &bid.Description, &bid.Status, &bid.AuthorType, &bid.AuthorID, &bid.TenderID, &bid.Version, &bid.CreatedAt); err != nil {
			err_info.status = sqlErrToStatus(err, 500)
			return nil, err_info
		}
		bids = append(bids, bid)
	}
	err_info.status = sqlErrToStatus(rows.Err(), 500)
	return bids, err_info

}

func hasUserAccesstoTender(db *sql.DB, user_name string, tender_id int) ErrorInfo {

	user_id, err_info := getUserId(db, user_name)
	if err_info.status != 200 {
		return err_info
	}
	tender, err_info := getTender(db, tender_id)
	if err_info.status != 200 {
		return err_info
	}
	err_info = isUserInOrganization(db, user_id, tender.OrganizationID)
	if err_info.status != 200 {
		return err_info
	}
	return err_info
}

func listBidsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err_info ErrorInfo
		vars := mux.Vars(r)
		s_tender_id := vars["tenderId"]
		user_name := r.URL.Query().Get("username")
		tender_id, err_info := atoi(s_tender_id)
		limit, offset, tmp_err_info := getLimitOffsetFromRequest(r)
		if err_info.status != 200 || tmp_err_info.status != 200 || user_name == "" {
			err_info.reason = ErrMessageWrongRequest
			err_info.status = 400
			sendHttpErr(w, err_info)
			return
		}

		err_info = hasUserAccesstoTender(db, user_name, tender_id)
		log.Println("status ", err_info.status)
		log.Println("reason", err_info.reason)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}
		bids, err_info := getTenderBids(db, tender_id, limit, offset)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bids)

	}
}

func handleGetBidStatus(db *sql.DB, w http.ResponseWriter, r *http.Request, bid_id int) {
	var err_info ErrorInfo
	user_name := r.URL.Query().Get("username")

	bid, err_info := getBid(db, bid_id)
	if err_info.status != 200 {
		sendHttpErr(w, err_info)
		return
	}
	err_info = hasUserAccesstoTender(db, user_name, bid.TenderID)
	if err_info.status != 200 {
		sendHttpErr(w, err_info)
		return
	}

	w.Write([]byte(bid.Status))
}

func updateBidStatus(db *sql.DB, bid_id int, status string) (string, ErrorInfo) {
	var err_info ErrorInfo
	err_info.status = 200
	query := `
		UPDATE bids
		SET status = $1
		WHERE id = $2
		RETURNING status
	`

	var updatedStatus string
	err := db.QueryRow(query, status, bid_id).Scan(&updatedStatus)
	if err != nil {
		err_info.status = sqlErrToStatus(err, 500)
		err_info.reason = ErrMessageServer
		return "", err_info
	}

	return updatedStatus, err_info
}

func handlePutBidStatus(db *sql.DB, w http.ResponseWriter, r *http.Request, bid_id int) {
	var err_info ErrorInfo
	user_name := r.URL.Query().Get("username")
	new_status := r.URL.Query().Get("status")
	if user_name == "" || !isNewStatusOk(new_status) {
		err_info.status = 400
		err_info.reason = ErrMessageWrongRequest
		sendHttpErr(w, err_info)
		return
	}

	bid, err_info := getBid(db, bid_id)
	if err_info.status != 200 {
		sendHttpErr(w, err_info)
		return
	}
	err_info = hasUserAccesstoTender(db, user_name, bid.TenderID)
	if err_info.status != 200 {
		sendHttpErr(w, err_info)
		return
	}

	new_status, err_info = updateBidStatus(db, bid_id, new_status)
	if err_info.status != 200 {
		sendHttpErr(w, err_info)
		return
	}
	bid.Status = new_status
	json.NewEncoder(w).Encode(bid)
}

func statusBidsHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info ErrorInfo
		err_info.status = 200
		vars := mux.Vars(r)
		s_bid_id := vars["bidId"]
		w.Header().Set("Content-Type", "application/json")
		bid_id, err_info := atoi(s_bid_id)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}

		log.Println("status handling ", s_bid_id)

		if r.Method == http.MethodGet {
			handleGetTenderStatus(db, w, r, bid_id)

		} else if r.Method == http.MethodPut {
			handlePutTenderStatus(db, w, r, bid_id)
		}
	}
}

func editBid(db *sql.DB, username string, bid *Bid, req_body *editBidRequestBody) ErrorInfo {
	var err_info ErrorInfo
	err_info.status = 200

	err_info = archiveBid(db, bid)
	if err_info.status != 200 {
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

func archiveBid(db *sql.DB, bid *Bid) ErrorInfo {
	var err_info ErrorInfo
	query := `
		INSERT INTO bids_archive (id,name, description, version)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	err := db.QueryRow(query, bid.ID, bid.Name, bid.Description, bid.Version).Scan(&bid.ID)
	err_info.status = sqlErrToStatus(err, http.StatusInternalServerError)
	if err_info.status != 200 {
		err_info.reason = ErrMessageServer
	}
	return err_info

}

func updateBid(db *sql.DB, bid *Bid) ErrorInfo {
	var err_info ErrorInfo
	query := `UPDATE bids 
	SET name = $1, description = $2, version = $3
	WHERE id = $4
	`
	_, err := db.Exec(query, bid.Name, bid.Description, bid.Version, bid.ID)
	if err != nil {
		log.Println(err)
	}
	err_info.status = sqlErrToStatus(err, 500)
	err_info.reason = ErrMessageServer
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

func editBidsHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info ErrorInfo
		vars := mux.Vars(r)
		s_bid_id := vars["bidId"]
		user_name := r.URL.Query().Get("username")

		var req_body editBidRequestBody
		bid_id, err_info := atoi(s_bid_id)
		if err := json.NewDecoder(r.Body).Decode(&req_body); err != nil || err_info.status != 200 || !validateEditBidParams(&req_body) {
			err_info.reason = ErrMessageWrongRequest
			err_info.status = 400
			log.Println(err_info.reason)
			sendHttpErr(w, err_info)
			return
		}

		bid, err_info := getBid(db, bid_id)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}

		err_info = hasUserAccesstoTender(db, user_name, bid.TenderID)
		if err_info.status != 200 {

			sendHttpErr(w, err_info)
			return
		}
		err_info = editBid(db, user_name, bid, &req_body)
		if err_info.status != 200 {

			sendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bid)
	}
}

func getArchivedBid(db *sql.DB, current_bid *Bid, version int) (bid *Bid, err_info ErrorInfo) {
	err_info.status = 200
	err_info.reason = ErrMessageServer
	*bid = *current_bid
	query := `
    SELECT t.name, t.description
    FROM bids_archive t
    WHERE t.id = $1 AND t.version = $2
    `

	rows, err := db.Query(query, bid.ID, version)
	if err != nil {
		err_info.status = sqlErrToStatus(err, 500)
		return
	}
	defer rows.Close()

	if rows.Next() {
		err := rows.Scan(&bid.Name, &bid.Description)
		bid.Version++
		if err != nil {
			err_info.status = sqlErrToStatus(err, 500)
			return
		}
		return
	}
	err_info.status = 404
	err_info.reason = "This version of bid does not exist."
	return
}

func rollbackBid(db *sql.DB, current_bid, old_bid *Bid) ErrorInfo {
	err_info := archiveBid(db, current_bid)
	if err_info.status != 200 {
		return err_info
	}
	old_bid.Version++
	err_info = updateBid(db, old_bid)
	return err_info
}

func rollbackBidsHandler(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var err_info ErrorInfo
		err_info.status = 200
		vars := mux.Vars(r)
		s_bid_id := vars["bidId"]
		s_version := vars["version"]
		version, err_info := atoi(s_version)
		bid_id, tmp_err_info := atoi(s_bid_id)

		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}
		if tmp_err_info.status != 200 {
			sendHttpErr(w, tmp_err_info)
			return
		}
		current_bid, err_info := getBid(db, bid_id)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}

		user_name := r.URL.Query().Get("username")
		err_info = hasUserAccesstoTender(db, user_name, current_bid.TenderID)
		old_bid := &Bid{}
		*old_bid = *current_bid
		old_bid, err_info = getArchivedBid(db, current_bid, version)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}
		err_info = rollbackBid(db, current_bid, old_bid)
		if err_info.status != 200 {
			sendHttpErr(w, err_info)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(old_bid)
	}
}
