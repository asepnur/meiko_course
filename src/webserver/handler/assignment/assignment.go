package assignment

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/asepnur/meiko_course/src/util/conn"

	asg "github.com/asepnur/meiko_course/src/module/assignment"
	att "github.com/asepnur/meiko_course/src/module/attendance"
	cs "github.com/asepnur/meiko_course/src/module/course"
	fl "github.com/asepnur/meiko_course/src/module/file"
	usr "github.com/asepnur/meiko_course/src/module/user"
	"github.com/asepnur/meiko_course/src/util/auth"
	"github.com/asepnur/meiko_course/src/util/helper"
	"github.com/asepnur/meiko_course/src/webserver/template"
	"github.com/julienschmidt/httprouter"
)

// GetHandler ...
func GetHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	resp := []getResponse{}
	sess := r.Context().Value("User").(*auth.User)

	params := getParams{
		scheduleID: r.FormValue("schedule_id"),
		filter:     r.FormValue("filter"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	var schedulesID []int64
	switch args.scheduleID.Valid {
	case true:
		// specific scheduleID
		if !cs.IsEnrolled(sess.ID, args.scheduleID.Int64) {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusNoContent))
			return
		}
		schedulesID = []int64{args.scheduleID.Int64}
	case false:
		// all enrolled schedule
		schedulesID, err = cs.SelectScheduleIDByUserID(sess.ID, cs.PStatusStudent)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		// return if empty
		if len(schedulesID) < 1 {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusOK).
				SetData(resp))
			return
		}
	}

	gps, err := cs.SelectGPBySchedule(schedulesID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	if len(gps) < 1 {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusOK).
			SetData(resp))
		return
	}

	var gpsID []int64
	for _, val := range gps {
		gpsID = append(gpsID, val.ID)
	}

	assignments, err := asg.SelectByGP(gpsID, true)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	var asgID []int64
	for _, val := range assignments {
		asgID = append(asgID, val.ID)
	}

	if len(asgID) < 1 {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusOK).
			SetData(resp))
		return
	}

	submitted, err := asg.SelectSubmittedByUser(asgID, sess.ID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	submitMap := map[int64]asg.UserAssignment{}
	for _, val := range submitted {
		submitMap[val.AssignmentID] = val
	}

	var desc, score, status string
	var isAllowUpload bool
	for _, assignment := range assignments {
		submit, exist := submitMap[assignment.ID]
		desc = "-"
		score = "-"
		status = "unsubmitted"
		isAllowUpload = true
		if assignment.DueDate.Before(time.Now()) {
			status = "overdue"
			isAllowUpload = false
		}
		if exist {
			status = "submitted"
			if submit.Score.Valid {
				status = "done"
				isAllowUpload = false
				score = fmt.Sprintf("%.3g", submit.Score.Float64)
			}
		}
		if assignment.Status == asg.StatusUploadNotRequired {
			isAllowUpload = false
		}
		if assignment.Description.Valid {
			desc = assignment.Description.String
		}
		if args.filter.Valid {
			if args.filter.String != status {
				continue
			}
		}
		resp = append(resp, getResponse{
			ID:            assignment.ID,
			DueDate:       assignment.DueDate.Format("Monday, 2 January 2006 15:04:05"),
			Name:          assignment.Name,
			Score:         score,
			Status:        status,
			IsAllowUpload: isAllowUpload,
			Description:   desc,
			UpdatedAt:     assignment.UpdatedAt.Format("Monday, 2 January 2006 15:04:05"),
		})
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(resp))
	return
}

