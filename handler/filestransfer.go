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

type fileInfo struct {
	Name    string
	URL     string
	Size    int64
	Type    string
	ModTime string
	IsDir   bool
}

func GetThumnail(fileName string) string {
	extension := strings.ToLower(filepath.Ext(fileName))

	// 根据文件类型设置不同的缩略图路径
	switch extension {
	case ".pdf":
		return "/static/pdf.png"
	case ".jpg", ".jpeg", ".png", ".gif":
		return "/static/pic.png"
	case ".txt":
		return "/static/txt.png"
	case ".xlsx", ".cvs", ".xls":
		return "/static/excel.png"
	case ".mp4", ".mov":
		return "/static/vidio.png"
	default:
		return "/static/ufo.png"
	}
}

func listFiles(directoryPath string) ([]fileInfo, error) {

	var files []fileInfo

	// 读取目录下的文件和子目录
	entries, err := os.ReadDir(directoryPath)
	if err != nil {
		return nil, err
	}

	// 遍历所有文件和子目录
	for _, entry := range entries {
		// 判断是否为文件
		if entry.IsDir() {
			// 如果是子目录，递归调用 ListFiles
			subDirectoryPath := filepath.Join(directoryPath, entry.Name())
			subFiles, err := listFiles(subDirectoryPath)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		} else {
			// 如果是文件，添加到列表中
			fileinfo, _ := entry.Info()

			modTimeFormatted := fileinfo.ModTime().Format("2006-01-02 15:04:05")

			files = append(files, fileInfo{
				Name:    entry.Name(),
				URL:     filepath.ToSlash(filepath.Join(directoryPath, entry.Name())),
				Size:    fileinfo.Size(),
				Type:    strings.ToLower(filepath.Ext(entry.Name())),
				ModTime: modTimeFormatted,
				IsDir:   false,
			})
		}
	}

	return files, nil
}

func ListFiles(db store.IStore) echo.HandlerFunc {
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
		var upload_path string
		if user.Admin {
			upload_path = "public"
		} else {
			upload_path = filepath.Join("public", "uploads", user.Username)
		}
		//upload_path := filepath.Join("public\\uploads", user.Username)
		files, err := listFiles(upload_path)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "read dir fail", err.Error()})
		}
		return c.Render(http.StatusOK, "filebrowser.html", map[string]interface{}{
			"username": user.Username,
			"Files":    files})
		//return c.JSON(http.StatusOK, jsonHTTPResponse{1, "read dir ok", files})
	}
}

func Upload(db store.IStore) echo.HandlerFunc {
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
		// 解析表单
		form, err := c.MultipartForm()
		if err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{0, "Error parsing form", err})
		}

		// 获取上传的文件
		files := form.File["files[]"]

		upload_path := filepath.Join("public", "uploads", user.Username)

		// 使用 os.MkdirAll 创建目录，包括所有不存在的父目录
		err = os.MkdirAll(upload_path, os.ModePerm)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{0, "create dir fail", err.Error()})
		}

		// 遍历处理每个文件
		for _, file := range files {
			// 打开目标文件
			dst, err := os.Create(filepath.Join(upload_path, file.Filename))
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
			return c.Redirect(http.StatusTemporaryRedirect, util.BasePath+"/login")
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
