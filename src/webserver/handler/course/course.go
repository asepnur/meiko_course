package course

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/asepnur/meiko_course/src/module/user"
	"github.com/asepnur/meiko_course/src/util/conn"

	"github.com/asepnur/meiko_course/src/util/helper"

	ag "github.com/asepnur/meiko_course/src/module/assignment"
	cs "github.com/asepnur/meiko_course/src/module/course"
	fl "github.com/asepnur/meiko_course/src/module/file"
	pl "github.com/asepnur/meiko_course/src/module/place"
	usr "github.com/asepnur/meiko_course/src/module/user"

	"github.com/asepnur/meiko_course/src/util/auth"
	"github.com/asepnur/meiko_course/src/webserver/template"
	"github.com/julienschmidt/httprouter"
)

// CreateHandler handles the http request for creating the course. Accessing this handler needs CREATE or XCREATE ability
/*
	@params:
		id			= required
		name		= required, alphabet and space only
		description	= optional
		ucu			= required, positive numeric
		semester	= required, positive numeric
		start_time	= required, positive numeric, minutes
		end_time	= required, positive numeric, minutes
		class		= required, character=1
		day			= required, [monday, tuesday, wednesday, thursday, friday, saturday, sunday]
		place		= required
		is_update	= optional, true
	@example:
		id			= D10K-7D02
		name		= Sistem Informasi Multimedia
		description	= Praktikum ini membahas mengenai Sistem Informasi Multimedia
		ucu			= 3
		semester	= 1
		start_time	= 600
		end_time	= 800
		class		= A
		day			= monday
		place		= UDJT-102
		is_update	= true
	@return
*/
func CreateHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleSchedule, auth.RoleCreate, auth.RoleXCreate) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	params := createParams{
		ID:             r.FormValue("id"),
		Name:           r.FormValue("name"),
		Description:    r.FormValue("description"),
		UCU:            r.FormValue("ucu"),
		Semester:       r.FormValue("semester"),
		Year:           r.FormValue("year"),
		StartTime:      r.FormValue("start_time"),
		EndTime:        r.FormValue("end_time"),
		Class:          r.FormValue("class"),
		Day:            r.FormValue("day"),
		PlaceID:        r.FormValue("place"),
		IsUpdate:       r.FormValue("is_update"),
		GradeParameter: r.FormValue("grade_parameter"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError(err.Error()))
		return
	}

	// is exist course and place
	scExist := cs.IsExistSchedule(args.Semester, args.Year, args.ID, args.Class)
	if scExist {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Schedule already exists"))
		return
	}
	csExist := cs.IsExist(args.ID)
	plExist := pl.IsExistID(args.PlaceID)

	tx := conn.DB.MustBegin()

	// insert new course and check create or xcreate roles
	if !csExist && sess.IsHasRoles(auth.ModuleCourse, auth.RoleCreate, auth.RoleXCreate) {
		err = cs.Insert(args.ID, args.Name, args.Description, args.UCU, tx)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		// update course and check update or xupdate roles
	} else if csExist && args.IsUpdate && sess.IsHasRoles(auth.ModuleCourse, auth.RoleUpdate, auth.RoleXUpdate) {
		err = cs.Update(args.ID, args.Name, args.Description, args.UCU, tx)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		// want to update course but dont have privilege
	} else if args.IsUpdate {
		tx.Rollback()
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You dont have privilege to create or update course"))
		return
	}

	// insert place
	if !plExist {
		err = pl.Insert(args.PlaceID, sql.NullString{}, tx)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
	}

	// insert schedule
	scheduleID, err := cs.InsertSchedule(sess.ID,
		args.StartTime,
		args.EndTime,
		args.Year,
		args.Semester,
		args.Day,
		cs.StatusScheduleActive,
		args.Class,
		args.ID,
		args.PlaceID,
		tx)
	if err != nil {
		tx.Rollback()
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	// set grade parameter
	if len(args.GradeParameter) > 0 {
		for _, val := range args.GradeParameter {
			err = cs.InsertGradeParameter(val.Type, val.Percentage, val.StatusChange, scheduleID, tx)
			if err != nil {
				tx.Rollback()
				template.RenderJSONResponse(w, new(template.Response).
					SetCode(http.StatusInternalServerError))
				return
			}
		}
	}

	err = tx.Commit()
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

// ReadHandler handles the http request for creating the course. Accessing this handler needs CREATE or XCREATE ability
/*
	@params:
		pg	= required, positive numeric
		ttl	= required, positive numeric
	@example:
		pg=1
		ttl=10
	@return
		[]{name, email, status, identity}
*/
func ReadHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleCourse, auth.RoleRead, auth.RoleXRead) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	params := readParams{
		page:  r.FormValue("pg"),
		total: r.FormValue("ttl"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("Invalid request"))
		return
	}

	if args.total > 100 {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Max total should be less than or equal to 100"))
		return
	}

	offset := (args.page - 1) * args.total
	courses, count, err := cs.SelectByPage(args.total, offset, true)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	var status string
	respCourses := []readCourse{}
	for _, val := range courses {

		if val.Schedule.Status == cs.StatusScheduleActive {
			status = "active"
		} else {
			status = "inactive"
		}

		respCourses = append(respCourses, readCourse{
			ID:         val.Course.ID,
			Name:       val.Course.Name,
			Class:      val.Schedule.Class,
			StartTime:  helper.MinutesToTimeString(val.Schedule.StartTime),
			EndTime:    helper.MinutesToTimeString(val.Schedule.EndTime),
			Day:        helper.IntDayToString(val.Schedule.Day),
			Status:     status,
			ScheduleID: val.Schedule.ID,
		})
	}

	totalPage := count / args.total
	if count%args.total > 0 {
		totalPage++
	}

	resp := readResponse{
		TotalPage: totalPage,
		Page:      args.page,
		Courses:   respCourses,
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(resp))
	return
}

func SearchHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleCourse, auth.RoleRead, auth.RoleXRead) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	params := searchParams{
		Text: r.FormValue("q"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("Invalid request"))
		return
	}

	var resp []searchResponse
	if len(args.Text) < 3 || len(args.Text) > 20 {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusOK).
			SetData(resp))
		return
	}

	courses, err := cs.SelectByName(args.Text)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	for _, val := range courses {
		resp = append(resp, searchResponse{
			ID:          val.ID,
			Name:        val.Name,
			Description: val.Description.String,
			UCU:         val.UCU,
		})
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(resp))
	return
}

// ReadDetailHandler handles the http request for read course details. Accessing this handler needs READ or XREAD ability
/*
	@params:
		schedule_id	= required, positive numeric
	@example:
		schedule_id = 123
	@return
		id = D10K-7D01
		name = Mobile Computing
		description = Mata kuliah ini mengajarkan tentang aplikasi client server
		ucu = 3
		status = 1
		semester = 7
		year = 2017
		start_time = 600
		end_time = 800
		class = A
		day = thursday
		place_id = UDJT102
		schedule_id = 123
*/
func ReadDetailHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleCourse, auth.RoleRead, auth.RoleXRead) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	params := readDetailParams{
		ScheduleID: ps.ByName("schedule_id"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	course, err := cs.GetByScheduleID(args.ScheduleID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	status := "active"
	if course.Schedule.Status == cs.StatusScheduleInactive {
		status = "inactive"
	}

	resp := readDetailResponse{
		ID:          course.Course.ID,
		Name:        course.Course.Name,
		Description: course.Course.Description.String,
		UCU:         course.Course.UCU,
		Status:      status,
		Semester:    course.Schedule.Semester,
		Year:        course.Schedule.Year,
		StartTime:   course.Schedule.StartTime,
		EndTime:     course.Schedule.EndTime,
		Class:       course.Schedule.Class,
		Day:         helper.IntDayToString(course.Schedule.Day),
		PlaceID:     course.Schedule.PlaceID,
		ScheduleID:  course.Schedule.ID,
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(resp))
	return
}

func UpdateHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleSchedule, auth.RoleUpdate, auth.RoleXUpdate) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	params := updateParams{
		ID:             r.FormValue("id"),
		Name:           r.FormValue("name"),
		Description:    r.FormValue("description"),
		UCU:            r.FormValue("ucu"),
		ScheduleID:     ps.ByName("schedule_id"),
		Status:         r.FormValue("status"),
		Semester:       r.FormValue("semester"),
		Year:           r.FormValue("year"),
		StartTime:      r.FormValue("start_time"),
		EndTime:        r.FormValue("end_time"),
		Class:          r.FormValue("class"),
		Day:            r.FormValue("day"),
		PlaceID:        r.FormValue("place"),
		IsUpdate:       r.FormValue("is_update"),
		GradeParameter: r.FormValue("grade_parameter"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError(err.Error()))
		return
	}

	// check if semester, year, id, class already used by another schedule
	if cs.IsExistSchedule(args.Semester, args.Year, args.ID, args.Class, args.ScheduleID) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusConflict).
			AddError("Schedule already exist"))
		return
	}

	// get old grade parameter
	gpsOld, err := cs.SelectGPBySchedule([]int64{args.ScheduleID})
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	var gpsTypeNew []string
	var gpsTypeOld []string
	var gpsInsert []gradeParameter
	var gpsUpdate []gradeParameter
	var gpsDelete []cs.GradeParameter

	// new requested parameter from database
	for _, val := range args.GradeParameter {
		gpsTypeNew = append(gpsTypeNew, val.Type)
	}

	// old grade parameter from database
	for _, val := range gpsOld {
		gpsTypeOld = append(gpsTypeOld, val.Type)
	}

	// get insert and update grade parameter
	for _, val := range args.GradeParameter {
		if helper.IsStringInSlice(val.Type, gpsTypeOld) {
			gpsUpdate = append(gpsUpdate, val)
			continue
		}
		gpsInsert = append(gpsInsert, val)
	}

	// get delete grade parameter type
	for _, val := range gpsOld {
		if !helper.IsStringInSlice(val.Type, gpsTypeNew) {
			gpsDelete = append(gpsDelete, val)
		}
	}

	// check deleted dependent assignment
	var dependent []string
	for _, val := range gpsDelete {
		if ag.IsExistByGradeParameterID(val.ID) {
			dependent = append(dependent, val.Type)
		}
	}

	if len(dependent) > 0 {
		msg := fmt.Sprintf("Delete failed: Some assignment dependent to %s", strings.Join(dependent, ", "))
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusConflict).
			AddError(msg))
		return
	}

	// is exist course and place
	scExist := cs.IsExistScheduleID(args.ScheduleID)
	if !scExist {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusNotFound))
		return
	}
	csExist := cs.IsExist(args.ID)
	plExist := pl.IsExistID(args.PlaceID)

	tx := conn.DB.MustBegin()

	// insert new course and check create or xcreate roles
	if !csExist && sess.IsHasRoles(auth.ModuleCourse, auth.RoleCreate, auth.RoleXCreate) {
		err = cs.Insert(args.ID, args.Name, args.Description, args.UCU, tx)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		// update course and check update or xupdate roles
	} else if csExist && args.IsUpdate && sess.IsHasRoles(auth.ModuleCourse, auth.RoleUpdate, auth.RoleXUpdate) {
		err = cs.Update(args.ID, args.Name, args.Description, args.UCU, tx)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
		// want to update course but dont have privilege
	} else if args.IsUpdate {
		tx.Rollback()
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You dont have privilege to create or update course"))
		return
	}

	// insert place if not exist
	if !plExist {
		err = pl.Insert(args.PlaceID, sql.NullString{}, tx)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
	}

	// update schedule
	err = cs.UpdateSchedule(args.ScheduleID, args.StartTime, args.EndTime, args.Year, args.Semester, args.Day, args.Status, args.Class, args.ID, args.PlaceID, tx)
	if err != nil {
		tx.Rollback()
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	// delete old grade parameter
	for _, val := range gpsDelete {
		err := cs.DeleteGradeParameter(val.ID, tx)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
	}

	// update the parameter
	for _, val := range gpsUpdate {
		err := cs.UpdateGradeParameter(val.Type, val.Percentage, val.StatusChange, args.ScheduleID, tx)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
	}

	// insert the parameter
	for _, val := range gpsInsert {
		err := cs.InsertGradeParameter(val.Type, val.Percentage, val.StatusChange, args.ScheduleID, tx)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
	}

	err = tx.Commit()
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

// GetHandler handles the http request return course list
/*
	@params:
		payload	= required
	@example:
		pg	= last or current or all
		ttl = 10
	@return
		[]{id, name, description}
*/
func GetHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)

	params := getParams{
		Payload: r.FormValue("payload"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	resp := []getResponse{}
	switch args.Payload {
	case "last":
		resp, err = getLast(sess.ID)
	case "current":
		resp, err = getCurrent(sess.ID)
	case "all":
		resp, err = getAll(sess.ID)
	default:
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(resp))
	return
}

// GetDetailHandler handles the http request return course list
/*
	@params:
		schedule_id	= required, numeric
	@example:
		schedule_id = 149
	@return
		[]{id, name, description}
*/
func GetDetailHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	if ps.ByName("schedule_id") == "today" {
		GetTodayHandler(w, r, ps)
		return
	}

	sess := r.Context().Value("User").(*auth.User)

	params := getDetailParams{
		scheduleID: ps.ByName("schedule_id"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	if !cs.IsEnrolled(sess.ID, args.scheduleID) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	c, err := cs.GetByScheduleID(args.scheduleID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	resp := getDetailResponse{
		ID:          c.Schedule.ID,
		Name:        c.Course.Name,
		Description: c.Course.Description.String,
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(resp))
	return
}

// GetAssistantHandler handles the http request return course assistant list
/*
	@params:
		payload	= required
	@example:
		pg	= last or current or all
		ttl = 10
	@return
		[]{id, name, description}
*/
func GetAssistantHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)

	params := getAssistantParams{
		payload:    r.FormValue("payload"),
		scheduleID: ps.ByName("schedule_id"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	isHasAccess := false
	switch args.payload {
	case "assistant":
		isHasAccess = cs.IsAssistant(sess.ID, args.scheduleID) &&
			sess.IsHasRoles(auth.ModuleCourse, auth.RoleXRead, auth.RoleRead)
	case "student":
		isHasAccess = cs.IsEnrolled(sess.ID, args.scheduleID)
	}

	if !isHasAccess {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You are not authorized"))
		return
	}

	uIDs, err := cs.SelectAssistantID(args.scheduleID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	var users []usr.UserReq
	if len(uIDs) > 0 {

		users, err = usr.RequestID(uIDs, false)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
	}

	tableID := helper.Int64ToStringSlice(uIDs)
	thumbs, err := fl.SelectByRelation(fl.TypProfPictThumb, tableID, nil)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	tImg := map[string]fl.File{}
	for _, val := range thumbs {
		if val.TableID.Valid {
			tImg[val.TableID.String] = val
		}
	}

	var thumb string
	res := []getAssistantResponse{}
	for _, val := range users {
		thumb = fl.UsrNoPhotoURL
		if v, ok := tImg[strconv.FormatInt(val.ID, 10)]; ok {
			thumb = fmt.Sprintf("/api/v1/file/profile/%s.%s", v.ID, v.Extension)
		}

		res = append(res, getAssistantResponse{
			Name:         val.Name,
			Email:        val.Email,
			Phone:        val.Phone,
			Roles:        "Assistant",
			URLThumbnail: thumb,
		})
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(res))
	return
}

// DeleteScheduleHandler handles the http request for deleting schedule. Accessing this handler require DELETE or XDELETE ability
/*
	@params:
		schedule_id = required, postitive numeric
	@example:
		schedule_id = 149
	@return
*/
func DeleteScheduleHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleSchedule, auth.RoleDelete, auth.RoleXDelete) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	params := deleteScheduleParams{
		ScheduleID: ps.ByName("schedule_id"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	if !cs.IsExistScheduleID(args.ScheduleID) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusNotFound).
			AddError("Not Found"))
		return
	}

	err = cs.DeleteSchedule(args.ScheduleID)
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

// ListParameterHandler handles the http request returns list of grade parameters. Accessing this handler require READ or XREAD ability
/*
	@params:
		schedule_id = required, postitive numeric
	@example:
		schedule_id = 149
	@return
*/
func ListParameterHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleSchedule, auth.RoleRead, auth.RoleXRead) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	resp := []listParameterResponse{
		{
			Name: cs.GradeParameterAttendance,
			Text: "Attendance",
		},
		{
			Name: cs.GradeParameterQuiz,
			Text: "Quiz",
		},
		{
			Name: cs.GradeParameterMid,
			Text: "UTS",
		},
		{
			Name: cs.GradeParameterFinal,
			Text: "UAS",
		},
		{
			Name: cs.GradeParameterAssignment,
			Text: "Assignment",
		},
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(resp))
	return
}

