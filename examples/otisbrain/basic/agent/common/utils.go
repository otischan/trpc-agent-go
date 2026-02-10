package common

import (
	"os"

	"github.com/sirupsen/logrus"
)

// WriteCriticalEvent 写入关键事件到日志
func WriteCriticalEvent(namespace, objType, objName, eventType, message string, logger *logrus.Logger) {
	logger.Errorf("CRITICAL_EVENT_LOG - Namespace: %s, Type: %s, Name: %s, Event: %s, Message: %s",
		namespace, objType, objName, eventType, message)

	// Write in format suitable for aggregation
	logger.WithFields(logrus.Fields{
		"namespace": namespace,
		"objType":   objType,
		"objName":   objName,
		"eventType": eventType,
		"message":   message,
	}).Error("CRITICAL_EVENT")
}

// OpenFile 是一个辅助函数，用于打开文件
func OpenFile(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
}
