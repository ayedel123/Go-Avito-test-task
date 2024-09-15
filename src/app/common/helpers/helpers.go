package helpers

import (
	"go_server/m/common/errinfo"
	"net/http"
	"strconv"
)

func IsNewStatusOk(status string) bool {
	return (status != "" && (status == "Created" || status == "Published" || status == "Closed"))
}

func Atoi(s string) (num int, err_info errinfo.ErrorInfo) {
	num = 0
	num, err := strconv.Atoi(s)
	if err == nil && num >= 0 {
		err_info.Status = 200
		return
	}
	err_info.Status = 400
	err_info.Reason = "Parametr must be positive number."
	return
}

func GetLimitOffsetFromRequest(r *http.Request) (limit int, offset int, err_info errinfo.ErrorInfo) {
	s_limit := r.URL.Query().Get("limit")
	s_offset := r.URL.Query().Get("offset")

	//var limit, offset int

	if s_limit == "" {
		s_limit = "5"
	}

	if s_offset == "" {
		s_offset = "0"
	}
	err_info.Status = 200
	limit, err_info = Atoi(s_limit)
	if err_info.Status == 200 {
		offset, err_info = Atoi(s_offset)
	}
	return

}
