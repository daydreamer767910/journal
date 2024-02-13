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

func mergeAudioFiles(audioFiles []string, outputFile string) (float64, error) {
	//ffmpeg -f concat -i audio.txt -c copy output.mp3
	cmdArgs := []string{"-y"}
	for _, file := range audioFiles {
		cmdArgs = append(cmdArgs, "-i", file)
	}
	cmdArgs = append(cmdArgs, "-filter_complex", "concat=n="+fmt.Sprintf("%d", len(audioFiles))+":v=0:a=1[out]")
	cmdArgs = append(cmdArgs, "-map", "[out]", outputFile)

	_, err := RunCommand("ffmpeg", cmdArgs...)
	if err != nil {
		fmt.Printf("failed merge audio file[%s]: %v\n", strings.Join(audioFiles, ","), err)
		return 0, err
	}
	fAudioDuration, err := getVideoDuration(outputFile)
	if err != nil {
		fmt.Println("getVideoDuration:", err)
		return 0, err
	}
	return fAudioDuration, nil
}

func buildFilterComplex(n int, resolution string) string {
	//-filter_complex "
	//[0:v]scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2[v0];
	//[1:v]scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2[v1];
	//[v0][v1]concat=n=2:v=1:a=0[outv]"
	var filter strings.Builder
	for i := 0; i < n; i++ {
		filter.WriteString(fmt.Sprintf("[%d:v]scale=%s:force_original_aspect_ratio=decrease,pad=%s:(ow-iw)/2:(oh-ih)/2[v%d];", i, resolution, resolution, i))
	}
	for i := 0; i < n; i++ {
		filter.WriteString(fmt.Sprintf("[v%d]", i))
	}
	filter.WriteString("concat=n=")
	filter.WriteString(fmt.Sprintf("%d", n))
	filter.WriteString(":v=1:a=0[outv]")
	return filter.String()
}

func scaleImgFiles(imageFiles []string, outputDir string) ([]string, error) {
	var scaledImgFiles []string
	resolution := "1280:720" // 默认分辨率
	//ffmpeg -i 10.jpg -vf "scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2" -q:v 1 out_10.jpg
	for i, imageFile := range imageFiles {
		cmdArgs := []string{"-y", "-i", imageFile, "-vf"}
		cmdArgs = append(cmdArgs, fmt.Sprintf("scale=%s:force_original_aspect_ratio=decrease,pad=%s:(ow-iw)/2:(oh-ih)/2", resolution, resolution))
		scaledImgFiles = append(scaledImgFiles, filepath.Join(outputDir, fmt.Sprintf("%d.jpg", i)))
		cmdArgs = append(cmdArgs, "-q:v", "1", scaledImgFiles[i])
		_, err := RunCommand("ffmpeg", cmdArgs...)
		if err != nil {
			fmt.Printf("failed scale imgs[%s]: %v", scaledImgFiles[i], err)
			return scaledImgFiles, err
		}
	}
	return scaledImgFiles, nil
}

func mergeImgFiles(outputDir string, outputFile string) error {
	//ffmpeg -y -framerate 1/5 -i %d.jpg -c:v libx264 -r 30 -pix_fmt yuv420p output.mp4
	durationPerPic := 2.5 //second
	frameRate := 1 / durationPerPic
	cmdArgs := []string{"-y", "-framerate", fmt.Sprintf("%.04f", frameRate)}
	inputImgs := filepath.Join(outputDir, "%d.jpg")
	cmdArgs = append(cmdArgs, "-i", inputImgs, "-c:v", "libx264", "-r", "30", "-pix_fmt", "yuv420p")
	cmdArgs = append(cmdArgs, outputFile)
	_, err := RunCommand("ffmpeg", cmdArgs...)
	if err != nil {
		fmt.Printf("failed to Combine img Files[%s]: %v", inputImgs, err)
		return err
	}
	return nil
}

