package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestGreetHandler(t *testing.T) {
	e := echo.New()

	// 模拟 GET 请求
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	// 执行请求
	e.ServeHTTP(rec, req)

	// 确保 HTTP 状态码是 200 OK
	assert.Equal(t, http.StatusOK, rec.Code)

	// 确保响应主体包含预期的 JSON 数据
	assert.JSONEq(t, `{"message":"Hello, Echo Test!"}`, rec.Body.String())
}
