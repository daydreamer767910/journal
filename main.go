// main.go
package main

import (
	"embed"
	"flag"
	"io/fs"
	"net/http"

	"journal/handler"
	"journal/router"
	"journal/store/jsondb"
	"journal/util"

	"github.com/labstack/echo/v4"
)

// embed the "templates" directory
//
//go:embed templates/*
var embeddedTemplates embed.FS

// embed the "web" directory
//
//go:embed web/*
var embeddedWeb embed.FS

var (
	// command-line banner information
	appVersion = "development"
	gitCommit  = "https://github.com/daydreamer767910/journal.git"
	//gitRef     = "N/A"
	//buildTime  = fmt.Sprintf(time.Now().UTC().Format("01-02-2006 15:04:05"))
	// configuration variables
	flagBindAddress string
	flagJwtSecret   string
	flagBasePath    string
)

func init() {
	flag.StringVar(&flagBindAddress, "bind-address", util.LookupEnvOrString("BIND_ADDRESS", "0.0.0.0:5000"), "Address:Port to which the app will be bound.")
	flag.StringVar(&flagBasePath, "base-path", "/", "The base path of the URL")
	flag.StringVar(&flagJwtSecret, "jwt-secret", util.LookupEnvOrString("JWT_SECRET", "12345678"), "The JWT secret")
	flag.Parse()

	util.BasePath = util.ParseBasePath(flagBasePath)
	util.BindAddress = flagBindAddress
	util.JwtSecret, _ = util.HashPassword(flagJwtSecret)
}

func main() {
	db, err := jsondb.New("./db")
	if err != nil {
		panic(err)
	}
	if err := db.Init(); err != nil {
		panic(err)
	}
	config, err := db.GetServerConf()
	if err != nil {
		panic(err)
	}
	util.ThumbnailCfg = config.ThumbnailCfg

	match, err := util.VerifyHash(util.JwtSecret, config.JwtSecret)
	if err != nil {
		panic(err)
	}
	if !match {
		config.JwtSecret = util.JwtSecret
		if err := db.SaveServerConf(config); err != nil {
			panic(err)
		}
	}
	// strip the "templates/" prefix from the embedded directory so files can be read by their direct name (e.g.
	// "base.html" instead of "templates/base.html")
	tmplDir, _ := fs.Sub(fs.FS(embeddedTemplates), "templates")
	//bootstrapDir, _ := fs.Sub(fs.FS(embeddedBootstrap), "bootstrap")

	// set app extra data
	extraData := make(map[string]interface{})
	extraData["appVersion"] = appVersion
	extraData["gitCommit"] = gitCommit
	extraData["basePath"] = util.BasePath

	// register routes
	app := router.New(tmplDir, extraData)
	app.GET(util.BasePath+"/login", handler.LoginPage(db))
	app.GET(util.BasePath+"/register", handler.RegisterPage(db))
	app.GET(util.BasePath+"/home", handler.HomePage(db), handler.ValidJWT)
	app.GET(util.BasePath+"/dashboard", handler.DashboardPage(db), handler.ValidJWT)
	app.GET(util.BasePath+"/security", handler.SecurityPage(db), handler.ValidJWT)
	app.GET(util.BasePath+"/auth2fa", handler.Auth2FAPage(db), handler.ValidJWT)
	app.GET(util.BasePath+"/filesbrowser", handler.FileBrowserPage(db), handler.ValidJWT)
	app.GET(util.BasePath+"/workshop", handler.WorkshopPage(db), handler.ValidJWT)
	app.GET(util.BasePath+"/video", handler.VideoPage(db), handler.ValidJWT)
	app.GET(util.BasePath+"/logout", handler.LogoutPage(db), handler.ValidJWT)

	app.POST(util.BasePath+"/transform", handler.Transform(db), handler.ValidJWT)
	app.POST(util.BasePath+"/combine", handler.CombineFiles(db), handler.ValidJWT)
	app.POST(util.BasePath+"/listfile", handler.ListFiles(db), handler.ValidJWT)
	app.POST(util.BasePath+"/listworkshop", handler.ListWorkshop(db), handler.ValidJWT)
	app.POST(util.BasePath+"/logout", handler.Logout(db), handler.ValidJWT)
	app.POST(util.BasePath+"/login", handler.Login(db))
	app.POST(util.BasePath+"/register", handler.Register(db))
	app.POST(util.BasePath+"/auth2fa", handler.Auth2FA(db), handler.ValidJWT)
	app.POST(util.BasePath+"/enable2fa", handler.Enalbe2FA(db), handler.ValidJWT)
	app.POST(util.BasePath+"/disable2fa", handler.Disalbe2FA(db), handler.ValidJWT)
	app.POST(util.BasePath+"/changepassword", handler.ChangePassword(db), handler.ValidJWT)
	app.POST(util.BasePath+"/upload", handler.Upload(db), handler.ValidJWT)
	app.POST(util.BasePath+"/delete", handler.DeleteFiles(db), handler.ValidJWT)

	webDir, _ := fs.Sub(fs.FS(embeddedWeb), "web")
	webHandler := http.FileServer(http.FS(webDir))
	// serves other static files
	app.GET(util.BasePath+"/*", echo.WrapHandler(http.StripPrefix(util.BasePath, webHandler)))

	publicHandler := http.StripPrefix(util.BasePath+"/public/", http.FileServer(http.Dir(util.BasePath+"public")))
	group := app.Group(util.BasePath + "/public/*")
	group.Use(handler.ValidJWT)
	group.GET("/*", echo.WrapHandler(publicHandler))

	app.Logger.Fatal(app.Start(util.BindAddress))
}
