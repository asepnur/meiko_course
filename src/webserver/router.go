package webserver

import (
	"net/http"

	"github.com/asepnur/meiko_course/src/util/auth"
	"github.com/asepnur/meiko_course/src/webserver/handler/assignment"
	"github.com/asepnur/meiko_course/src/webserver/handler/attendance"
	"github.com/asepnur/meiko_course/src/webserver/handler/bot"
	"github.com/asepnur/meiko_course/src/webserver/handler/course"
	"github.com/asepnur/meiko_course/src/webserver/handler/file"
	"github.com/asepnur/meiko_course/src/webserver/handler/place"
	"github.com/asepnur/meiko_course/src/webserver/handler/tutorial"

	"github.com/julienschmidt/httprouter"
)

// Load returns all routing of this server
func loadRouter(r *httprouter.Router) {

	// ========================== File Handler ==========================
	r.GET("/api/v1/filerouter", auth.OptionalAuthorize(file.RouterFileHandler))
	r.GET("/api/v1/file/:payload/:filename", file.GetFileHandler)
	r.GET("/api/v1/admin/file/types/available", auth.MustAuthorize(file.AvailableTypes))
	r.GET("/api/v1/image/:payload", auth.MustAuthorize(file.GetProfileHandler))
	r.POST("/api/v1/image/profile", auth.MustAuthorize(file.UploadProfileImageHandler))
	r.POST("/api/admin/v1/image/information", auth.MustAuthorize(file.UploadInformationImageHandler))
	// r.POST("/api/v1/file/assignment", auth.MustAuthorize(file.UploadAssignmentHandler))
	r.POST("/api/v1/file", auth.MustAuthorize(file.UploadFileHandler))
	r.GET("/static/*filepath", file.StaticHandler)
	// ======================== End File Handler ========================

	// ========================= Course Handler =========================
	// User section
	r.GET("/api/v1/course", auth.MustAuthorize(course.GetHandler))
	r.GET("/api/v1/course/:schedule_id", auth.MustAuthorize(course.GetDetailHandler))
	r.GET("/api/v1/course/:schedule_id/assistant", auth.MustAuthorize(course.GetAssistantHandler))
	r.POST("/api/v1/course/:schedule_id/enrollment", auth.MustAuthorize(course.EnrollRequestHandler))

	// Admin section
	r.GET("/api/admin/v1/course", auth.MustAuthorize(course.ReadHandler))
	r.POST("/api/admin/v1/course", auth.MustAuthorize(course.CreateHandler))
	r.GET("/api/admin/v1/course/:schedule_id", auth.MustAuthorize(course.ReadDetailHandler))                      //read
	r.GET("/api/admin/v1/course/:schedule_id/parameter", auth.MustAuthorize(course.ReadScheduleParameterHandler)) //read
	r.PATCH("/api/admin/v1/course/:schedule_id", auth.MustAuthorize(course.UpdateHandler))
	r.DELETE("/api/admin/v1/course/:schedule_id", auth.MustAuthorize(course.DeleteScheduleHandler))
	r.POST("/api/admin/v1/course/:schedule_id/assistant", auth.MustAuthorize(course.AddAssistantHandler))
	r.GET("/api/admin/v1/list/course/parameter", auth.MustAuthorize(course.ListParameterHandler))
	r.GET("/api/admin/v1/list/course/search", auth.MustAuthorize(course.SearchHandler))
	// ======================== End Course Handler ======================

	// ======================== Tutorial Handler ========================
	r.GET("/api/v1/tutorial", auth.MustAuthorize(tutorial.ReadHandler)) // for admin and user
	r.POST("/api/admin/v1/tutorial", auth.MustAuthorize(tutorial.CreateHandler))
	r.GET("/api/admin/v1/tutorial/:tutorial_id", auth.MustAuthorize(tutorial.ReadDetailHandler))
	r.PATCH("/api/admin/v1/tutorial/:tutorial_id", auth.MustAuthorize(tutorial.UpdateHandler))
	r.DELETE("/api/admin/v1/tutorial/:tutorial_id", auth.MustAuthorize(tutorial.DeleteHandler))
	// ====================== End Tutorial Handler ======================

	// ======================= Attendance Handler =======================
	// Admin section
	r.GET("/api/v1/attendance/list", auth.MustAuthorize(attendance.ListStudentHandler))
	r.GET("/api/v1/attendance/summary", auth.MustAuthorize(attendance.GetAttendanceHandler))
	r.GET("/api/admin/v1/attendance", auth.MustAuthorize(attendance.ReadMeetingHandler))
	r.POST("/api/admin/v1/attendance", auth.MustAuthorize(attendance.CreateMeetingHandler))
	r.GET("/api/admin/v1/attendance/:meeting_id", auth.MustAuthorize(attendance.ReadMeetingDetailHandler))
	r.DELETE("/api/admin/v1/attendance/:meeting_id", auth.MustAuthorize(attendance.DeleteMeetingHandler))
	r.PATCH("/api/admin/v1/attendance/:meeting_id", auth.MustAuthorize(attendance.UpdateMeetingHandler))
	// ===================== End Attendance Handler =====================
	// =========================== Bot Handler ==========================
	// User section
	r.GET("/api/v1/bot", auth.MustAuthorize(bot.LoadHistoryHandler))
	r.POST("/api/v1/bot", auth.MustAuthorize(bot.BotHandler))
	// ========================= End Bot Handler ========================

	// ========================= Assignment Handler ========================
	r.GET("/api/admin/v1/assignment", auth.MustAuthorize(assignment.ReadHandler))
	r.GET("/api/admin/v1/assignment/:id/available", auth.MustAuthorize(assignment.GetAvailableGP))
	r.POST("/api/admin/v1/assignment", auth.MustAuthorize(assignment.CreateHandler))
	r.PATCH("/api/admin/v1/assignment/:id", auth.MustAuthorize(assignment.UpdateHandler))
	r.GET("/api/admin/v1/assignment/:id", auth.MustAuthorize(assignment.DetailHandler))
	r.DELETE("/api/admin/v1/assignment/:id", auth.MustAuthorize(assignment.DeleteHandler))
	// r.GET("/api/admin/v1/assignment/:id/:assignment_id", auth.MustAuthorize(assignment.GetUploadedAssignmentByAdminHandler))
	// r.GET("/api/admin/v1/score/:schedule_id/:assignment_id", auth.MustAuthorize(assignment.GetDetailAssignmentByAdmin))
	// r.POST("/api/admin/v1/score/:schedule_id/:assignment_id/create", auth.MustAuthorize(assignment.CreateScoreHandler)) // update score

	r.GET("/api/v1/assignment", auth.MustAuthorize(assignment.GetHandler))           // assignment list
	r.GET("/api/v1/assignment/:id", auth.MustAuthorize(assignment.GetDetailHandler)) // assignment detail
	r.PUT("/api/v1/assignment/:id", auth.MustAuthorize(assignment.SubmitHandler))    // assignment submit
	// r.POST("/api/v1/assignment", auth.MustAuthorize(assignment.CreateHandlerByUser))                                     // create upload by user
	// r.GET("/api/v1/assignment/:id/:schedule_id/:assignment_id", auth.MustAuthorize(assignment.GetUploadedDetailHandler)) // detail user assignments
	// r.GET("/api/v1/assignment-schedule", auth.MustAuthorize(assignment.GetAssignmentByScheduleHandler))                  // List assignments
	r.GET("/api/v1/grade", auth.MustAuthorize(assignment.GetReportHandler))
	// r.GET("/api/v1/grade/:id", auth.MustAuthorize(assignment.GradeBySchedule))
	// ===================== End Assignment Handler =====================

	// ========================== Place Handler =========================
	// Public section
	r.GET("/api/v1/place/search", place.SearchHandler)
	// ======================== End Place Handler =======================

	// Catch
	r.NotFound = http.HandlerFunc(file.IndexHandler)
	// r.MethodNotAllowed = http.RedirectHandler("/", http.StatusPermanentRedirect)
}
