package helpers

import (
	"go_server/m/common/errinfo"
	"net/http"
	"strconv"

	"github.com/google/uuid"
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

func IsOkServiceType(service_type string) bool {
	return (service_type == "" || (service_type == "Construction" || service_type == "Delivery" || service_type == "Manufacture"))
}

func ParseUUID(s_id string) (uuid.UUID, errinfo.ErrorInfo) {
	id, err := uuid.Parse(s_id)
	var err_info errinfo.ErrorInfo
	err_info.Init(200, "Ok")
	if err != nil {
		err_info.Init(400, errinfo.ErrMessageWrongRequest)
	}

	return id, err_info
}
