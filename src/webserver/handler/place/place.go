package place

import (
	"net/http"

	pl "github.com/asepnur/meiko_course/src/module/place"
	"github.com/asepnur/meiko_course/src/webserver/template"
	"github.com/julienschmidt/httprouter"
)

func SearchHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	params := searchParams{
		Query: r.FormValue("qry"),
	}

	args, err := params.Validate()
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusBadRequest).
			AddError("Invalid Request"))
		return
	}

	if len(args.Query) <= 3 {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusOK).
			SetData(searchResponse{
				ID: []string{},
			}))
		return
	}

	places, err := pl.Search(args.Query)
	if err != nil {
		template.RenderJSONResponse(w, new(template.Response).
			SetCode(http.StatusInternalServerError))
		return
	}

	template.RenderJSONResponse(w, new(template.Response).
		SetCode(http.StatusOK).
		SetData(searchResponse{
			ID: places,
		}))
	return

}
