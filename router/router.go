package router

import (
	"errors"
	"html/template"
	"io"
	"io/fs"
	"reflect"
	"strings"

	"journal/handler"
	"journal/model"
	"journal/util"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

// TemplateRegistry defines the custom renderer for Echo
type TemplateRegistry struct {
	templates map[string]*template.Template
	extraData map[string]interface{}
}

// Render implements the Renderer interface for custom rendering
func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// Custom rendering logic using t.templates and data
	tmpl, ok := t.templates[name]
	if !ok {
		err := errors.New("Template not found -> " + name)
		return err
	}
	//fmt.Println(name)
	// inject more app data information. E.g. appVersion
	if reflect.TypeOf(data).Kind() == reflect.Map {
		for k, v := range t.extraData {
			data.(map[string]interface{})[k] = v
		}

		data.(map[string]interface{})["server_config"] = model.ServerConf{}
	}

	if name == "home.html" {
		return tmpl.Execute(w, data)
	}

	return tmpl.ExecuteTemplate(w, "base", data)

}

func New(tmplDir fs.FS, extraData map[string]interface{}) *echo.Echo {
	tmplHomeString, err := util.StringFromEmbedFile(tmplDir, "home.html")
	if err != nil {
		log.Fatal(err)
	}
	tmplRegisterString, err := util.StringFromEmbedFile(tmplDir, "register.html")
	if err != nil {
		log.Fatal(err)
	}
	tmplLoginString, err := util.StringFromEmbedFile(tmplDir, "login.html")
	if err != nil {
		log.Fatal(err)
	}
	// read html template file to string
	tmplDashboardString, err := util.StringFromEmbedFile(tmplDir, "dashboard.html")
	if err != nil {
		log.Fatal(err)
	}
	tmplFilebrowserString, err := util.StringFromEmbedFile(tmplDir, "filebrowser.html")
	if err != nil {
		log.Fatal(err)
	}
	tmplSecurityString, err := util.StringFromEmbedFile(tmplDir, "security.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl2faverifyString, err := util.StringFromEmbedFile(tmplDir, "2faverify.html")
	if err != nil {
		log.Fatal(err)
	}
	tmplBaseString, err := util.StringFromEmbedFile(tmplDir, "base.html")
	if err != nil {
		log.Fatal(err)
	}

	// create template list
	funcs := template.FuncMap{
		"StringsJoin":  strings.Join,
		"GetThumbnail": handler.GetThumnail,
	}
	templates := make(map[string]*template.Template)
	templates["home.html"] = template.Must(template.New("home").Funcs(funcs).Parse(tmplHomeString))
	templates["register.html"] = template.Must(template.New("register").Funcs(funcs).Parse(tmplBaseString + tmplRegisterString))
	templates["login.html"] = template.Must(template.New("login").Funcs(funcs).Parse(tmplBaseString + tmplLoginString))
	templates["dashboard.html"] = template.Must(template.New("dashboard").Funcs(funcs).Parse(tmplBaseString + tmplDashboardString))
	templates["security.html"] = template.Must(template.New("security").Funcs(funcs).Parse(tmplBaseString + tmplSecurityString))
	templates["2faverify.html"] = template.Must(template.New("2faverify").Funcs(funcs).Parse(tmplBaseString + tmpl2faverifyString))
	templates["filebrowser.html"] = template.Must(template.New("filebrowser").Funcs(funcs).Parse(tmplBaseString + tmplFilebrowserString))

	lvl, err := util.ParseLogLevel("INFO")
	if err != nil {
		log.Fatal(err)
	}
	logConfig := middleware.DefaultLoggerConfig
	/*logConfig.Skipper = func(c echo.Context) bool {
		resp := c.Response()
		if resp.Status >= 500 && lvl > log.ERROR { // do not log if response is 5XX but log level is higher than ERROR
			return true
		} else if resp.Status >= 400 && lvl > log.WARN { // do not log if response is 4XX but log level is higher than WARN
			return true
		} else if lvl > log.DEBUG { // do not log if log level is higher than DEBUG
			return true
		}
		return false
	}*/
	logConfig.Output = &lumberjack.Logger{
		Filename:   "logs/app.log",
		MaxSize:    10, // MB
		MaxBackups: 5,
		MaxAge:     7, // days
		Compress:   true,
	}

	e := echo.New()

	e.Logger.SetLevel(lvl)
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.LoggerWithConfig(logConfig))

	// 使用 echo.Static 中间件处理静态文件
	e.Use(middleware.Static(""))

	e.HideBanner = true
	e.HidePort = lvl > log.INFO // hide the port output if the log level is higher than INFO
	e.Validator = NewValidator()
	e.Renderer = &TemplateRegistry{
		templates: templates,
		extraData: extraData,
	}
	// 中间件：HTTPS重定向
	e.Pre(middleware.HTTPSRedirect())

	return e
}
