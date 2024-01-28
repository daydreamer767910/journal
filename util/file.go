package util

import (
	"embed"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type fileInfo struct {
	Name     string
	Thumnail string
	URL      string
	Size     int64
	Type     string
	ModTime  string
	IsDir    bool
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

func ListFiles(directoryPath string) ([]fileInfo, error) {

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
			subFiles, err := ListFiles(subDirectoryPath)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		} else {
			// 如果是文件，添加到列表中
			fileinfo, _ := entry.Info()

			modTimeFormatted := fileinfo.ModTime().Format("2006-01-02 15:04:05")

			files = append(files, fileInfo{
				Name:     entry.Name(),
				Thumnail: GetThumnail(entry.Name()),
				URL:      filepath.ToSlash(filepath.Join(directoryPath, entry.Name())),
				Size:     fileinfo.Size(),
				Type:     strings.ToLower(filepath.Ext(entry.Name())),
				ModTime:  modTimeFormatted,
				IsDir:    false,
			})
		}
	}

	return files, nil
}

// parseTemplates 解析嵌入式文件系统中的模板文件
func parseTemplates(fs embed.FS, pattern string) (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	// 获取模板文件列表
	files, err := fs.ReadDir("templates")
	if err != nil {
		return nil, err
	}

	// 解析每个模板文件
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".html") {
			tmpl, err := template.ParseFS(fs, "templates/"+file.Name())
			if err != nil {
				return nil, err
			}
			templates[file.Name()] = tmpl
		}
	}

	return templates, nil
}

func StringFromEmbedFile(embed fs.FS, filename string) (string, error) {
	file, err := embed.Open(filename)
	if err != nil {
		return "", err
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