// GetDetailHandler ...
func GetDetailHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)
	params := getDetailParams{
		id: ps.ByName("id"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	assignment, err := asg.GetByID(args.id)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusNoContent))
		return
	}

	scheduleID, err := cs.GetScheduleIDByGP(assignment.GradeParameterID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	if !cs.IsEnrolled(sess.ID, scheduleID) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusNoContent))
		return
	}

	submitted, err := asg.GetSubmittedByUser(args.id, sess.ID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	// response data preparation
	status := "unsubmitted"
	score := "-"
	submittedDate := "-"
	submittedDesc := ""
	isAllowUpload := true
	if assignment.DueDate.Before(time.Now()) {
		status = "overdue"
		isAllowUpload = false
	}
	if submitted != nil {
		status = "submitted"
		submittedDesc = submitted.Description.String
		submittedDate = submitted.UpdatedAt.Format("Monday, 2 January 2006 15:04:05")
		if submitted.Score.Valid {
			isAllowUpload = false
			status = "done"
			score = fmt.Sprintf("%.3g", submitted.Score.Float64)
		}
	}
	if assignment.Status == asg.StatusUploadNotRequired {
		status = "notrequired"
		isAllowUpload = false
	}

	tableID := []string{strconv.FormatInt(args.id, 10)}
	asgFile, err := fl.SelectByRelation(fl.TypAssignment, tableID, nil)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	submittedFile, err := fl.SelectByRelation(fl.TypAssignmentUpload, tableID, &sess.ID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	// file from assistant
	rAsgFile := []file{}
	for _, val := range asgFile {
		rAsgFile = append(rAsgFile, file{
			ID:           val.ID,
			Name:         fmt.Sprintf("%s.%s", val.Name, val.Extension),
			URL:          fmt.Sprintf("/api/v1/file/assignment/%s.%s", val.ID, val.Extension),
			URLThumbnail: helper.MimeToThumbnail(val.Mime),
		})
	}

	// file from student
	rSubmittedFile := []file{}
	for _, val := range submittedFile {
		rSubmittedFile = append(rSubmittedFile, file{
			ID:           val.ID,
			Name:         fmt.Sprintf("%s.%s", val.Name, val.Extension),
			URL:          fmt.Sprintf("/api/v1/file/assignment/%s.%s", val.ID, val.Extension),
			URLThumbnail: helper.MimeToThumbnail(val.Mime),
		})
	}

	resp := getDetailResponse{
		ID:                   assignment.ID,
		Name:                 assignment.Name,
		Status:               status,
		Description:          assignment.Description.String,
		DueDate:              assignment.DueDate.Format("Monday, 2 January 2006 15:04:05"),
		Score:                score,
		CreatedAt:            assignment.CreatedAt.Format("Monday, 2 January 2006"),
		UpdatedAt:            assignment.UpdatedAt.Format("Monday, 2 January 2006"),
		AssignmentFile:       rAsgFile,
		IsAllowUpload:        isAllowUpload,
		SubmittedDescription: submittedDesc,
		SubmittedFile:        rSubmittedFile,
		SubmittedDate:        submittedDate,
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(resp))
	return
}

// SubmitHandler ...
func SubmitHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)

	params := submitParams{
		id:          ps.ByName("id"),
		description: r.FormValue("description"),
		fileID:      r.FormValue("file_id"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError(err.Error()))
		return
	}

	assignment, err := asg.GetByID(args.id)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusNoContent))
		return
	}

	if assignment.Status == asg.StatusUploadNotRequired {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("Invalid Request"))
		return
	}

	if assignment.DueDate.Before(time.Now()) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("Sorry, you can't upload this assignment because of overdue."))
		return
	}

	scheduleID, err := cs.GetScheduleIDByGP(assignment.GradeParameterID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	if !cs.IsEnrolled(sess.ID, scheduleID) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	upload, err := asg.GetSubmittedByUser(args.id, sess.ID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	// update
	if upload != nil {
		err = handleSubmitUpdate(assignment.ID, sess.ID, args.description, args.fileID)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}

		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusOK).
			SetMessage("Success"))
		return
	}

	// insert
	err = handleSubmitInsert(assignment.ID, sess.ID, args.description, args.fileID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetMessage("Success"))
	return
}

