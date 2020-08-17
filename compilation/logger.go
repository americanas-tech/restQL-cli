package compilation

import (
	"fmt"
	"log"
)

func LogInfo(pattern string, args ...interface{}) {
	msg := fmt.Sprintf("[INFO] %s\n", pattern)
	log.Printf(msg, args...)
}

func LogWarn(pattern string, args ...interface{}) {
	msg := fmt.Sprintf("[WARN] %s\n", pattern)
	log.Printf(msg, args...)
}

func LogError(pattern string, args ...interface{}) {
	msg := fmt.Sprintf("[ERROR] %s\n", pattern)
	log.Printf(msg, args...)
}
