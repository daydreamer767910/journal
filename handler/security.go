package handler

import (
	"crypto/subtle"
	"journal/store"
	"journal/util"
	"net/http"

	"github.com/labstack/echo/v4"
)

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

func Disalbe2FA(db store.IStore) echo.HandlerFunc {
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
		user.Enable2FA = false
		user.Secret2FA = ""
		db.SaveUser(user)
		return c.JSON(http.StatusOK, jsonHTTPResponse{1, "2FA disabled", ""})
	}
}

func Enalbe2FA(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		userid := c.Get("userid").(string)

		user, err := db.GetUserByID(userid)
		if err != nil {
			return c.Redirect(http.StatusTemporaryRedirect, util.BasePath+"/login")
		}
		totp_key, err := generate2FAKey(user.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "generate key fail", err.Error()})
		}
		user.Secret2FA = totp_key.Secret()
		user.Enable2FA = true
		//need a new JWT as well
		tokenstring, err := generateToken(userid, util.JwtSecret, "2FA")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "JWT generate fail", err.Error()})
		}
		//update JWT
		user.Token = tokenstring

		//use cookie to keep the token, Bearer Tokens to be added...
		setCookie(c, tokenstring)
		db.SaveUser(user)
		return c.JSON(http.StatusOK, jsonHTTPResponse{1, totp_key.URL(), totp_key.Secret()})
	}
}

func ChangePassword(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		userid := c.Get("userid").(string)

		user, err := db.GetUserByID(userid)
		if err != nil {
			return c.Redirect(http.StatusTemporaryRedirect, util.BasePath+"/login")
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