// GetAvailableGP ..
func GetAvailableGP(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleAssignment, auth.RoleXCreate, auth.RoleCreate) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}
	params := availableParams{
		id: ps.ByName("id"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	if !cs.IsAssistant(sess.ID, args.id) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}
	gps, err := cs.SelectGPBySchedule([]int64{args.id})
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}
	res := []respGP{}
	for _, gp := range gps {
		res = append(res, respGP{
			ID:   gp.ID,
			Name: gp.Type,
		})
	}
	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).SetData(res))
	return

}

// ReadHandler ..
func ReadHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	resp := readResponse{Assignments: []read{}}
	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleAssignment, auth.RoleXCreate, auth.RoleCreate) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	params := readParams{
		scheduleID: r.FormValue("schedule_id"),
		page:       r.FormValue("pg"),
		total:      r.FormValue("ttl"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	if !cs.IsAssistant(sess.ID, args.scheduleID) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	gps, err := cs.SelectGPBySchedule([]int64{args.scheduleID})
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	if len(gps) < 1 {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusOK).
			SetData(resp))
		return
	}

	var gpsID []int64
	for _, val := range gps {
		gpsID = append(gpsID, val.ID)
	}

	offset := (args.page - 1) * args.total
	assignments, count, err := asg.SelectByPage(gpsID, args.total, offset, true)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	rAssignment := []read{}
	for _, val := range assignments {
		rAssignment = append(rAssignment, read{
			ID:        val.ID,
			Name:      val.Name,
			URL:       fmt.Sprintf("/api/v1/filerouter/?id=%d&payload=assignment&role=assistant", val.ID),
			DueDate:   val.DueDate.Format("Monday, 2 January 2006 15:04:05"),
			UpdatedAt: val.UpdatedAt.Format("Monday, 2 January 2006 15:04:05"),
		})
	}

	totalPage := count / args.total
	if count%args.total > 0 {
		totalPage++
	}

	resp = readResponse{
		Assignments: rAssignment,
		Page:        args.page,
		TotalPage:   totalPage,
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(resp))
	return
}

// CreateHandler ..
func CreateHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleAssignment, auth.RoleXCreate, auth.RoleCreate) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	params := createParams{
		name:             r.FormValue("name"),
		description:      r.FormValue("description"),
		dueDate:          r.FormValue("due_date"),
		filesID:          r.FormValue("files_id"),
		gpID:             r.FormValue("grade_parameter_id"),
		status:           r.FormValue("status"),
		allowedTypesFile: r.FormValue("allowed_types"),
		maxFile:          r.FormValue("max_file"),
		maxSizeFile:      r.FormValue("max_size"),
	}
	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError(err.Error()))
		return
	}
	scheduleID, err := cs.GetScheduleIDByGP(args.gpID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	if !cs.IsAssistant(sess.ID, scheduleID) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}
	if len(args.filesID) > 0 {
		filesID, err := fl.SelectIDStatusByID(args.filesID)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		for _, inFile := range args.filesID {
			for _, dbFile := range filesID {
				if inFile == dbFile.ID && dbFile.Status == fl.StatusDeleted {
					template.RenderJSONResponse(w, new(template.Response).
						SetCode(http.StatusBadRequest).
						AddError("ID file has been deleted, you can not use it again"))
					return
				}
			}
		}
	}
	tx := conn.DB.MustBegin()
	id, err := asg.Insert(args.name, args.description, args.gpID, args.maxSizeFile, args.maxFile, args.dueDate, args.status, tx)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}
	idStr := strconv.FormatInt(id, 10)
	if len(args.filesID) > 0 {
		for _, fileID := range args.filesID {
			if fl.UpdateRelation(fileID, fl.TypAssignment, idStr, tx) != nil {
				tx.Rollback()
				template.RenderJSONResponse(w, new(template.Response).
					SetCode(http.StatusBadRequest).
					AddError("Wrong File ID"))
				return
			}
		}
	}
	if len(args.allowedTypesFile) > 0 {
		err := fl.InsertType(args.allowedTypesFile, id, tx)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusBadRequest).
				AddError("Wrong File ID"))
			return
		}
	}
	tx.Commit()
	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetMessage("Assignment created successfully"))
	return
}

