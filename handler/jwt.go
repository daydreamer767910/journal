package handler

import (
	"fmt"
	"journal/util"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
)

const TokenExpDuration = time.Hour * 24

func generateToken(userID string, secret string, other interface{}) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"exp":   time.Now().Add(TokenExpDuration).Unix(),
		"other": other,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func setCookie(c echo.Context, token string) {
	cookie := &http.Cookie{
		Name:     "Authorization",
		Value:    token,
		Expires:  time.Now().Add(time.Hour * 1), // 1小时过期
		HttpOnly: true,
		Secure:   true, // 仅在 HTTPS 连接中发送
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}
	c.SetCookie(cookie)
}

func extractToken(c echo.Context) (string, error) {
	cookie, err := c.Cookie("Authorization")
	if err != nil {
		return "", err
	}
	//fmt.Println("cookie.Value:", cookie.Value)
	return cookie.Value, nil
}

func validateToken(tokenString string, secret string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

func ValidJWT(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		token, err := extractToken(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, jsonHTTPResponse{0, "no JWT", err})
		}
		jwtToken, err := validateToken(token, util.JwtSecret)
		if err != nil {
			return c.JSON(http.StatusNonAuthoritativeInfo, jsonHTTPResponse{0, "bad JWT", err})
		}

		claims := jwtToken.Claims.(jwt.MapClaims)

		userID, ok := claims["sub"].(string)
		if !ok {
			return c.JSON(http.StatusNonAuthoritativeInfo, jsonHTTPResponse{0, "bad JWT", "sub"})
		}
		other, ok := claims["other"].(interface{})
		if !ok {
			return c.JSON(http.StatusNonAuthoritativeInfo, jsonHTTPResponse{0, "bad JWT", "type"})
		}
		expTime, ok := claims["exp"].(float64)
		if !ok {
			return c.JSON(http.StatusNonAuthoritativeInfo, jsonHTTPResponse{0, "bad JWT", "exp"})
		}

		fmt.Printf("user:%v, exp:%v, other:%v\n", userID, expTime, other)
		c.Set("userid", userID)
		c.Set("jwttype", other.(string))

		return next(c)
	}
}
