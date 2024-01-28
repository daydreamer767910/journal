package handler

import (
	"journal/store"
	"journal/util"
	"net/http"

	"github.com/labstack/echo/v4"
)

// LoginPage handler
func HomePage() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "home.html", map[string]interface{}{})
	}
}

func RegisterPage() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "register.html", map[string]interface{}{})
	}
}

// LoginPage handler
func LoginPage() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "login.html", map[string]interface{}{})
	}
}

func SecurityPage(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		userid := c.Get("userid").(string)

		user, err := db.GetUserByID(userid)
		if err != nil {
			return c.Redirect(http.StatusTemporaryRedirect, util.BasePath+"/login")
		}
		return c.Render(http.StatusOK, "security.html", map[string]interface{}{
			"username":  user.Username,
			"enable2fa": user.Enable2FA,
		})
	}
}

func Auth2FAPage(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "2faverify.html", map[string]interface{}{})
	}
}

func DashboardPage(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		userid := c.Get("userid").(string)
		tokentype := c.Get("jwttype").(string)

		user, err := db.GetUserByID(userid)
		if err != nil {
			return c.Redirect(http.StatusTemporaryRedirect, util.BasePath+"/login")
		}
		if user.Enable2FA == true && tokentype != "2FA" {
			return c.JSON(http.StatusUnauthorized, jsonHTTPResponse{0, "need to pass 2FA auth first", ""})
		}
		return c.Render(http.StatusOK, "dashboard.html", map[string]interface{}{
			"username": user.Username,
		})
	}
}

func FileBrowser(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "filebrowser.html", map[string]interface{}{})
	}
}