// UpdateHandler ..
func UpdateHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleAssignment, auth.RoleUpdate, auth.RoleXRead) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	params := updateParams{
		ID:               ps.ByName("id"),
		name:             r.FormValue("name"),
		description:      r.FormValue("description"),
		dueDate:          r.FormValue("due_date"),
		filesID:          r.FormValue("files_id"),
		gpID:             r.FormValue("grade_parameter_id"),
		status:           r.FormValue("status"),
		allowedTypesFile: r.FormValue("allowed_types"),
		maxFile:          r.FormValue("max_file"),
		maxSizeFile:      r.FormValue("max_size"),
	}
	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError(err.Error()))
		return
	}
	scheduleID, err := cs.GetScheduleIDByGP(args.gpID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	if !cs.IsAssistant(sess.ID, scheduleID) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}
	tx := conn.DB.MustBegin()
	if len(args.filesID) > 0 {
		filesID, err := fl.SelectIDStatusByID(args.filesID)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		for _, inFile := range args.filesID {
			for _, dbFile := range filesID {
				if inFile == dbFile.ID && dbFile.Status == fl.StatusDeleted {
					template.RenderJSONResponse(w, new(template.Response).
						SetCode(http.StatusBadRequest).
						AddError("ID file has been deleted, you can not use it again"))
					return
				}
			}
		}
		activefilesID, err := fl.SelectIDByRelation(fl.TypAssignment, params.ID, sess.ID)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		var uptFiles []string
		var dltFiles []string
		for _, val := range args.filesID {
			isInActive := 0
			for _, actFile := range activefilesID {
				if actFile == val {
					isInActive++
					break
				}
			}
			if isInActive == 0 {
				uptFiles = append(uptFiles, val)
			}
		}
		for _, actFile := range activefilesID {
			isInActive := 0
			for _, val := range args.filesID {
				if actFile == val {
					isInActive++
					break
				}
			}
			if isInActive == 0 {
				dltFiles = append(dltFiles, actFile)
			}
		}
		if len(uptFiles) > 0 {
			for _, val := range uptFiles {
				err := fl.UpdateRelation(val, fl.TypAssignment, params.ID, tx)
				if err != nil {
					tx.Rollback()
					template.RenderJSONResponse(w, new(template.Response).
						SetCode(http.StatusInternalServerError))
					return
				}
			}
		}
		if len(dltFiles) > 0 {
			for _, val := range dltFiles {
				err := fl.Delete(val, tx)
				if err != nil {
					tx.Rollback()
					template.RenderJSONResponse(w, new(template.Response).
						SetCode(http.StatusInternalServerError))
					return
				}
			}
		}
	}
	if args.status == 0 {
		addedTypes, err := fl.SelectTypeByID(args.ID)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		if len(addedTypes) > 0 {
			err = fl.DeleteTypeByID(addedTypes, args.ID, tx)
			if err != nil {
				tx.Rollback()
				template.RenderJSONResponse(w, new(template.Response).
					SetCode(http.StatusInternalServerError))
				return
			}
		}
	}
	if len(args.allowedTypesFile) > 0 && args.status > 0 {
		addedTypes, err := fl.SelectTypeByID(args.ID)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		var dltType []string
		var uptType []string
		if len(addedTypes) > 0 {
			for _, addedType := range addedTypes {
				count := 0
				for _, alwdType := range args.allowedTypesFile {
					if alwdType == addedType {
						count++
						break
					}
				}
				if count == 0 {
					dltType = append(dltType, addedType)
				}
			}
		}
		for _, alwdType := range args.allowedTypesFile {
			count := 0
			for _, addedType := range addedTypes {
				if alwdType == addedType {
					count++
					break
				}
			}
			if count == 0 {
				uptType = append(uptType, alwdType)
			}
		}
		if len(dltType) > 0 {
			err := fl.DeleteTypeByID(dltType, args.ID, tx)
			if err != nil {
				tx.Rollback()
				template.RenderJSONResponse(w, new(template.Response).
					SetCode(http.StatusInternalServerError))
				return
			}
		}
		if len(uptType) > 0 {
			err := fl.InsertType(uptType, args.ID, tx)
			if err != nil {
				tx.Rollback()
				template.RenderJSONResponse(w, new(template.Response).
					SetCode(http.StatusInternalServerError))
				return
			}
		}
	}

	err = asg.Update(args.name, args.description, args.ID, args.gpID, args.maxSizeFile, args.maxFile, args.dueDate, args.status, tx)
	if err != nil {
		tx.Rollback()
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}
	err = tx.Commit()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}
	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetMessage("Assignment updated successfully"))
	return

}

