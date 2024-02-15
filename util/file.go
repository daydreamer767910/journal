package util

import (
	"errors"
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
	SubTitleFile
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

	".srt": "subtitle/srt",
	".ass": "subtitle/ass",
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
	case "subtitle":
		return SubTitleFile
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

func mergeAudioFiles(audioFiles []string, outputFile string, opts ...map[string]interface{}) error {
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
		return err
	}

	return nil
}

func buildFilterComplex(n int, filter *strings.Builder, opts ...map[string]interface{}) string {
	filter_complex := ""
	for _, opt := range opts {
		value, ok := opt["scale"]
		if ok {
			resolution := value.(string)
			filter_complex = fmt.Sprintf("scale=%s:force_original_aspect_ratio=decrease,pad=%s:(ow-iw)/2:(oh-ih)/2,", resolution, resolution)
		}
		if value, ok := opt["drawtext"]; ok {
			if drawtextOptions, ok := value.(map[string]interface{}); ok {
				text := drawtextOptions["text"].(string)
				size := drawtextOptions["fontsize"].(string)
				color := drawtextOptions["fontcolor"].(string)
				if text != "" && size != "" && color != "" {
					filter_complex = fmt.Sprintf("drawtext=text='%s':x=(w-text_w)/2:y=h-th-40:fontsize=%s:fontcolor=%s", text, size, color)
				}
			}
		}
	}
	// 去除最后一个逗号
	filter_complex = strings.TrimSuffix(filter_complex, ",")

	for i := 0; i < n; i++ {
		filter.WriteString(fmt.Sprintf("[%d:v]%s[v%d];", i, filter_complex, i))
	}
	for i := 0; i < n; i++ {
		filter.WriteString(fmt.Sprintf("[v%d]", i))
	}

	filter.WriteString(fmt.Sprintf("concat=n=%d:v=1:a=0[outv]", n))

	return "[outv]"
}

func scaleImgFiles(imageFiles []string, outputDir string, opts ...map[string]interface{}) ([]string, error) {
	var scaledImgFiles []string
	resolution := "1280:720" // 默认分辨率
	for _, opt := range opts {
		value, ok := opt["scale"]
		if ok {
			resolution = value.(string)
		}
	}
	fmt.Printf("scale imgs [%s]...", resolution)
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
		//fmt.Printf("scale imgs[%s] ok\n", scaledImgFiles[i])
	}
	return scaledImgFiles, nil
}

