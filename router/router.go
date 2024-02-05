package router

import (
	"html/template"
	"io"
	"io/fs"
	"reflect"
	"strings"

	"journal/model"
	"journal/util"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

// TemplateRegistry defines the custom renderer for Echo
type TemplateRegistry struct {
	templates *template.Template
	extraData map[string]interface{}
}

// Render implements the Renderer interface for custom rendering
func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// Custom rendering logic using t.templates and data
	// inject more app data information. E.g. appVersion
	if reflect.TypeOf(data).Kind() == reflect.Map {
		for k, v := range t.extraData {
			data.(map[string]interface{})[k] = v
		}
		data.(map[string]interface{})["server_config"] = model.ServerConf{}
	}
	//fmt.Printf("render[%s]\n", name)
	return t.templates.ExecuteTemplate(w, name, data)
}

func New(tmplDir fs.FS, extraData map[string]interface{}) *echo.Echo {

	funcs := template.FuncMap{
		"StringsJoin": strings.Join,
	}

	templates := template.Must(template.New("journal").Funcs(funcs).ParseFS(tmplDir, "*.html"))

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

	e.HideBanner = true
	e.HidePort = lvl > log.INFO // hide the port output if the log level is higher than INFO
	e.Validator = NewValidator()

	e.Renderer = &TemplateRegistry{
		templates: templates,
		extraData: extraData,
	}
	// 中间件：HTTPS重定向
	//e.Pre(middleware.HTTPSRedirect())
	// 使用 echo.Static 中间件处理静态文件
	e.Use(middleware.Static(util.BasePath + "web"))

	return e
}
