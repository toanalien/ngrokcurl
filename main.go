package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	uploadDir   = "./uploads"
	maxFileSize = 100 * 1024 * 1024 // 100MB
	port        = "8080"
)

type FileInfo struct {
	ID       string    `json:"id"`
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
	UploadAt time.Time `json:"upload_at"`
}

func main() {
	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}

	// Setup routes
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/files/", handleDownload)
	http.HandleFunc("/health", handleHealth)

	log.Printf("Server starting on port %s...", port)
	log.Printf("Upload directory: %s", uploadDir)
	log.Printf("Max file size: %d MB", maxFileSize/(1024*1024))

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <title>File Transfer Service</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        .section { margin: 20px 0; padding: 20px; background: #f5f5f5; border-radius: 5px; }
        code { background: #e0e0e0; padding: 2px 5px; border-radius: 3px; }
        pre { background: #2d2d2d; color: #f8f8f8; padding: 15px; border-radius: 5px; overflow-x: auto; }
    </style>
</head>
<body>
    <h1>📁 File Transfer Service</h1>
    <p>A simple service for uploading and sharing files, similar to transfer.sh</p>
    
    <div class="section">
        <h2>Upload a File</h2>
        <p>Using curl:</p>
        <pre>curl -F "file=@yourfile.pdf" http://` + getHost(r) + `/upload</pre>
        
        <p>Or use the form below:</p>
        <form action="/upload" method="post" enctype="multipart/form-data">
            <input type="file" name="file" required>
            <button type="submit">Upload</button>
        </form>
    </div>
    
    <div class="section">
        <h2>Download a File</h2>
        <p>Using curl:</p>
        <pre>curl http://` + getHost(r) + `/files/{file-id} -o downloaded-file</pre>
        
        <p>Or open in browser:</p>
        <pre>http://` + getHost(r) + `/files/{file-id}</pre>
    </div>
    
    <div class="section">
        <h2>Features</h2>
        <ul>
            <li>Max file size: 100 MB</li>
            <li>Files stored locally</li>
            <li>Unique IDs for each upload</li>
            <li>Simple REST API</li>
        </ul>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form with max memory
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "File too large or invalid form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Generate unique file ID
	fileID := generateFileID()

	// Save file with ID prefix and original filename
	fileName := fileID + "_" + header.Filename
	filePath := filepath.Join(uploadDir, fileName)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		log.Printf("Error creating file: %v", err)
		return
	}
	defer dst.Close()

	// Copy uploaded file to destination
	size, err := io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		log.Printf("Error saving file: %v", err)
		return
	}

	// Build download URL
	downloadURL := fmt.Sprintf("http://%s/files/%s", getHost(r), fileID)

	log.Printf("File uploaded: %s (%.2f MB)", header.Filename, float64(size)/(1024*1024))

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	response := fmt.Sprintf(`{"id":"%s","filename":"%s","size":%d,"url":"%s"}`,
		fileID, header.Filename, size, downloadURL)
	w.Write([]byte(response))
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract file ID from URL path
	fileID := r.URL.Path[len("/files/"):]
	if fileID == "" {
		http.Error(w, "File ID required", http.StatusBadRequest)
		return
	}

	// Find file with this ID (try different extensions)
	var filePath string
	var found bool

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		http.Error(w, "Failed to read uploads directory", http.StatusInternalServerError)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			// Check if filename starts with the fileID
			if len(name) >= len(fileID) && name[:len(fileID)] == fileID {
				filePath = filepath.Join(uploadDir, name)
				found = true
				break
			}
		}
	}

	if !found {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Failed to get file info", http.StatusInternalServerError)
		return
	}

	// Extract original filename (remove ID prefix)
	originalFilename := filepath.Base(filePath)
	if idx := len(fileID) + 1; len(originalFilename) > idx {
		originalFilename = originalFilename[idx:] // Remove "fileID_" prefix
	}

	// Set headers with original filename
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", originalFilename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Stream file to response
	io.Copy(w, file)

	log.Printf("File downloaded: %s", filepath.Base(filePath))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","service":"file-transfer"}`))
}

func generateFileID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if random fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)[:12]
}

func getHost(r *http.Request) string {
	host := r.Host
	if host == "" {
		host = "localhost:" + port
	}
	return host
}
