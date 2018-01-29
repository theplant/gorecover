package gorecover

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"runtime"

	. "github.com/paulbellamy/mango"
	"github.com/theplant/airbrake-go"
	"github.com/theplant/mangotemplate"
	"github.com/theplant/qortex/i18n"
	"gopkg.in/mgo.v2"
)

const (
	ajax_key            = "x-requested-with"
	ajax_value          = "XMLHttpRequest"
	content_type_key    = "Content-Type"
	json_content_type   = "application/json"
	html_content_type   = "text/html; charset=utf8"
	not_found_body      = "Page not found"
	internal_error_body = "Internal error"
)

var templateString = `
	<html><body>
       	<h1>{{.Body}}</h1>
       	</body></html>`

var defaultTemplate = template.Must(template.New("default").Parse(templateString))

type Pages struct {
	NotFoundPath      string
	notFound          string
	InternalErrorPath string
	internalError     string
}

func (this *Pages) excute() (err error) {
	buffer := bytes.NewBufferString("")
	if this.NotFoundPath == "" {
		defaultTemplate.Execute(buffer, struct{ Body string }{not_found_body})
	} else {
		notFoundTemplate, err := template.ParseFiles(this.NotFoundPath)
		if err != nil {
			return err
		}
		notFoundTemplate.Execute(buffer, nil)
	}
	this.notFound = buffer.String()

	buffer = bytes.NewBufferString("")
	if this.InternalErrorPath == "" {
		defaultTemplate.Execute(buffer, struct{ Body string }{internal_error_body})
	} else {
		internalErrorTemplate, err := template.ParseFiles(this.InternalErrorPath)
		if err != nil {
			return err
		}
		internalErrorTemplate.Execute(buffer, nil)
	}
	this.internalError = buffer.String()

	return
}

func ErrorRecover(pages *Pages) Middleware {
	if err := pages.excute(); err != nil {
		panic(err)
	}

	return func(env Env, app App) (status Status, headers Headers, body Body) {
		defer func() {
			if err := recover(); err != nil {

				fmt.Fprintf(os.Stderr, "-------> recover: %v\n", err)

				airbrake.Error(fmt.Errorf("%+v", err), env.Request().Request)
				for skip := 1; ; skip++ {
					pc, file, line, ok := runtime.Caller(skip)
					if !ok {
						break
					}
					if file[len(file)-1] == 'c' {
						continue
					}
					f := runtime.FuncForPC(pc)
					log.Printf("%s:%d %s()\n", file, line, f.Name())
				}
				println("<------- \n")

				headers = Headers{}
				if env.Request().Header.Get(ajax_key) == ajax_value {
					headers.Set(content_type_key, json_content_type)
				} else {
					headers.Set(content_type_key, html_content_type)
				}

				if err == mgo.ErrNotFound {
					status = 404
					body = Body(pages.notFound)
				} else {
					status = 500
					body = Body(pages.internalError)
				}
			}
		}()

		return app(env)
	}
}

func ErrorRecoverI18n(pages *Pages) Middleware {

	return func(env Env, app App) (status Status, headers Headers, body Body) {

		defer func() {
			if err := recover(); err != nil {

				locale := i18n.GetLocale(env)
				if locale == nil {
					locale = i18n.EN
				}

				fmt.Fprintf(os.Stderr, "-------> recover: %v\n", err)

				airbrake.Error(fmt.Errorf("%+v", err), env.Request().Request)
				for skip := 1; ; skip++ {
					pc, file, line, ok := runtime.Caller(skip)
					if !ok {
						break
					}
					if file[len(file)-1] == 'c' {
						continue
					}
					f := runtime.FuncForPC(pc)
					log.Printf("%s:%d %s()\n", file, line, f.Name())
				}
				println("<------- \n")

				headers = Headers{}
				if env.Request().Header.Get(ajax_key) == ajax_value {
					headers.Set(content_type_key, json_content_type)
				} else {
					headers.Set(content_type_key, html_content_type)
				}

				localizedTemplate := locale.GetLocalizedTemplate()
				if err == mgo.ErrNotFound {
					status = 404
					body = Body(mangotemplate.RenderToStringT(pages.NotFoundPath, localizedTemplate, nil))
				} else {
					status = 500
					body = Body(mangotemplate.RenderToStringT(pages.InternalErrorPath, localizedTemplate, nil))
				}
			}

			if status == 404 {
				if env.Request().Header.Get(ajax_key) != ajax_value {
					locale := i18n.GetLocale(env)
					if locale == nil {
						locale = i18n.EN
					}
					localizedTemplate := locale.GetLocalizedTemplate()
					body = Body(mangotemplate.RenderToStringT(pages.NotFoundPath, localizedTemplate, nil))
				}
			} else if status == 500 {
				if env.Request().Header.Get(ajax_key) != ajax_value {
					locale := i18n.GetLocale(env)
					if locale == nil {
						locale = i18n.EN
					}
					localizedTemplate := locale.GetLocalizedTemplate()
					body = Body(mangotemplate.RenderToStringT(pages.InternalErrorPath, localizedTemplate, nil))
				}
			}
		}()

		return app(env)
	}
}
