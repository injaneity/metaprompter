package artifacts

import (
	"fmt"
	"os"
	"time"
)

func BuildLogHeader(generatedAt time.Time) string {
	return fmt.Sprintf("# Execution Log\n\nSession Started: %s\n\n", generatedAt.UTC().Format(time.RFC3339))
}

func ResetLog(path string, generatedAt time.Time) error {
	return os.WriteFile(path, []byte(BuildLogHeader(generatedAt)), 0o644)
}

func FormatLogEvent(timestamp time.Time, issueNumber int, action string) string {
	return fmt.Sprintf("%s [%d] %s", timestamp.UTC().Format(time.RFC3339), issueNumber, action)
}

func AppendLogEvent(path string, timestamp time.Time, issueNumber int, action string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	line := FormatLogEvent(timestamp, issueNumber, action) + "\n"
	_, err = f.WriteString(line)
	return err
}
