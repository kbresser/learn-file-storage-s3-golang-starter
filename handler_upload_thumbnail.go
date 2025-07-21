package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

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

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, headers, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()
	mediaType, _, err := mime.ParseMediaType(headers.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "invalid file type", err)
		return
	}

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Bad request", err)
		return
	}

	if userID != videoData.CreateVideoParams.UserID {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	thumbExt, err := mime.ExtensionsByType(mediaType)
	if err != nil || len(thumbExt) == 0 {
		respondWithError(w, http.StatusBadRequest, "Could not get extention", err)
		return
	}

	input := make([]byte, 32)
	rand.Read(input)
	encodedPath := base64.RawURLEncoding.EncodeToString(input)

	path := filepath.Join(cfg.assetsRoot, encodedPath+thumbExt[0])
	imageFile, err := os.Create(path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not create file", err)
		return
	}

	if _, err := io.Copy(imageFile, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not copy file", err)
		return
	}

	ThumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, encodedPath+thumbExt[0])
	videoData.ThumbnailURL = &ThumbnailURL

	err = cfg.db.UpdateVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoData)
}
