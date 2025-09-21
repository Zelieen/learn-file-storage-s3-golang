package main

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"os/exec"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getVideoAspectRatio(filePath string) (string, error) {
	FetchCommand := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var buf []byte
	FetchBuffer := bytes.NewBuffer(buf)
	FetchCommand.Stdout = FetchBuffer

	err := FetchCommand.Run()
	if err != nil {
		return "other", err
	}

	type VideoProbe struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		}
	}
	var VideoMeasures VideoProbe
	err = json.Unmarshal(FetchBuffer.Bytes(), &VideoMeasures)
	if err != nil {
		return "other", err
	}

	ratio := float64(VideoMeasures.Streams[0].Width) / float64(VideoMeasures.Streams[0].Height)

	if int(math.Round(ratio*9)) == 16 {
		return "16:9", nil
	} else if int(math.Round(ratio*16)) == 9 {
		return "9:16", nil
	} else {
		return "other", nil
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	processedPath := filePath + ".processing"
	process := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", processedPath)

	err := process.Run()

	return processedPath, err
}
