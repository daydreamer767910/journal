package util

/*
#cgo pkg-config: libavcodec libavformat libavutil libavfilter
#include "avtransformer.h"
*/
import "C"
import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unsafe"
)

/*
func AddSubtitlesToVideo(inputVideo string, inputSubtitles []string, outputVideo string) error {
	// 将 Go 字符串转换为 C 字符串
	inputVideoC := C.CString(inputVideo)
	outputVideoC := C.CString(outputVideo)
	defer C.free(unsafe.Pointer(inputVideoC))
	defer C.free(unsafe.Pointer(outputVideoC))

	// 将输入字幕文件列表转换为 C 字符串数组
	numSubtitles := len(inputSubtitles)
	inputSubtitlesC := make([]*C.char, numSubtitles)
	for i, subtitle := range inputSubtitles {
		inputSubtitlesC[i] = C.CString(subtitle)
		defer C.free(unsafe.Pointer(inputSubtitlesC[i]))
	}

	// 调用 C 函数
	ret := C.addSubtitlesToVideo(inputVideoC, (**C.char)(unsafe.Pointer(&inputSubtitlesC[0])), C.int(numSubtitles), outputVideoC)
	if ret < 0 {
		return fmt.Errorf("add subtitles fail: %d", ret)
		//return errors.New(fmt.Sprintf("Failed to add subtitles, error code: %d", ret))
	}
	return nil
}
*/

func MergeVideos(inputVideos []string, outputVideo string, options string) error {
	// 将 Go 字符串转换为 C 字符串
	//var options string
	//for i := 0; i < len(inputVideos); i++ {
	//options += fmt.Sprintf("scale=w=4096:h=2160:force_original_aspect_ratio=decrease,")
	//options += fmt.Sprintf("pad=w=4096:h=2160")
	//}
	//for i := 0; i < len(inputVideos); i++ {
	//	options += fmt.Sprintf("[pad%d]", i)
	//}
	//options += fmt.Sprintf("concat=n=%d", len(inputVideos)) + ":v=1:a=0[out],"
	//options = strings.TrimSuffix(options, ",")
	outputVideo += "000.mp4"
	optionC := C.CString(options)
	outputVideoC := C.CString(outputVideo)
	defer C.free(unsafe.Pointer(optionC))
	defer C.free(unsafe.Pointer(outputVideoC))

	// 将输入字幕文件列表转换为 C 字符串数组
	numVideos := len(inputVideos)
	inputVideosC := make([]*C.char, numVideos)
	for i, video := range inputVideos {
		inputVideosC[i] = C.CString(video)
		defer C.free(unsafe.Pointer(inputVideosC[i]))
	}

	// 调用 C 函数
	ret := C.mergeVideos((**C.char)(unsafe.Pointer(&inputVideosC[0])), C.int(numVideos), outputVideoC, optionC)
	if ret < 0 {
		return fmt.Errorf("add subtitles fail: %d", ret)
		//return errors.New(fmt.Sprintf("Failed to add subtitles, error code: %d", ret))
	}
	return nil
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
		filter.WriteString(fmt.Sprintf("[%d]%s[v%d];", i, filter_complex, i))
	}
	for i := 0; i < n; i++ {
		filter.WriteString(fmt.Sprintf("[v%d]", i))
	}

	filter.WriteString(fmt.Sprintf("concat=n=%d:v=1:a=0[out]", n))

	return "[out]"
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
	//for test
	MergeVideos(videoFiles, outputFile, filter.String())

	cmdArgs = append(cmdArgs, outputFile)
	fmt.Printf("ffmpeg %s\n", strings.Join(cmdArgs, " "))
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
func mergeSubTitles_cmd(videoFile string, subTitleFiles []string, outputFile string) error {
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
	cmdArgs = append(cmdArgs, "-map", "0")
	for i := range subTitleFiles {
		cmdArgs = append(cmdArgs, "-map", fmt.Sprintf("%d", i+1))
		cmdArgs = append(cmdArgs, fmt.Sprintf("-metadata:s:s:%d", i), fmt.Sprintf("title=ttl%d", i))
	}

	cmdArgs = append(cmdArgs, outputFile)
	//fmt.Printf("ffmpeg %v\n", cmdArgs)
	_, err := RunCommand("ffmpeg", cmdArgs...)
	if err != nil {
		fmt.Printf("\nfailed to CombineFiles[%s]: %v\n", outputFile, err)
		return err
	}
	return nil
}

/*
func mergeSubTitles(videoFile string, subTitleFiles []string, outputFile string) error {

	err := AddSubtitlesToVideo(videoFile, subTitleFiles, outputFile)
	if err != nil {
		fmt.Println("Error:", err)
	}
	return nil
}
*/
