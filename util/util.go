package util

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/gommon/log"
)

func RandomString(length int) string {
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func ParseBasePath(basePath string) string {
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	if strings.HasSuffix(basePath, "/") {
		basePath = strings.TrimSuffix(basePath, "/")
	}
	return basePath
}

func ParseLogLevel(lvl string) (log.Lvl, error) {
	switch strings.ToLower(lvl) {
	case "debug":
		return log.DEBUG, nil
	case "info":
		return log.INFO, nil
	case "warn":
		return log.WARN, nil
	case "error":
		return log.ERROR, nil
	case "off":
		return log.OFF, nil
	default:
		return log.DEBUG, fmt.Errorf("not a valid log level: %s", lvl)
	}
}

func LookupEnvOrString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func RunCommand(cmd string, arg ...string) ([]byte, error) {
	// 创建日志目录
	logsDir := "logs"
	err := os.MkdirAll(logsDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %v", err)
	}

	// 构建日志文件路径
	logFileName := fmt.Sprintf("cmd-%s.log", time.Now().Format("2006-01-02-15-04-05.000"))
	logFilePath := filepath.Join(logsDir, logFileName)

	// 打开日志文件用于写入
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %v", err)
	}
	defer logFile.Close()

	// 创建 cmd 对象并设置输出重定向
	command := exec.Command(cmd, arg...)
	command.Stdout = logFile
	command.Stderr = logFile
	//fmt.Println(cmd, arg)
	// 执行命令
	err = command.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run command: %v", err)
	}

	// 在这里读取日志文件内容并返回
	output, err := os.ReadFile(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %v", err)
	}

	return output, nil
}
