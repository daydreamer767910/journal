package handler

import (
	"fmt"
	"io"
	"journal/store"
	"journal/util"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

func ListFiles(db store.IStore) echo.HandlerFunc {
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
		var upload_path string
		if user.Admin {
			upload_path = "public"
		} else {
			upload_path = filepath.Join("public", "uploads", user.Username)
		}

		files, _ := util.ListFiles(upload_path)

		return c.JSON(http.StatusOK, jsonHTTPResponse{1, "read dir ok", files})
	}
}

func Upload(db store.IStore) echo.HandlerFunc {
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
		// 解析表单
		form, err := c.MultipartForm()
		if err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{0, "Error parsing form", err})
		}

		// 获取上传的文件
		files := form.File["files[]"]
		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Println("Failed to get current directory:", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "", err})

		}
		upload_path := filepath.Join(currentDir, "public", "uploads", user.Username)

		// 使用 os.MkdirAll 创建目录，包括所有不存在的父目录
		err = os.MkdirAll(upload_path, os.ModePerm)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "create dir fail", err.Error()})
		}

		// 遍历处理每个文件
		for _, file := range files {
			// 打开目标文件
			dst_file := filepath.Join(upload_path, file.Filename)
			dst, err := os.Create(dst_file)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "Error creating destination file", err.Error()})
			}
			defer dst.Close()
			// 复制文件内容到目标文件
			src, err := file.Open()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "Error opening source file", err.Error()})
			}
			defer src.Close()
			if _, err := io.Copy(dst, src); err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "Error copying file", err.Error()})
			}
			if strings.Contains(util.GetMediaType(dst_file), "video") {
				var percentages []int
				var durations []int
				for _, cfg := range util.ThumbnailCfg {
					percentages = append(percentages, cfg.PercentPosition)
					durations = append(durations, cfg.Duration)
				}

				util.GenerateThumbnail(dst_file, percentages, durations)
			}

		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{1, "Files uploaded successfully", ""})
	}
}

func DeleteFiles(db store.IStore) echo.HandlerFunc {
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
		var request jsonHTTPDeleteFiles
		/*struct {
			Files []string `json:"files"`
		}*/
		if err := c.Bind(&request); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{0, "Bad post data", err.Error()})
		}

		for _, file := range request.Files {
			file, _ = url.QueryUnescape(file)
			// 实现您的文件删除逻辑
			if err := os.Remove(file); err != nil {
				fmt.Printf("Deleting file[%s][%v] error: %s\n", file, file, err.Error())
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "remove err:", err.Error()})
			}
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{1, "Files deleted successfully", ""})
	}
}
