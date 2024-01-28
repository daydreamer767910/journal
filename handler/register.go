package handler

import (
	"journal/store"
	"journal/util"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/xid"
)

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
