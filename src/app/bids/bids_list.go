package bids

import (
	"database/sql"
	"encoding/json"
	"go_server/m/common/dbhelp"
	"go_server/m/common/errinfo"
	"go_server/m/common/helpers"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func GetTenderBids(db *sql.DB, tender_id uuid.UUID, limit, offset int) ([]Bid, errinfo.ErrorInfo) {
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

func ListBidsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err_info errinfo.ErrorInfo
		vars := mux.Vars(r)
		s_tender_id := vars["tenderId"]
		user_name := r.URL.Query().Get("username")
		tender_id, err_info := helpers.ParseUUID(s_tender_id)
		limit, offset, tmp_err_info := helpers.GetLimitOffsetFromRequest(r)
		if err_info.Status != 200 || tmp_err_info.Status != 200 || user_name == "" {
			err_info.Init(400, errinfo.ErrMessageWrongRequest)
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