// DetailHandler ..
func DetailHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleAssignment, auth.RoleRead, auth.RoleXRead) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	params := detailParams{
		ID:      ps.ByName("id"),
		total:   r.FormValue("ttl"),
		page:    r.FormValue("pg"),
		payload: r.FormValue("payload"),
	}
	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError(err.Error()))
		return
	}

	if !asg.IsAssignmentExist(args.ID) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Assignment does not exist"))
		return
	}
	gp := asg.GetGradeParameterID(args.ID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusNotFound))
		return
	}
	scheduleID, err := cs.GetScheduleIDByGP(gp)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	if !cs.IsAssistant(sess.ID, scheduleID) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}
	if args.payload == "update" {
		typs := []string{}
		assignment, err := asg.GetByID(args.ID)
		if err != nil {
			fmt.Println(err)
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		typs, err = fl.SelectTypeByID(args.ID)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		tableID := []string{strconv.FormatInt(args.ID, 10)}
		asgFile, err := fl.SelectByRelation(fl.TypAssignment, tableID, nil)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}

		// file from assistant
		rAsgFile := []file{}
		for _, val := range asgFile {
			rAsgFile = append(rAsgFile, file{
				ID:           val.ID,
				Name:         fmt.Sprintf("%s.%s", val.Name, val.Extension),
				URL:          fmt.Sprintf("/api/v1/file/assignment/%s.%s", val.ID, val.Extension),
				URLThumbnail: helper.MimeToThumbnail(val.Mime),
			})
		}
		desc := "-"
		if assignment.Description.Valid {
			desc = assignment.Description.String
		}
		res := respDetailUpdate{}
		if assignment.Status == 1 {
			var size int8
			var max int8
			if assignment.MaxFile.Valid {
				max = int8(assignment.MaxFile.Int64)
			}
			if assignment.MaxSize.Valid {
				size = int8(assignment.MaxSize.Int64)
			}
			res = respDetailUpdate{
				ID:               assignment.ID,
				Name:             assignment.Name,
				Description:      desc,
				DueDate:          assignment.DueDate,
				GradeParameterID: assignment.GradeParameterID,
				Status:           assignment.Status,
				MaxFile:          max,
				MaxSize:          size,
				Type:             typs,
				FilesID:          rAsgFile,
			}
		} else {
			res = respDetailUpdate{
				ID:               assignment.ID,
				Name:             assignment.Name,
				Description:      desc,
				DueDate:          assignment.DueDate,
				GradeParameterID: assignment.GradeParameterID,
				Status:           assignment.Status,
				FilesID:          rAsgFile,
			}
		}
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusOK).
			SetData(res))
		return

	}
	offset := (args.page - 1) * args.total
	assignment := asg.GetAssignmentByID(args.ID)
	var res respDetAsgUser
	asgUser := []detAsgUser{}
	var totalPg int
	if assignment.Status == 0 {
		ids, total, err := usr.SelectIDByScheduleID(scheduleID, args.total, offset, true)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		totalPg = total / asg.MaxPage
		if total%asg.MaxPage > 0 {
			totalPg++
		}
		users, err := usr.SelectConciseUserByID(ids)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		if len(users) > 0 {
			for _, val := range users {
				asgUser = append(asgUser, detAsgUser{
					ID:          val.IdentityCode,
					Name:        val.Name,
					Description: "-",
					UploadedAt:  "-",
					Link:        "-",
				})
			}
		}
	} else {
		total, err := asg.SelectCountUsrAsgByID(args.ID)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		totalPg = total / asg.MaxPage
		if total%asg.MaxPage > 0 {
			totalPg++
		}
		sbmtdAsg, err := asg.SelectUserAssignmentByID(args.ID, args.total, offset)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		if len(sbmtdAsg) > 0 {
			var usersID []int64
			for _, val := range sbmtdAsg {
				usersID = append(usersID, val.UserID)
			}
			users, err := usr.SelectConciseUserByID(usersID)
			if err != nil {
				template.RenderJSONResponse(w, new(template.Response).
					SetCode(http.StatusInternalServerError))
				return
			}
			for _, val := range sbmtdAsg {
				var desc string
				if val.Description.Valid {
					desc = val.Description.String
				}
				asgUser = append(asgUser, detAsgUser{
					ID:          val.UserID,
					Description: desc,
					UploadedAt:  val.UpdatedAt.Format("Monday, 2 January 2006 15:04:05"),
					Link:        "-",
				})
			}
			for _, a := range asgUser {
				for i, u := range users {
					if a.ID == u.ID {
						asgUser[i].ID = u.IdentityCode
						asgUser[i].Name = u.Name
					}
				}
			}
		}
	}
	status := "must_upload"
	if assignment.Status == 0 {
		status = "upload_not_required"
	}
	res = respDetAsgUser{
		TotalPage:   totalPg,
		CurrentPage: args.page,
		ID:          assignment.ID,
		Name:        assignment.Name,
		Status:      status,
		DueDate:     assignment.DueDate.Format("Monday, 2 January 2006 15:04:05"),
		DetAsgUser:  asgUser,
	}
	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(res))
	return
}

