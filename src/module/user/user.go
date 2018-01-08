package user

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/asepnur/meiko_course/src/util/helper"

	"github.com/asepnur/meiko_course/src/util/conn"
)

// SelectIDByIdentityCode ...
func SelectIDByIdentityCode(identityCode []int64) ([]int64, error) {
	if len(identityCode) > 0 {
		var ic []string
		for _, val := range identityCode {
			ic = append(ic, fmt.Sprintf("%d", val))
		}
		codes := strings.Join(ic, "~")
		data := url.Values{}
		data.Set("identity_code", codes)
		params := data.Encode()
		req, err := http.NewRequest("POST", "http://localhost:9000/api/v1/user/identity-code", strings.NewReader(params))
		if err != nil {
			return nil, err
		}
		req.Header.Add("Authorization", "abc")
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(params)))

		client := http.Client{
			Timeout: time.Second * 2,
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		res := &UserHTTPResponseByIdentityCode{}
		err = json.Unmarshal(body, res)
		if err != nil {
			return nil, err
		}
		if res == nil {
			return nil, fmt.Errorf("Data null")
		}
		usr := res.Data
		return usr, nil

	}
	var ids []int64
	queryIdentity := strings.Join(helper.Int64ToStringSlice(identityCode), ", ")
	query := fmt.Sprintf(`
		SELECT
			id
		FROM
			users
		WHERE
			identity_code IN (%s)
		;`, queryIdentity)
	err := conn.DB.Select(&ids, query)
	if err != nil {
		return ids, err
	}
	return ids, nil

}

// SelectIDByScheduleID ..
func SelectIDByScheduleID(scheduleID int64, limit, offset int, isCount bool) ([]int64, int, error) {
	var total int
	data := url.Values{}
	data.Set("schedule_id", fmt.Sprintf("%d", scheduleID))
	data.Set("limit", fmt.Sprintf("%d", limit))
	data.Set("offset", fmt.Sprintf("%d", offset))
	count := "0"
	if isCount {
		count = "1"
	}
	data.Set("count", count)
	params := data.Encode()
	req, err := http.NewRequest("POST", "http://localhost:9000/api/v1/user/schedule-id", strings.NewReader(params))
	if err != nil {
		return nil, total, err
	}
	req.Header.Add("Authorization", "abc")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(params)))

	client := http.Client{
		Timeout: time.Second * 2,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, total, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, total, nil
	}
	res := &UserHTTPResponseByScheduleID{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, total, err
	}
	if res == nil {
		return nil, total, fmt.Errorf("Data null")
	}
	usr := res.Data.Data
	total = res.Data.Total
	return usr, total, nil

}

// RequestID ..
func RequestID(id []int64, isSort bool, column ...string) ([]UserReq, error) {
	var user []UserReq
	data := url.Values{}
	var ids []string
	for _, val := range id {
		ids = append(ids, fmt.Sprintf("%d", val))
	}
	reqIds := strings.Join(ids, "~")
	data.Set("id", reqIds)
	var sort = "0"
	if isSort {
		sort = "1"
	}
	data.Set("is_sort", sort)
	params := data.Encode()
	req, err := http.NewRequest("POST", "http://localhost:9000/api/v1/user/exhange-id", strings.NewReader(params))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "abc")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(params)))

	client := http.Client{
		Timeout: time.Second * 60,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil
	}
	res := &UserHTTPResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return user, err
	}
	if res == nil {
		return nil, fmt.Errorf("Data nil")
	}
	usr := res.Data
	return usr, nil
}
