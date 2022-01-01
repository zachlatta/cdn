package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	_ "embed"

	"github.com/h2non/filetype"
)

var port, destDir, baseURLRaw, allowedSubnet string
var baseURL *url.URL

func main() {
	port = os.Getenv("PORT")
	destDir = os.Getenv("FS_DEST_DIR")
	baseURLRaw = os.Getenv("BASE_URL")
	allowedSubnet = os.Getenv("ALLOWED_SUBNET")

	if destDir == "" || baseURLRaw == "" {
		fmt.Fprintln(os.Stderr, "FS_DEST_DIR and BASE_URL must both be set")
		os.Exit(1)
	}

	var err error
	baseURL, err = url.Parse(baseURLRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "BASE_URL is not a valid URL")
		os.Exit(1)
	}

	if port == "" {
		defaultPort := "7777"

		log.Println("PORT not set, using default", defaultPort)
		port = defaultPort
	}

	if allowedSubnet == "" {
		defaultSubnet := "100.64.0.0/10"

		log.Println("ALLOWED_SUBNET not set, using default", defaultSubnet)
		allowedSubnet = defaultSubnet
	}

	http.HandleFunc("/", root)
	http.HandleFunc("/upload", upload)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

//go:embed README.md
var readme string

func root(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, readme)
}

func upload(w http.ResponseWriter, r *http.Request) {
	tempFile, err := os.CreateTemp(destDir, "tmp-upload-*")
	if err != nil {
		handleErr(err, r, w, "error writing temporary file to disk", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())

	if _, err := io.Copy(tempFile, r.Body); err != nil {
		handleErr(err, r, w, "error writing uploaded file to disk", http.StatusInternalServerError)
		return
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, tempFile); err != nil {
		handleErr(err, r, w, "error hashing file", http.StatusInternalServerError)
		return
	}

	fileExt, err := inferFileExtension(tempFile.Name())
	if err != nil || fileExt == "unknown" {
		handleErr(err, r, w, "failed to infer file type", http.StatusBadRequest)
		return
	}

	fileHash := hex.EncodeToString(hasher.Sum(nil))
	fileName, err := shortestAvailableTruncatedFilename(2, destDir, fileHash, fileExt)
	if err != nil {
		handleErr(err, r, w, "error generating filename for uploaded file", http.StatusInternalServerError)
		return
	}

	destPath := filepath.Join(destDir, fileName)

	if err := os.Rename(tempFile.Name(), destPath); err != nil {
		handleErr(err, r, w, "error moving temporary file to FS_DEST_DIR", http.StatusInternalServerError)
		return
	}

	fileURLRelative, err := url.Parse("./" + fileName)
	if err != nil {
		handleErr(err, r, w, "internal error constructing file URL", http.StatusInternalServerError)
		return
	}
	fileURL := baseURL.ResolveReference(fileURLRelative)

	fmt.Fprintln(w, fileURL.String())
}

func handleErr(err error, r *http.Request, w http.ResponseWriter, userMsg string, statusCode int) {
	http.Error(w, userMsg, statusCode)
	log.Println("IP:", r.RemoteAddr, "Path:", r.URL.Path, "Error: '", userMsg, "-", err)
}

// find the shortest available truncation of nameToTruncate that doesn't
// yet exist in destDir. returned truncations will have a basename no shorter
// than startingLength.
func shortestAvailableTruncatedFilename(startingLength int, destDir, nameToTruncate, ext string) (string, error) {
	truncated := firstN(nameToTruncate, startingLength) + "." + ext

	path := filepath.Join(destDir, truncated)

	exists, err := fileExists(path)
	if err != nil {
		return "", err
	}

	if exists {
		return shortestAvailableTruncatedFilename(startingLength+1, destDir, nameToTruncate, ext)
	}

	return truncated, nil
}

// from https://stackoverflow.com/a/41604514
func firstN(s string, n int) string {
	i := 0
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}

// from https://stackoverflow.com/a/22467409
func fileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err != nil, err
}

func inferFileExtension(path string) (string, error) {
	t, err := filetype.MatchFile(path)
	if err != nil {
		return "", err
	}

	return t.Extension, nil
}
