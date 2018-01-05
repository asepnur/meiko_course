package notification

import (
	"net/http"

	"github.com/asepnur/meiko_course/src/module/notification"
	"github.com/asepnur/meiko_course/src/util/auth"
	"github.com/asepnur/meiko_course/src/webserver/template"
	"github.com/julienschmidt/httprouter"
)

func GetHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	param := getNotificationParam{
		page: r.FormValue("_pg"),
	}

	args, err := param.validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError(err.Error()))
		return
	}

	const limit = uint8(10)
	u := r.Context().Value("User").(*auth.User)

	n, err := notification.Get(u.ID, args.page, limit)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError).
			AddError(err.Error()))
		return
	}

	var res []Notification
	for _, v := range n {
		res = append(res, Notification{
			ID:          v.ID,
			Name:        v.Name,
			Description: v.Description,
			URL:         v.GetURL(),
			CreatedAt:   v.CreatedAt.Unix(),
		})
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(res))
	return
}
