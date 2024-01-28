package handler

import (
	"journal/store"
	"journal/util"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func generate2FAKey(username string) (*otp.Key, error) {

	totp_key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Journal",
		AccountName: username,
		SecretSize:  64,
	})

	/*
		QRcode, err := qrcode.New(totp_key.URL(), qrcode.Medium)
		if err != nil {
			fmt.Println("Error generating QR code:", err)
		}

		fmt.Println(QRcode.ToSmallString(false))*/
	return totp_key, err
}

func Auth2FA(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		userid := c.Get("userid").(string)

		user, err := db.GetUserByID(userid)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, jsonHTTPResponse{0, "bad user id", ""})
		}

		var req jsonHTTPVerify2FA
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{0, "Bad post data", err.Error()})
		}
		//fmt.Println(req)
		valid := totp.Validate(req.Code, user.Secret2FA)
		if !valid {
			return c.JSON(http.StatusUnauthorized, jsonHTTPResponse{0, "Invalid 2FA code", ""})
		}

		tokenstring, err := generateToken(userid, util.JwtSecret, "2FA")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "JWT generate fail", err.Error()})
		}
		//update dbuser JWT
		user.Token = tokenstring
		db.SaveUser(user)
		//use cookie to keep the token, Bearer Tokens to be added...
		setCookie(c, tokenstring)
		return c.JSON(http.StatusOK, jsonHTTPResponse{1, "2FA successfully", ""})
	}
}

func Disalbe2FA(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		userid := c.Get("userid").(string)
		tokentype := c.Get("jwttype").(string)

		user, err := db.GetUserByID(userid)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, jsonHTTPResponse{0, "bad user id", ""})
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
			return c.JSON(http.StatusUnauthorized, jsonHTTPResponse{0, "bad user id", ""})
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
