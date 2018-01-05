package handler

import (
	"net/http"

	"github.com/asepnur/meiko_course/src/webserver/template"
	"github.com/julienschmidt/httprouter"
)

func HelloHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetMessage("Hello from meiko. Have a nice day! :)"))
	return
}
