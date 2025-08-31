package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	//the upload here
	const maxMemory = 10 << 20 // bit shift from 10 to 10 MB
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not parse form", err)
		return
	}

	// "thumbnail" should match the HTML form input name
	fileUpload, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer fileUpload.Close()

	// read the data
	mediaType := header.Header.Get("Content-Type")
	extension := strings.Split(mediaType, "/")[1]

	// call metadata from database
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Found no matching video", err)
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not the owner of the video", err)
	}

	// create a name for the thumbnail
	bytes := make([]byte, 32)
	rand.Read(bytes)
	thumbnailName := base64.RawURLEncoding.EncodeToString(bytes)

	// create a file
	filePath := filepath.Join(cfg.assetsRoot, thumbnailName)
	assetFile, err := os.Create(fmt.Sprintf("%s.%s", filePath, extension))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Could not create file", err)
	}
	defer assetFile.Close()

	_, err = io.Copy(assetFile, fileUpload)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Could not copy file to system", err)
	}

	// update the thumbnail in the database
	assetURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, thumbnailName, extension)
	video.ThumbnailURL = &assetURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update thumbnail in database", err)
	}

	respondWithJSON(w, http.StatusOK, video)
}
