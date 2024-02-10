package util

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	VideoFile = iota
	AudioFile
	ImageFile
	TextFile
	AppFile
	UnknownFile
)

var thumbnailExtension = map[int]string{
	VideoFile: ".mp4",
	AudioFile: ".jpg",
}

var mediaType = map[string]string{
	".mp4":  "video/mp4",
	".mov":  "video/quicktime",
	".avi":  "video/x-msvideo",
	".wmv":  "video/x-ms-wmv",
	".flv":  "video/x-flv",
	".mpeg": "video/mpeg",
	".mpg":  "video/mpeg",
	".mkv":  "video/x-matroska",

	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".flac": "audio/flac",
	".wma":  "audio/x-ms-wma",

	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",

	".pdf":  "application/pdf",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".xls":  "application/vnd.ms-excel",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",

	".txt": "text/plain",
	".csv": "text/csv",
}

type fileInfo struct {
	Name          string
	Thumbnail     string
	ThumbnailType string
	URL           string
	Size          int64
	Type          string
	ModTime       string
	IsDir         bool
}

func GetMediaType(fileName string) int {
	extension := strings.ToLower(filepath.Ext(fileName))
	strType, ok := mediaType[extension]
	if !ok {
		return UnknownFile
	}
	strType = strings.Split(strType, "/")[0]
	switch strType {
	case "video":
		return VideoFile
	case "audio":
		return AudioFile
	case "image":
		return ImageFile
	case "text":
		return TextFile
	case "application":
		return AppFile
	default:
		return UnknownFile
	}
}

// GetVideoDuration 获取视频的长度（持续时间）
func getVideoDuration(videoFile string) (float64, error) {
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

func GenerateAudioThumbnail(inputFile string) error {
	extension := strings.ToLower(filepath.Ext(inputFile))

	thumbnail_path := filepath.Join(filepath.Dir(inputFile), "thumbnail")
	err := os.MkdirAll(thumbnail_path, os.ModePerm)
	if err != nil {
		return err
	}
	base := strings.TrimSuffix(filepath.Base(inputFile), extension)
	outputFile := filepath.Join(thumbnail_path, base+".jpg")

	// 执行 FFmpeg 命令来生成缩略图
	cmdArgs := []string{"-i", inputFile}
	cmdArgs = append(cmdArgs, "-an", "-vcodec", "copy", outputFile)
	//fmt.Println("ffmpeg", cmdArgs)
	_, err = RunCommand("ffmpeg", cmdArgs...)
	if err != nil {
		fmt.Printf("failed to generate thumbnail[%s]: %v", outputFile, err)
		return err
	}
	//fmt.Printf("Thumbnail %s generated for %s\n", outputFile, inputFile)
	return nil
}

func GenerateVideoThumbnail(inputFile string, percentages []int, durations []int) error {
	extension := strings.ToLower(filepath.Ext(inputFile))

	thumbnail_path := filepath.Join(filepath.Dir(inputFile), "thumbnail")
	err := os.MkdirAll(thumbnail_path, os.ModePerm)
	if err != nil {
		return err
	}
	base := strings.TrimSuffix(filepath.Base(inputFile), extension)
	outputFile := filepath.Join(thumbnail_path, base+thumbnailExtension[VideoFile])

	// 获取视频总时长
	duration, err := getVideoDuration(inputFile)
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

func getThumbnail(directoryPath string, fileName string) (Thumbnail string, ThumbnailType string) {
	extension := strings.ToLower(filepath.Ext(fileName))
	base := strings.TrimSuffix(fileName, extension)
	mType := GetMediaType(fileName)
	if mType == UnknownFile {
		Thumbnail = "/assets/ufo.png"
		ThumbnailType = extension
	} else if mType == VideoFile || mType == AudioFile {
		Thumbnail = filepath.Join(directoryPath, "thumbnail", base+thumbnailExtension[mType])
		ThumbnailType = mediaType[thumbnailExtension[mType]]
	} else {
		Thumbnail = filepath.Join(directoryPath, fileName)
		ThumbnailType = mediaType[extension]
	}
	return
}

func CombineFiles(files []string, outputDir string, outputFile string) error {
	var imageFiles []string
	var audioFiles []string
	// 创建输出目录
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		fmt.Println("Failed to create output directory:", err)
		return err
	}

	for _, file := range files {
		file, _ = url.QueryUnescape(file)
		filename := filepath.Base(file)
		mType := GetMediaType(filename)
		if mType == ImageFile {
			imageFiles = append(imageFiles, file)
		} else if mType == AudioFile {
			audioFiles = append(audioFiles, file)
		}
	}

	// 构建FFmpeg命令
	//ffmpeg -y -framerate 0.33 -stream_loop -1 -i image%d.jpg -i audio.mp3 -c:v libx264 -c:a
	//aac -strict experimental output.mp4

	cmdArgs := []string{"-y", "-framerate", "0.33"}
	for _, imageFile := range imageFiles {
		cmdArgs = append(cmdArgs, "-i", imageFile)
	}
	for _, audioFile := range audioFiles {
		cmdArgs = append(cmdArgs, "-i", audioFile)
	}

	cmdArgs = append(cmdArgs, "-c:v", "libx264", "-c:a", "aac", "-strict", "experimental", filepath.Join(outputDir, outputFile))
	// 执行 ffmpeg 命令
	//fmt.Println("ffmpeg", cmdArgs)
	_, err = RunCommand("ffmpeg", cmdArgs...)
	if err != nil {
		fmt.Printf("failed to CombineFiles[%s]: %v", outputFile, err)
		return err
	}
	fmt.Printf("[%s] is created!\n", outputFile)
	return nil
}

func DeleteFiles(files []string) error {
	for _, file := range files {
		file, _ = url.QueryUnescape(file)
		// 实现您的文件删除逻辑
		if err := os.Remove(file); err != nil {
			fmt.Println(err.Error())
			return err
		}
		mType := GetMediaType(file)
		if mType == VideoFile || mType == AudioFile {
			extension := strings.ToLower(filepath.Ext(file))
			base := strings.TrimSuffix(filepath.Base(file), extension)
			path := filepath.Dir(file)
			Thumbnail := filepath.Join(path, "thumbnail", base+thumbnailExtension[mType])
			if err := os.Remove(Thumbnail); err != nil {
				fmt.Println(err.Error())
				return err
			}
		}
	}
	return nil
}

func ListFiles(directoryPath string, fileType int) ([]fileInfo, error) {

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
			subFiles, err := ListFiles(subDirectoryPath, fileType)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		} else {
			if fileType != 255 && fileType != GetMediaType(entry.Name()) {
				continue
			}
			// 如果是文件，添加到列表中
			fileinfo, _ := entry.Info()

			modTimeFormatted := fileinfo.ModTime().Format("2006-01-02 15:04:05")
			thumbnail, thumbnailtype := getThumbnail(directoryPath, entry.Name())
			url := filepath.Join(directoryPath, entry.Name())
			extension := strings.ToLower(filepath.Ext(entry.Name()))
			files = append(files, fileInfo{
				Name:          entry.Name(),
				Thumbnail:     thumbnail,
				ThumbnailType: thumbnailtype,
				URL:           filepath.ToSlash(url),
				Size:          fileinfo.Size(),
				Type:          mediaType[extension],
				ModTime:       modTimeFormatted,
				IsDir:         false,
			})
		}
	}

	return files, nil
}
