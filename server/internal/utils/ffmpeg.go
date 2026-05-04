package utils

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

func RunFFmpeg(args []string) error {
	cmdLine := "ffmpeg " + strings.Join(args, " ")
	log.Printf("[ffmpeg] running: %s", cmdLine)
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("[ffmpeg] command failed: %v", err)
		return err
	}
	return nil
}