// DeleteHandler ..
func DeleteHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleAssignment, auth.RoleXDelete, auth.RoleDelete) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	params := deleteParams{
		id: ps.ByName("id"),
	}
	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}
	countAsg, err := asg.SelectCountByID(args.id)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusNoContent))
		return
	}
	if countAsg == 0 {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Assignment does not exist"))
		return
	}
	assignment, err := asg.GetByID(args.id)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusNoContent))
		return
	}
	sbmtdAsg, err := asg.SelectCountUsrAsgByID(args.id)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}
	if sbmtdAsg > 0 {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("not allowed to delete this assignments"))
		return
	}
	scheduleID, err := cs.GetScheduleIDByGP(assignment.GradeParameterID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	if !cs.IsAssistant(sess.ID, scheduleID) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}
	tx := conn.DB.MustBegin()
	typs, err := fl.SelectCountTypeByID(args.id)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}
	if typs > 0 {
		err = fl.DeleteAllTypeByID(args.id, tx)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
	}
	id := strconv.FormatInt(args.id, 10)
	fls, err := fl.SelectCountIDByRelation(fl.TypAssignment, id, sess.ID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}
	if fls > 0 {
		err := fl.DeleteByRelation(fl.TypAssignment, id, tx)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
	}

	err = asg.DeleteAssignment(args.id, tx)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	err = tx.Commit()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetMessage("Assignment deleted successfully"))
	return
}