// ReadScheduleParameterHandler handles the http request returns list of grade parameters to the specific schedules_id. Accessing this handler require READ or XREAD ability
/*
	@params:
		schedule_id = required, postitive numeric
	@example:
		schedule_id = 149
	@return
*/
func ReadScheduleParameterHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleSchedule, auth.RoleRead, auth.RoleXRead) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	params := readScheduleParameterParams{
		ScheduleID: ps.ByName("schedule_id"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	gps, err := cs.SelectGPBySchedule([]int64{args.ScheduleID})
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	var resp []readScheduleParameterResponse
	for _, val := range gps {
		resp = append(resp, readScheduleParameterResponse{
			Type:         val.Type,
			Percentage:   val.Percentage,
			StatusChange: val.StatusChange,
		})
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(resp))
	return
}

func ListEnrolledHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleSchedule, auth.RoleRead, auth.RoleXRead) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	params := listStudentParams{
		scheduleID: r.FormValue("schedule_id"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	studentIDs, err := cs.SelectEnrolledStudentID(args.scheduleID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	resp := []listStudentResponse{}
	if len(studentIDs) < 1 {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusOK).
			SetData(resp))
		return
	}

	users, err := usr.RequestID(studentIDs, true)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	for _, val := range users {
		resp = append(resp, listStudentResponse{
			UserIdentityCode: val.IdentityCode,
			UserName:         val.Name,
		})
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(resp))
	return
}

func AddAssistantHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)
	if !sess.IsHasRoles(auth.ModuleCourse, auth.RoleXCreate, auth.RoleCreate, auth.RoleXUpdate, auth.RoleUpdate) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusForbidden).
			AddError("You don't have privilege"))
		return
	}

	// assistant_id = user identity code
	params := addAssistantParams{
		assistentIdentityCodes: r.FormValue("assistant_id"),
		scheduleID:             ps.ByName("schedule_id"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	// check if creator or assistant of specific schedule id
	if !cs.IsAssistant(sess.ID, args.scheduleID) {
		if !cs.IsCreator(sess.ID, args.scheduleID) {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusForbidden).
				AddError("You are not authorized"))
			return
		}
	}

	newAssistant := []int64{}
	if len(args.assistentIdentityCodes) > 0 {
		newAssistant, err = user.SelectIDByIdentityCode(args.assistentIdentityCodes)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
	}

	// check if all user id is registered or valid
	if len(newAssistant) != len(args.assistentIdentityCodes) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	oldAssistant, err := cs.SelectAssistantID(args.scheduleID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	var insert []int64
	for _, val := range newAssistant {
		if !helper.Int64InSlice(val, oldAssistant) {
			insert = append(insert, val)
		}
	}

	var delete []int64
	for _, val := range oldAssistant {
		if !helper.Int64InSlice(val, newAssistant) {
			delete = append(delete, val)
		}
	}

	tx, err := conn.DB.Beginx()
	if err != nil {
		tx.Rollback()
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	if len(delete) > 0 {
		err = cs.DeleteAssistant(delete, args.scheduleID, tx)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		} //udah
	}

	if len(insert) > 0 {
		err = cs.InsertAssistant(insert, args.scheduleID, tx)
		if err != nil {
			tx.Rollback()
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
	}

	tx.Commit()

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetMessage("Success"))
	return
}