func mergeVideoFiles(videoFiles []string, outputFile string) error {
	resolution := "1280:720"  // 默认分辨率
	cmdArgs := []string{"-y"} // 覆盖输出文件
	for _, videoFile := range videoFiles {
		cmdArgs = append(cmdArgs, "-i", videoFile)
	}
	cmdArgs = append(cmdArgs,
		"-filter_complex",
		buildFilterComplex(len(videoFiles), resolution),
		"-map", "[outv]",
		outputFile,
	)
	_, err := RunCommand("ffmpeg", cmdArgs...)
	if err != nil {
		fmt.Printf("failed to Combine video Files[%s]: %v", strings.Join(videoFiles, ","), err)
		return err
	}
	return nil
}

func CombineFiles(files []string, outputDir string, outputFile string) error {
	var imageFiles []string
	var audioFiles []string
	var videoFiles []string
	var tempFiles []string
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
			imageFiles = append(imageFiles, filepath.Join(file))
		} else if mType == AudioFile {
			audioFiles = append(audioFiles, filepath.Join(file))
		} else if mType == VideoFile {
			videoFiles = append(videoFiles, filepath.Join(file))
		}
	}
	//1.先合成音频文件并且读取合成音频时长
	combinedAudioFile := filepath.Join(outputDir, "step1.mp3")
	fTotalAudioDuration, err := mergeAudioFiles(audioFiles, combinedAudioFile)
	if err != nil {
		fmt.Println("mergeAudioFiles:", err)
		return err
	}
	tempFiles = append(tempFiles, combinedAudioFile)
	defer func() {
		//clear the temporary img files
		for _, file := range tempFiles {
			err := os.Remove(file)
			if err != nil {
				fmt.Printf("delete temporary file fail %s: %s\n", file, err)
				continue
			}
			//fmt.Printf("remove %s ok\n", file)
		}
	}()
	fmt.Println("audio merged ok and duration is :", fTotalAudioDuration)
	//2.把图片处理成相同分辨率和sar并合成视频
	scaledImgFiles, err := scaleImgFiles(imageFiles, outputDir)
	tempFiles = append(tempFiles, scaledImgFiles...)
	if err != nil {
		fmt.Println("scaleImgFiles:", err)
		return err
	}
	combinedImgFile := filepath.Join(outputDir, "step2.mp4")
	err = mergeImgFiles(outputDir, combinedImgFile)
	if err != nil {
		fmt.Println("mergeImgFiles:", err)
		return err
	}
	tempFiles = append(tempFiles, combinedImgFile)
	//fmt.Println("image merged ok")
	videoFiles = append(videoFiles, combinedImgFile)
	//3.把多个视频合成一个
	if len(videoFiles) > 1 {
		combinedImgFile = filepath.Join(outputDir, "step3.mp4")
		err = mergeVideoFiles(videoFiles, combinedImgFile)
		if err != nil {
			fmt.Println("mergeVideoFiles:", err)
			return err
		}
		tempFiles = append(tempFiles, combinedImgFile)
		//fmt.Println("video merged ok")
	}
	//4.把已生成音频和视频进一步合成最终视频(时长以音频为准,视频循环播放)
	//ffmpeg -stream_loop -1 -i output.mp4 -i audio.mp3 -c:v copy -c:a aac -strict experimental -t duration output_with_audio.mp4
	cmdArgs := []string{"-y", "-stream_loop", "-1"}
	cmdArgs = append(cmdArgs, "-i", combinedImgFile)
	cmdArgs = append(cmdArgs, "-i", combinedAudioFile)
	cmdArgs = append(cmdArgs, "-c:v", "copy", "-c:a", "aac", "-strict", "experimental")
	cmdArgs = append(cmdArgs, "-t", fmt.Sprintf("%f", fTotalAudioDuration))
	cmdArgs = append(cmdArgs, filepath.Join(outputDir, outputFile))
	// 执行 ffmpeg 命令
	//fmt.Println("ffmpeg", cmdArgs)
	_, err = RunCommand("ffmpeg", cmdArgs...)
	if err != nil {
		fmt.Printf("failed to CombineFiles[%s]: %v", outputFile, err)
		return err
	}
	fmt.Printf("[%s] is created!\n", outputFile)
	//removeTempFiles(tempFiles)
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