func mergeImgFiles(outputDir string, outputFile string, opts ...map[string]interface{}) error {
	//ffmpeg -y -framerate 1/5 -i %d.jpg -c:v libx264 -r 30 -pix_fmt yuv420p output.mp4
	durationPerPic := 2.5 //second
	for _, opt := range opts {
		value, ok := opt["duration"]
		if ok {
			durationPerPic, _ = strconv.ParseFloat(value.(string), 64)
		}
	}
	frameRate := 1 / durationPerPic
	fmt.Printf("merge imgs frameRate[%v] duration[%v]", frameRate, durationPerPic)
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

func mergeVideoFiles(videoFiles []string, outputFile string, opts ...map[string]interface{}) error {
	var filter strings.Builder
	cmdArgs := []string{"-y"} // 覆盖输出文件
	for _, videoFile := range videoFiles {
		cmdArgs = append(cmdArgs, "-i", videoFile)
	}
	complexFilterName := buildFilterComplex(len(videoFiles), &filter, opts...)

	cmdArgs = append(cmdArgs, "-filter_complex", filter.String(), "-map", complexFilterName)

	cmdArgs = append(cmdArgs, outputFile)
	//fmt.Printf("ffmpeg %s\n", strings.Join(cmdArgs, " "))
	_, err := RunCommand("ffmpeg", cmdArgs...)
	if err != nil {
		fmt.Printf("failed to Combine video Files[%s]: %v", strings.Join(videoFiles, ","), err)
		return err
	}
	return nil
}

func mergeVideoAudioFile(file1 string, file2 string, outputFile string) error {

	//(时长短的循环播放)
	duration1, err := getVideoDuration(file1)
	if err != nil {
		fmt.Println("getVideoDuration:", err)
		return err
	}
	duration2, err := getVideoDuration(file2)
	if err != nil {
		fmt.Println("getVideoDuration:", err)
		return err
	}
	cmdArgs := []string{"-y"}
	if duration1 > duration2 {
		cmdArgs = append(cmdArgs, "-stream_loop", "-1")
		cmdArgs = append(cmdArgs, "-i", file2)
		cmdArgs = append(cmdArgs, "-i", file1)
		cmdArgs = append(cmdArgs, "-c:v", "copy", "-c:a", "aac", "-strict", "experimental")
		cmdArgs = append(cmdArgs, "-t", fmt.Sprintf("%f", duration1))
	} else if duration1 < duration2 {
		cmdArgs = append(cmdArgs, "-stream_loop", "-1")
		cmdArgs = append(cmdArgs, "-i", file1)
		cmdArgs = append(cmdArgs, "-i", file2)
		cmdArgs = append(cmdArgs, "-c:v", "copy", "-c:a", "aac", "-strict", "experimental")
		cmdArgs = append(cmdArgs, "-t", fmt.Sprintf("%f", duration2))
	} else {
		cmdArgs = append(cmdArgs, "-i", file1)
		cmdArgs = append(cmdArgs, "-i", file2)
		cmdArgs = append(cmdArgs, "-c:v", "copy", "-c:a", "aac", "-strict", "experimental")
	}

	cmdArgs = append(cmdArgs, outputFile)
	// 执行 ffmpeg 命令
	//fmt.Println("ffmpeg", cmdArgs)
	_, err = RunCommand("ffmpeg", cmdArgs...)
	if err != nil {
		fmt.Printf("failed to CombineFiles[%s]: %v", outputFile, err)
		return err
	}
	//fmt.Printf("[%s]created", outputFile)
	return nil
}

func mergeSubTitles(videoFile string, subTitleFiles []string, outputFile string) error {
	//ffmpeg -i 22.mp4 -i subtitle1.srt -i subtitle2.srt
	//-c copy -c:s srt
	//-metadata:s:s:0 language=eng -metadata:s:s:0 title="a"
	//-metadata:s:s:1 language=eng -metadata:s:s:1 title="b"
	//-map 0:v -map 0:a
	//-map 1:s -map 2:s
	//output.mkv
	cmdArgs := []string{"-y", "-i", videoFile}
	for _, subtitleFile := range subTitleFiles {
		cmdArgs = append(cmdArgs, "-i", subtitleFile)
	}
	cmdArgs = append(cmdArgs, "-c", "copy")
	//cmdArgs = append(cmdArgs, "-map 0 -dn -map \"-0:s\" -map \"-0:d\"")
	/*for i := range subTitleFiles {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-map %d:0 -metadata:s:s:%d title=t-%d", i+1, i, i))
	}*/

	cmdArgs = append(cmdArgs, outputFile)
	//fmt.Printf("ffmpeg %v\n", cmdArgs)
	//time.Sleep(2 * time.Second)
	_, err := RunCommand("ffmpeg", cmdArgs...)
	if err != nil {
		fmt.Printf("\nfailed to CombineFiles[%s]: %v\n", outputFile, err)
		return err
	}
	return nil
}

func CombineFiles(files []string, outputDir string, outputFile string, opts ...map[string]interface{}) error {
	var imageFiles []string
	var audioFiles []string
	var videoFiles []string
	var subTitleFiles []string
	var tempFiles []string

	combinedAudioFile := ""
	combinedVideoFile := ""
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
		} else if mType == SubTitleFile {
			subTitleFiles = append(subTitleFiles, filepath.Join(file))
		}
	}
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
	//1.根据参数合成新音频文件
	if len(audioFiles) > 0 {
		combinedAudioFile = filepath.Join(outputDir, "step1.mp3")
		err = mergeAudioFiles(audioFiles, combinedAudioFile, opts...)
		if err != nil {
			fmt.Println("mergeAudioFiles:", err)
			return err
		}
		tempFiles = append(tempFiles, combinedAudioFile)

		fmt.Println("audios merged ok")
	}
	//2.把图片处理成相同分辨率和sar并合成视频
	if len(imageFiles) > 0 {
		scaledImgFiles, err := scaleImgFiles(imageFiles, outputDir, opts...)
		tempFiles = append(tempFiles, scaledImgFiles...)
		if err != nil {
			fmt.Println("scaleImgFiles:", err)
			return err
		}
		combinedVideoFile = filepath.Join(outputDir, "step2.mp4")
		err = mergeImgFiles(outputDir, combinedVideoFile, opts...)
		if err != nil {
			fmt.Println("mergeImgFiles:", err)
			return err
		}
		tempFiles = append(tempFiles, combinedVideoFile)
		//add the merged video file to the list
		videoFiles = append(videoFiles, combinedVideoFile)
		fmt.Println("images merged ok")
	}
	//3.把视频根据分辨率等参数合成一个新视频
	if len(videoFiles) > 0 {
		combinedVideoFile = filepath.Join(outputDir, "step3.mp4")
		err = mergeVideoFiles(videoFiles, combinedVideoFile, opts...)
		if err != nil {
			fmt.Println("mergeVideoFiles:", err)
			return err
		}
		tempFiles = append(tempFiles, combinedVideoFile)
		fmt.Println("videos merged ok")
	}

	//4.把已生成音频和视频进一步合成最终视频
	if combinedVideoFile == "" {
		return errors.New("no img or video input")
	} else if combinedAudioFile == "" {
		//return os.Rename(combinedVideoFile, filepath.Join(outputDir, outputFile))
	} else {
		vaFile := filepath.Join(outputDir, "step4.mp4")
		err = mergeVideoAudioFile(combinedVideoFile, combinedAudioFile, vaFile)
		if err != nil {
			fmt.Println("mergeVideoAudioFile:", err)
			return err
		}
		combinedVideoFile = vaFile
		tempFiles = append(tempFiles, combinedVideoFile)
		fmt.Println("video and autio merged ok")
	}

	//5.给视频添加字幕
	if len(subTitleFiles) > 0 {
		subtitledVideoFile := filepath.Join(outputDir, "step5.mkv")
		err = mergeSubTitles(combinedVideoFile, subTitleFiles, subtitledVideoFile)
		if err != nil {
			fmt.Println("mergeSubTitles:", err)
			return err
		}
		combinedVideoFile = subtitledVideoFile
		//tempFiles = append(tempFiles, combinedVideoFile)
		fmt.Println("subtitles merged ok")
	}

	//6.返回指定文件
	return os.Rename(combinedVideoFile, filepath.Join(outputDir, outputFile))

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
