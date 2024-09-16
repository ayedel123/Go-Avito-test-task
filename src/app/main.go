package main

import (
	"database/sql"

	"go_server/m/bids"
	_ "go_server/m/common/errinfo"
	"go_server/m/tenders"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	_ "github.com/lib/pq"
)

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func httpSetHandlers(db *sql.DB) {
	r := mux.NewRouter()

	//r.HandleFunc("/api/tenders", tenders.TendersHandler(db)).Methods("GET")
	//r.HandleFunc("/api/archived_tenders", tenders.TendersArchiveHandler(db)).Methods("GET")
	//r.HandleFunc("/api/bids", bids.BidsHandler(db)).Methods("GET")
	//For manual testing

	r.HandleFunc("/api/ping", pingHandler).Methods("GET")

	r.HandleFunc("/api/tenders/new", tenders.NewTenderHandler(db)).Methods("POST")
	r.HandleFunc("/api/tenders/my", tenders.MyTendersHandler(db)).Methods("GET")

	r.HandleFunc("/api/tenders/{tenderId}/status", tenders.StatusTendersHandler(db)).Methods("GET", "PUT")
	r.HandleFunc("/api/tenders/{tenderId}/edit", tenders.EditTendersHandler(db)).Methods("PATCH")
	r.HandleFunc("/api/tenders/{tenderId}/rollback/{version}", tenders.RollbackTendersHandler(db)).Methods("PUT")

	r.HandleFunc("/api/bids/new", bids.NewBidHandler(db)).Methods("POST")
	r.HandleFunc("/api/bids/my", bids.MyBidsHandler(db)).Methods("GET")

	r.HandleFunc("/api/bids/{tenderId}/list", bids.ListBidsHandler(db)).Methods("GET")
	r.HandleFunc("/api/bids/{bidId}/status", bids.StatusBidsHandler(db)).Methods("GET", "PUT")
	r.HandleFunc("/api/bids/{bidId}/edit", bids.EditBidsHandler(db)).Methods("PATCH")
	r.HandleFunc("/api/bids/{bidId}/rollback/{version}", bids.RollbackBidsHandler(db)).Methods("PUT")
	r.HandleFunc("/api/bids/{bidId}/feedback", bids.FeedbackHandler(db)).Methods("PUT")
	r.HandleFunc("/api/bids/{tenderId}/reviews", bids.ReviewsHandler(db)).Methods("GET")
	r.HandleFunc("/api/bids/{bidId}/submit_decision", bids.SubmitDecisionHandler(db)).Methods("PUT")

	http.Handle("/", r)
}

func main() {
	connStr := os.Getenv("POSTGRES_CONN")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	httpSetHandlers(db)
	log.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