// GetTodayHandler ...
func GetTodayHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)

	dayNow := time.Now().Weekday()

	schedulesID, err := cs.SelectScheduleIDByUserID(sess.ID, cs.PStatusStudent)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	courses, err := cs.SelectByDayScheduleID(int8(dayNow), schedulesID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	resp := []getTodayResponse{}
	for _, val := range courses {
		t1 := helper.MinutesToTimeString(val.Schedule.StartTime)
		t2 := helper.MinutesToTimeString(val.Schedule.EndTime)
		t := fmt.Sprintf("%s - %s", t1, t2)
		resp = append(resp, getTodayResponse{
			ID:    val.Schedule.ID,
			Name:  val.Course.Name,
			Place: val.Schedule.PlaceID,
			Time:  t,
		})
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(resp))
	return
}

func EnrollRequestHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sess := r.Context().Value("User").(*auth.User)

	params := enrollRequestParams{
		scheduleID: ps.ByName("schedule_id"),
		payload:    r.FormValue("payload"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	if !cs.IsExistScheduleID(args.scheduleID) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	if cs.IsEnrolled(sess.ID, args.scheduleID) || cs.IsAssistant(sess.ID, args.scheduleID) {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	isUnapproved := cs.IsUnapproved(sess.ID, args.scheduleID)
	switch args.payload {
	case "enroll":
		if isUnapproved {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusBadRequest).
				AddError("Invalid Request"))
			return
		}
		err = cs.InsertUnapproved(sess.ID, args.scheduleID)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
	case "cancel":
		if !isUnapproved {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusBadRequest).
				AddError("Invalid Request"))
			return
		}
		err = cs.DeleteUserRelation(sess.ID, args.scheduleID)
		if err != nil {
			template.RenderJSONResponse(w, new(template.Response).
				SetCode(http.StatusInternalServerError))
			return
		}
	default:
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetMessage("Success"))
	return
}

func ExchangeInvolvedHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	params := exchangeInvolvedParams{
		userID: r.FormValue("user_id"),
		role:   r.FormValue("role"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	var status int8
	switch args.role {
	case "assistant":
		status = cs.PStatusAssistant
	case "student":
		status = cs.PStatusStudent
	default:
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	scheduleID, err := cs.SelectScheduleIDByUserID(args.userID, status)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	courses, err := cs.SelectByScheduleID(scheduleID, cs.StatusScheduleActive)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	resp := []exchangeInvolvedResponse{}
	for _, val := range courses {
		resp = append(resp, exchangeInvolvedResponse{
			Course: course{
				ID:          val.Course.ID,
				Name:        val.Course.Name,
				UCU:         val.Course.UCU,
				Description: val.Course.Description,
			},
			Schedule: schedule{
				ID:        val.Schedule.ID,
				Class:     val.Schedule.Class,
				CourseID:  val.Schedule.CourseID,
				CreatedBy: val.Schedule.CreatedBy,
				Day:       val.Schedule.Day,
				EndTime:   val.Schedule.EndTime,
				PlaceID:   val.Schedule.PlaceID,
				Semester:  val.Schedule.Semester,
				StartTime: val.Schedule.StartTime,
				Status:    val.Schedule.Status,
				Year:      val.Schedule.Year,
			},
		})
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetData(resp).
		SetCode(http.StatusOK))
	return
}

func ExchangeByScheduleHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	params := exchangeByScheduleParams{
		userID:     r.FormValue("user_id"),
		role:       r.FormValue("role"),
		scheduleID: r.FormValue("schedule_id"),
	}

	args, err := params.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	isInvolved := false
	switch args.role {
	case "assistant":
		isInvolved = cs.IsAssistant(args.userID, args.scheduleID)
	case "student":
		isInvolved = cs.IsEnrolled(args.userID, args.scheduleID)
	}

	resp := exchangeByScheduleResponse{
		Involved: isInvolved,
	}

	if !isInvolved {
		template.RenderJSONResponse(w, new(template.Response).
			SetData(resp).
			SetCode(http.StatusOK))
		return
	}

	c, err := cs.GetByScheduleID(args.scheduleID)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	resp.Course = &courseSchedule{
		Course: course{
			ID:          c.Course.ID,
			Name:        c.Course.Name,
			UCU:         c.Course.UCU,
			Description: c.Course.Description,
		},
		Schedule: schedule{
			ID:        c.Schedule.ID,
			Class:     c.Schedule.Class,
			CourseID:  c.Schedule.CourseID,
			CreatedBy: c.Schedule.CreatedBy,
			Day:       c.Schedule.Day,
			EndTime:   c.Schedule.EndTime,
			PlaceID:   c.Schedule.PlaceID,
			Semester:  c.Schedule.Semester,
			StartTime: c.Schedule.StartTime,
			Status:    c.Schedule.Status,
			Year:      c.Schedule.Year,
		},
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetData(resp).
		SetCode(http.StatusOK))
	return
}
