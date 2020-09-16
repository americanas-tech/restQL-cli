package restql

import (
	"fmt"
	"log"
)

func logInfo(pattern string, args ...interface{}) {
	msg := fmt.Sprintf("[INFO] %s\n", pattern)
	log.Printf(msg, args...)
}

func logWarn(pattern string, args ...interface{}) {
	msg := fmt.Sprintf("[WARN] %s\n", pattern)
	log.Printf(msg, args...)
}

func logError(pattern string, args ...interface{}) {
	msg := fmt.Sprintf("[ERROR] %s\n", pattern)
	log.Printf(msg, args...)
}
