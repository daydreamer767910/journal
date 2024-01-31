package util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const thumbnailExtension = ".mp4"

type fileInfo struct {
	Name      string
	Thumbnail string
	URL       string
	Size      int64
	Type      string
	ModTime   string
	IsDir     bool
}

// GetVideoDuration 获取视频的长度（持续时间）
func GetVideoDuration(videoFile string) (float64, error) {
	output, err := RunCommand("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", "-i", videoFile)
	if err != nil {
		return 0, fmt.Errorf("获取视频长度失败: %v", err)
	}
	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("解析视频长度失败: %v", err)
	}

	return duration, nil
}

func GenerateThumbnail(inputFile string, percentages []int, durations []int) error {
	extension := strings.ToLower(filepath.Ext(inputFile))
	switch extension {
	case ".mp4", ".mov", ".avi", ".wmv", ".flv", ".mpeg", ".mpg", ".mkv":
		break
		//fmt.Printf("inputFile %s\n", inputFile)
	default:
		//fmt.Printf("inputFile %s no need thumbnail[%s]\n", inputFile, extension)
		return errors.New("no need to generate thumbnail for " + extension)
	}
	thumbnail_path := filepath.Join(filepath.Dir(inputFile), "thumbnail")
	err := os.MkdirAll(thumbnail_path, os.ModePerm)
	if err != nil {
		return err
	}
	base := strings.TrimSuffix(filepath.Base(inputFile), extension)
	outputFile := filepath.Join(thumbnail_path, base+thumbnailExtension)

	// 获取视频总时长
	duration, err := GetVideoDuration(inputFile)
	if err != nil {
		fmt.Println("错误:", err)
		return err
	}
	// 生成截图的时间点
	var filters []string
	for i, percentage := range percentages {
		ts := percentage * int(duration) / 100
		filters = append(filters, fmt.Sprintf("between(n,%d,%d)", ts*30, (ts+durations[i])*30)) // 每秒30帧的视频
	}
	selectFilter := strings.Join(filters, "+")
	setptsFilter := "N/FRAME_RATE/TB"
	// 执行 FFmpeg 命令来生成缩略图
	cmdArgs := []string{"-i", inputFile}
	cmdArgs = append(cmdArgs, "-vf", fmt.Sprintf("select='%s',setpts='%s'", selectFilter, setptsFilter))
	cmdArgs = append(cmdArgs, "-an", outputFile)
	//fmt.Println("ffmpeg", cmdArgs)
	_, err = RunCommand("ffmpeg", cmdArgs...)
	if err != nil {
		fmt.Printf("failed to generate thumbnail[%s]: %v", outputFile, err)
		return err
	}
	//fmt.Printf("Thumbnail %s generated for %s\n", outputFile, inputFile)
	return nil
}

func GetThumbnail(directoryPath string, fileName string) string {
	extension := strings.ToLower(filepath.Ext(fileName))
	base := strings.TrimSuffix(fileName, extension)

	// 根据文件类型设置不同的缩略图路径
	switch extension {
	case ".pdf":
		return "/static/pdf.png"
	case ".jpg", ".jpeg", ".png", ".gif":
		return filepath.Join(directoryPath, fileName)
	case ".txt":
		return "/static/txt.png"
	case ".xlsx", ".cvs", ".xls":
		return "/static/excel.png"
	case ".mp4", ".mov", ".avi", ".wmv", ".flv", ".mpeg", ".mpg", ".mkv":
		return filepath.Join(directoryPath, "thumbnail", base+thumbnailExtension)
		//return filepath.Join(directoryPath, fileName)
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
			if entry.Name() == "thumbnail" {
				//ignore the folder of thumbnail
				continue
			}
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
				Name:      entry.Name(),
				Thumbnail: filepath.ToSlash(GetThumbnail(directoryPath, entry.Name())),
				URL:       filepath.ToSlash(filepath.Join(directoryPath, entry.Name())),
				Size:      fileinfo.Size(),
				Type:      strings.ToLower(filepath.Ext(entry.Name())),
				ModTime:   modTimeFormatted,
				IsDir:     false,
			})
		}
	}

	return files, nil
}
