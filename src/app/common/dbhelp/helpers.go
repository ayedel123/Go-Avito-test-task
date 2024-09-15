package dbhelp

import (
	"database/sql"
	"go_server/m/common/errinfo"
	"net/http"
)

func SqlErrToErrInfo(err error, err_status int, message string) errinfo.ErrorInfo {
	var err_info errinfo.ErrorInfo
	if message != "" {
		err_info.Reason = message
	}
	err_info.Status = SqlErrToStatus(err, err_status)
	return err_info
}

func SqlErrToStatus(err error, err_status int) int {
	status := 200
	if err != nil {
		if err == sql.ErrNoRows {
			status = err_status
		} else {
			status = 500
		}

	}
	return status
}

func GetUserNameFromRequest(r *http.Request) string {
	username := r.URL.Query().Get("username")
	return username
}

func GetUserName(db *sql.DB, user_id int) (user_name string, err_info errinfo.ErrorInfo) {
	user_name = ""
	query := `
        SELECT e.username
        FROM  employee e WHERE e.id = $1
		LIMIT 1
    `
	err := db.QueryRow(query, user_id).Scan(&user_name)

	err_info.Status = SqlErrToStatus(err, 401)
	if err_info.Status != 200 {
		err_info.Reason = "User does not exist."
	}
	return
}

func IsUserExistAndResponsible(db *sql.DB, user_name string, organization_id int) (user_id int, err_info errinfo.ErrorInfo) {
	user_id, err_info = GetUserId(db, user_name)
	if err_info.Status != 200 {
		return
	}
	err_info = IsUserInOrganization(db, user_id, organization_id)
	if err_info.Status != 200 {
		err_info.Status = http.StatusForbidden
		err_info.Reason = errinfo.ErrMessageNoPermission
		return
	}
	return
}

func IsUserInOrganization(db *sql.DB, user_id, organization_id int) errinfo.ErrorInfo {

	query := `
        SELECT orgr.user_id 
        FROM organization_responsible orgr 
        WHERE orgr.user_id = $1 AND orgr.organization_id = $2
		LIMIT 1
    `
	err := db.QueryRow(query, user_id, organization_id).Scan(&user_id)
	var err_info errinfo.ErrorInfo
	err_info.Status = SqlErrToStatus(err, 403)
	err_info.Reason = "User does not have permission."
	return err_info
}

func GetUserId(db *sql.DB, user_name string) (user_id int, err_info errinfo.ErrorInfo) {
	query := `
        SELECT e.id 
        FROM  employee e WHERE e.username = $1
		LIMIT 1
    `
	err := db.QueryRow(query, user_name).Scan(&user_id)

	err_info.Status = SqlErrToStatus(err, 401)
	if err_info.Status != 200 {
		err_info.Reason = "User does not exist."
	}

	return
}
