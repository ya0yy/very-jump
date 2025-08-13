package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// SessionRecorder 会话录制器
type SessionRecorder struct {
	sessionID   string
	filePath    string
	file        *os.File
	startTime   time.Time
	mutex       sync.Mutex
	isRecording bool
	width       int
	height      int
}

// AsciinemaHeader asciinema文件头
type AsciinemaHeader struct {
	Version   int    `json:"version"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Timestamp int64  `json:"timestamp"`
	Command   string `json:"command,omitempty"`
	Title     string `json:"title,omitempty"`
}

// AsciinemaEvent asciinema事件
type AsciinemaEvent struct {
	Time float64 `json:"time"`
	Type string  `json:"type"`
	Data string  `json:"data"`
}

// NewSessionRecorder 创建新的会话录制器
func NewSessionRecorder(sessionID, filePath string, width, height int) *SessionRecorder {
	return &SessionRecorder{
		sessionID: sessionID,
		filePath:  filePath,
		startTime: time.Now(),
		width:     width,
		height:    height,
	}
}

// Start 开始录制
func (r *SessionRecorder) Start() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.isRecording {
		return fmt.Errorf("recording already started")
	}

	// 创建录制文件
	file, err := os.Create(r.filePath)
	if err != nil {
		return fmt.Errorf("failed to create recording file: %v", err)
	}

	r.file = file
	r.isRecording = true

	// 写入asciinema头部
	header := AsciinemaHeader{
		Version:   2,
		Width:     r.width,
		Height:    r.height,
		Timestamp: r.startTime.Unix(),
		Title:     fmt.Sprintf("Terminal Session %s", r.sessionID),
		Command:   "ssh",
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return fmt.Errorf("failed to marshal header: %v", err)
	}

	if _, err := r.file.Write(append(headerBytes, '\n')); err != nil {
		return fmt.Errorf("failed to write header: %v", err)
	}

	log.Printf("Started recording for session %s to file %s", r.sessionID, r.filePath)
	return nil
}

// WriteOutput 录制输出数据
func (r *SessionRecorder) WriteOutput(data []byte) error {
	if !r.isRecording || r.file == nil {
		return nil
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 处理ttyd消息格式，剥离消息类型前缀
	var outputData string
	if len(data) > 0 {
		// ttyd消息格式：第一个字符是消息类型，'0'=OUTPUT, '1'=SET_WINDOW_TITLE, '2'=SET_PREFERENCES
		switch data[0] {
		case '0': // OUTPUT - 终端输出数据
			if len(data) > 1 {
				outputData = string(data[1:]) // 去除'0'前缀
			}
		case '1': // SET_WINDOW_TITLE - 忽略窗口标题设置
			return nil
		case '2': // SET_PREFERENCES - 忽略偏好设置
			return nil
		default: // 其他情况，直接记录原始数据
			outputData = string(data)
		}
	} else {
		return nil
	}

	// 如果没有实际输出数据，不记录
	if outputData == "" {
		return nil
	}

	// 计算相对时间戳
	elapsed := time.Since(r.startTime).Seconds()

	// 创建事件
	event := []interface{}{
		elapsed,
		"o", // output
		outputData,
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	if _, err := r.file.Write(append(eventBytes, '\n')); err != nil {
		return fmt.Errorf("failed to write event: %v", err)
	}

	return nil
}

// WriteInput 录制输入数据
func (r *SessionRecorder) WriteInput(data []byte) error {
	if !r.isRecording || r.file == nil {
		return nil
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 处理ttyd消息格式，剥离消息类型前缀
	var inputData string
	if len(data) > 0 {
		// ttyd客户端输入消息格式：'0'=INPUT, '1'=RESIZE_TERMINAL
		switch data[0] {
		case '0': // INPUT - 用户输入数据
			if len(data) > 1 {
				inputData = string(data[1:]) // 去除'0'前缀
			}
		case '1': // RESIZE_TERMINAL - 忽略终端大小调整
			return nil
		default: // 其他情况，直接记录原始数据
			inputData = string(data)
		}
	} else {
		return nil
	}

	// 如果没有实际输入数据，不记录
	if inputData == "" {
		return nil
	}

	// 计算相对时间戳
	elapsed := time.Since(r.startTime).Seconds()

	// 创建事件
	event := []interface{}{
		elapsed,
		"i", // input
		inputData,
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	if _, err := r.file.Write(append(eventBytes, '\n')); err != nil {
		return fmt.Errorf("failed to write event: %v", err)
	}

	return nil
}

// Stop 停止录制
func (r *SessionRecorder) Stop() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.isRecording || r.file == nil {
		return nil
	}

	r.isRecording = false

	if err := r.file.Close(); err != nil {
		log.Printf("Failed to close recording file: %v", err)
		return err
	}

	log.Printf("Stopped recording for session %s", r.sessionID)
	return nil
}

// IsRecording 检查是否正在录制
func (r *SessionRecorder) IsRecording() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.isRecording
}

// GetFilePath 获取录制文件路径
func (r *SessionRecorder) GetFilePath() string {
	return r.filePath
}