// not finished yet
func GetReportHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	resp := []getReportResponse{}
	sess := r.Context().Value("User").(*auth.User)

	payload := r.FormValue("schedule_id")
	if !helper.IsEmpty(payload) {
		scheduleID, err := strconv.ParseInt(payload, 10, 64)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusBadRequest).
				AddError("Invalid Request"))
			return
		}
		gradeResp, statusCode, err := handleGradeBySchedule(scheduleID, sess.ID)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(statusCode).
				AddError("Invalid Request"))
			return
		}
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusOK).
			SetData(gradeResp))
		return
	}

	schedulesID, err := cs.SelectScheduleIDByUserID(sess.ID, cs.PStatusStudent)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	if len(schedulesID) < 1 {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusOK).
			SetData(resp))
		return
	}

	courses, err := cs.SelectByScheduleID(schedulesID, cs.StatusScheduleActive)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	if len(courses) < 1 {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusOK).
			SetData(resp))
		return
	}

	schedulesID = []int64{}
	for _, val := range courses {
		schedulesID = append(schedulesID, val.Schedule.ID)
	}

	gps, err := cs.SelectGPBySchedule(schedulesID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	if len(gps) < 1 {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusOK).
			SetData(resp))
		return
	}

	var gpsID []int64
	scheduleGP := map[int64][]cs.GradeParameter{}
	for _, gp := range gps {
		gpsID = append(gpsID, gp.ID)
		scheduleGP[gp.ScheduleID] = append(scheduleGP[gp.ScheduleID], gp)
	}

	assignments, err := asg.SelectByGP(gpsID, false)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	var asgID []int64
	gpAsg := map[int64][]asg.Assignment{}
	for _, val := range assignments {
		asgID = append(asgID, val.ID)
		gpAsg[val.GradeParameterID] = append(gpAsg[val.GradeParameterID], val)
	}

	if len(asgID) < 1 {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusOK).
			SetData(resp))
		return
	}

	submitted, err := asg.SelectSubmittedByUser(asgID, sess.ID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	asgSubmit := map[int64]asg.UserAssignment{}
	for _, val := range submitted {
		asgSubmit[val.AssignmentID] = val
	}

	attReport, err := att.CountByUserSchedule(sess.ID, schedulesID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	for _, c := range courses {
		rep := getReportResponse{
			CourseName: c.Course.Name,
			ScheduleID: c.Schedule.ID,
			Assignment: "-",
			Attendance: "-",
			Quiz:       "-",
			Mid:        "-",
			Final:      "-",
			Total:      "-",
		}
		total := float64(0)
		for _, gp := range scheduleGP[c.Schedule.ID] {
			scoreFloat64 := float64(0)
			if gp.Type == "ATTENDANCE" {
				attendance := attReport[gp.ScheduleID]
				if attendance.MeetingTotal > 0 {
					scoreFloat64 = (float64(attendance.AttendanceTotal) / float64(attendance.MeetingTotal)) * float64(gp.Percentage) / 100
				}
				rep.Attendance = fmt.Sprintf("%.3g", scoreFloat64)
			} else {
				count := len(gpAsg[gp.ID])
				for _, assignment := range gpAsg[gp.ID] {
					submit, exist := asgSubmit[assignment.ID]
					if !exist {
						continue
					}
					if submit.Score.Valid {
						scoreFloat64 += submit.Score.Float64
					} else {
						count--
					}
				}
				if count > 0 {
					scoreFloat64 = scoreFloat64 / float64(count)
				}
				total += (scoreFloat64 * float64(gp.Percentage) / 100)
				switch gp.Type {
				case "ASSIGNMENT":
					rep.Assignment = fmt.Sprintf("%.3g", scoreFloat64)
				case "QUIZ":
					rep.Quiz = fmt.Sprintf("%.3g", scoreFloat64)
				case "MID":
					rep.Mid = fmt.Sprintf("%.3g", scoreFloat64)
				case "FINAL":
					rep.Final = fmt.Sprintf("%.3g", scoreFloat64)
				}
			}
		}
		rep.Total = fmt.Sprintf("%.3g", total)
		resp = append(resp, rep)
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(resp))
	return
}
