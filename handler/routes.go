package handler

import (
	"crypto/subtle"
	"journal/store"
	"journal/util"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/xid"
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

func Register(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var data jsonHTTPRegisterForm
		if err := c.Bind(&data); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{0, "Bad post data:", err.Error()})
		}
		if err := c.Validate(data); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{0, "Bad post data:", err.Error()})
		}
		dbuser, err := db.GetUserByName(data.Username)
		if err == nil {
			return c.JSON(http.StatusUnauthorized, jsonHTTPResponse{0, "user existed", ""})
		}
		userid := xid.New().String()
		tokenstring, err := generateToken(userid, util.JwtSecret, "login")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "JWT generate fail", err.Error()})
		}
		//regist dbuser
		dbuser.Userid = userid
		dbuser.Username = data.Username
		dbuser.Password = data.Password
		dbuser.Email = data.Email
		dbuser.Phone = data.Phone
		dbuser.Adress = data.Adress
		dbuser.PasswordHash, _ = util.HashPassword(data.Password)
		dbuser.Enable2FA = false
		dbuser.Secret2FA = ""
		dbuser.Token = tokenstring
		db.SaveUser(dbuser)
		//use cookie to keep the token, Bearer Tokens to be added...
		setCookie(c, tokenstring)
		//return c.JSON(http.StatusOK, map[string]string{"token": token, "secret": user.Secret})
		return c.JSON(http.StatusOK, jsonHTTPResponse{1, "Register successfully", dbuser})
	}
}

// Login for signing in handler
func Login(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var data jsonHTTPLoginForm
		if err := c.Bind(&data); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{0, "Bad post data:", err.Error()})
		}

		username := data.Username
		password := data.Password
		//rememberMe := data.RememberMe
		dbuser, err := db.GetUserByName(username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "Cannot query user ", username})
		}
		userCorrect := subtle.ConstantTimeCompare([]byte(username), []byte(dbuser.Username)) == 1

		var passwordCorrect bool
		if dbuser.PasswordHash != "" {
			match, err := util.VerifyHash(dbuser.PasswordHash, password)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "Wrong password", ""})
			}
			passwordCorrect = match
		} else {
			passwordCorrect = subtle.ConstantTimeCompare([]byte(password), []byte(dbuser.Password)) == 1
		}

		if userCorrect && passwordCorrect {
			userid := xid.New().String()
			tokenstring, err := generateToken(userid, util.JwtSecret, "login")
			if err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "JWT generate fail", err.Error()})
			}
			//update dbuser
			if dbuser.PasswordHash == "" {
				dbuser.PasswordHash, _ = util.HashPassword(password)
			}
			dbuser.Userid = userid
			dbuser.Token = tokenstring
			db.SaveUser(dbuser)
			//use cookie to keep the token, Bearer Tokens to be added...
			setCookie(c, tokenstring)
			//return c.JSON(http.StatusOK, map[string]string{"token": token, "secret": user.Secret})
			return c.JSON(http.StatusOK, jsonHTTPResponse{1, "Logged in successfully", dbuser})
		}
		return c.JSON(http.StatusUnauthorized, jsonHTTPResponse{0, "Invalid credentials", ""})
	}
}

// Logout to log a user out
func Logout(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		userid := c.Get("userid").(string)
		tokentype := c.Get("jwttype").(string)

		user, err := db.GetUserByID(userid)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, jsonHTTPResponse{0, "need to login first", ""})
		}
		if user.Enable2FA == true && tokentype != "2FA" {
			return c.JSON(http.StatusUnauthorized, jsonHTTPResponse{0, "need to pass 2FA auth first", ""})
		}
		setCookie(c, "")
		return c.Redirect(http.StatusTemporaryRedirect, util.BasePath+"/home")
	}
}
