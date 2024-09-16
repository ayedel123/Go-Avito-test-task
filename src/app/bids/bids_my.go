package bids

import (
	"database/sql"
	"encoding/json"
	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"go_server/m/common/helpers"
	"net/http"
)

func getUserBids(db *sql.DB, user_id, limit, offset int) ([]Bid, errinfo.ErrorInfo) {
	var err_info errinfo.ErrorInfo
	err_info.Reason = errinfo.ErrMessageServer
	query := `
	SELECT id, name, description, status, author_type, author_id, tender_id, version, created_at
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
