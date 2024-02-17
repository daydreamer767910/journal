package util

/*
#cgo pkg-config: libavcodec libavformat libavutil
#include "avtransformer.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

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
