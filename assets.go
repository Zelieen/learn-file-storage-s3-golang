package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
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

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	presignedHTTPRequest, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{Bucket: &bucket, Key: &key}, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}
	return presignedHTTPRequest.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	parts := strings.Split(fmt.Sprint(*video.VideoURL), ",")
	if len(parts) != 2 {
		return video, fmt.Errorf("unexpected length of video URL list: found %d, expected 2. Prints VideoURL: %s", len(parts), *video.VideoURL)
	}
	bucket, key := parts[0], parts[1]

	signedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Duration(3)*time.Minute)
	if err != nil {
		return video, err
	}
	video.VideoURL = &signedURL
	return video, nil
}
