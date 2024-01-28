package handler

import (
	"crypto/subtle"
	"journal/store"
	"journal/util"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/xid"
)

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
		return c.JSON(http.StatusOK, jsonHTTPResponse{1, "Logged out successfully", ""})
	}
}

func ChangePassword(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		userid := c.Get("userid").(string)

		user, err := db.GetUserByID(userid)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, jsonHTTPResponse{0, "bad user id", ""})
		}
		// 解析 JSON 请求体
		var request jsonHTTPChangePassword
		if err := c.Bind(&request); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{0, "Invalid request", ""})
		}
		if err := c.Validate(request); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{0, "Bad post data:", err.Error()})
		}
		var passwordCorrect bool
		if user.PasswordHash != "" {
			match, err := util.VerifyHash(user.PasswordHash, request.OldPassword)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, err.Error(), ""})
			}
			passwordCorrect = match
		} else {
			passwordCorrect = subtle.ConstantTimeCompare([]byte(request.OldPassword), []byte(user.Password)) == 1
		}

		if passwordCorrect {
			user.Password = request.NewPassword
			user.PasswordHash, _ = util.HashPassword(request.NewPassword)
			db.SaveUser(user)
			return c.JSON(http.StatusOK, jsonHTTPResponse{1, "Password changed successfully", ""})
		}
		return c.JSON(http.StatusBadRequest, jsonHTTPResponse{0, "Wrong old password", ""})
	}
}
