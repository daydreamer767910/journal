package handler

import (
	"fmt"
	"io"
	"journal/store"
	"journal/util"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/labstack/echo/v4"
)

func CombineFiles(db store.IStore) echo.HandlerFunc {
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
		var request jsonHTTPCombineFiles
		/*request := jsonHTTPCombineFiles{
			Opts:       map[string]interface{}{"scale": "w=1280:h=720", "duration": "1.5"},
		}*/
		if err := c.Bind(&request); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{0, "Bad post data", err.Error()})
		}
		//fmt.Println(request)
		output_path := filepath.Join("public", "works", user.Username)
		err = util.CombineFiles(request.Files, output_path, request.OutputFile, request.Opts.(map[string]interface{}))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "CombineFiles", err.Error()})
		}
		return c.JSON(http.StatusOK, jsonHTTPResponse{1, "combine ok", ""})
	}
}

func ListWorkshop(db store.IStore) echo.HandlerFunc {
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

		fileType := c.QueryParam("type")
		var upload_path string
		if user.Admin {
			upload_path = "public"
		} else {
			upload_path = filepath.Join("public", "works", user.Username)
		}
		nType, _ := strconv.Atoi(fileType)
		files, _ := util.ListFiles(upload_path, nType)

		return c.JSON(http.StatusOK, jsonHTTPResponse{1, "read dir ok", files})
	}
}

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

		fileType := c.QueryParam("type")
		var upload_path string
		if user.Admin {
			upload_path = "public"
		} else {
			upload_path = filepath.Join("public", "uploads", user.Username)
		}
		nType, _ := strconv.Atoi(fileType)
		files, _ := util.ListFiles(upload_path, nType)

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
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{0, "Error parsing form", err.Error()})
		}

		// 获取上传的文件
		files := form.File["files[]"]
		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Println("Failed to get current directory:", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "", err.Error()})

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
			if util.GetMediaType(dst_file) == util.VideoFile {
				var percentages []int
				var durations []int
				for _, cfg := range util.ThumbnailCfg {
					percentages = append(percentages, cfg.PercentPosition)
					durations = append(durations, cfg.Duration)
				}

				util.GenerateVideoThumbnail(dst_file, percentages, durations)
			} else if util.GetMediaType(dst_file) == util.AudioFile {
				util.GenerateAudioThumbnail(dst_file)
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

		if err := util.DeleteFiles(request.Files); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "remove err:", err.Error()})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{1, "Files deleted successfully", ""})
	}
}
