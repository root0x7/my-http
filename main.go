package main

import (
	"bufio"
	"fmt"
	_ "io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	_ "strconv"
	"strings"
	"syscall"
	"time"
)

const (
	DefaultPort     = "8080"
	DocumentRoot    = "./www"
	ServerName      = "SimpleHTTP/1.0"
	MaxRequestSize  = 8192
	ReadTimeout     = 30 * time.Second
	WriteTimeout    = 30 * time.Second
)

const (
	StatusOK                  = "200 OK"
	StatusNotFound           = "404 Not Found"
	StatusMethodNotAllowed   = "405 Method Not Allowed"
	StatusInternalServerError = "500 Internal Server Error"
	StatusBadRequest         = "400 Bad Request"
)

var mimeTypes = map[string]string{
	".html": "text/html",
	".htm":  "text/html",
	".css":  "text/css",
	".js":   "application/javascript",
	".json": "application/json",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".ico":  "image/x-icon",
	".txt":  "text/plain",
	".pdf":  "application/pdf",
	".zip":  "application/zip",
}

type ServerStats struct {
	TotalRequests     int64
	SuccessfulRequests int64
	ErrorRequests     int64
	StartTime         time.Time
}

type HTTPRequest struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
}

type HTTPResponse struct {
	Status      string
	Headers     map[string]string
	Body        []byte
	ContentType string
}

type Server struct {
	Port     string
	Root     string
	Stats    *ServerStats
	listener net.Listener
}

func NewServer(port, root string) *Server {
	return &Server{
		Port:  port,
		Root:  root,
		Stats: &ServerStats{StartTime: time.Now()},
	}
}

func (s *Server) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", ":"+s.Port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %v", s.Port, err)
	}

	log.Printf("SimpleHTTP Server started on port %s", s.Port)
	log.Printf("Document root: %s", s.Root)
	log.Println("Press Ctrl+C to stop")

	if err := os.MkdirAll(s.Root, 0755); err != nil {
		log.Printf("Warning: Could not create document root: %v", err)
	}

	go s.handleShutdown()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}

	return nil
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(ReadTimeout))
	conn.SetWriteDeadline(time.Now().Add(WriteTimeout))

	log.Printf("Connection from %s", conn.RemoteAddr())

	request, err := s.parseRequest(conn)
	if err != nil {
		s.sendErrorResponse(conn, StatusBadRequest, "Bad Request")
		s.Stats.ErrorRequests++
		log.Printf("Error parsing request: %v", err)
		return
	}

	s.Stats.TotalRequests++

	response := s.handleRequest(request)

	if err := s.sendResponse(conn, response); err != nil {
		log.Printf("Error sending response: %v", err)
		s.Stats.ErrorRequests++
		return
	}

	if response.Status == StatusOK {
		s.Stats.SuccessfulRequests++
	} else {
		s.Stats.ErrorRequests++
	}

	s.logRequest(request, response.Status)
}

func (s *Server) parseRequest(conn net.Conn) (*HTTPRequest, error) {
	reader := bufio.NewReader(conn)

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("error reading request line: %v", err)
	}

	parts := strings.Fields(strings.TrimSpace(requestLine))
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid request line format")
	}

	request := &HTTPRequest{
		Method:  parts[0],
		Path:    parts[1],
		Version: parts[2],
		Headers: make(map[string]string),
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("error reading headers: %v", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break 
		}

		headerParts := strings.SplitN(line, ":", 2)
		if len(headerParts) == 2 {
			key := strings.TrimSpace(headerParts[0])
			value := strings.TrimSpace(headerParts[1])
			request.Headers[strings.ToLower(key)] = value
		}
	}

	return request, nil
}

func (s *Server) handleRequest(request *HTTPRequest) *HTTPResponse {

	if request.Method != "GET" {
		return s.createErrorResponse(StatusMethodNotAllowed, "Method Not Allowed")
	}

	if !s.isSafePath(request.Path) {
		return s.createErrorResponse(StatusNotFound, "Not Found")
	}

	filePath := filepath.Join(s.Root, request.Path)

	if strings.HasSuffix(request.Path, "/") {
		filePath = filepath.Join(filePath, "index.html")
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil || fileInfo.IsDir() {
		return s.createErrorResponse(StatusNotFound, "Not Found")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return s.createErrorResponse(StatusInternalServerError, "Internal Server Error")
	}

	return &HTTPResponse{
		Status:      StatusOK,
		ContentType: s.getMimeType(filePath),
		Body:        content,
		Headers:     make(map[string]string),
	}
}

func (s *Server) sendResponse(conn net.Conn, response *HTTPResponse) error {

	headers := fmt.Sprintf("HTTP/1.1 %s\r\n", response.Status)
	headers += fmt.Sprintf("Server: %s\r\n", ServerName)
	headers += fmt.Sprintf("Date: %s\r\n", time.Now().UTC().Format(time.RFC1123))
	headers += fmt.Sprintf("Content-Type: %s\r\n", response.ContentType)
	headers += fmt.Sprintf("Content-Length: %d\r\n", len(response.Body))
	headers += "Connection: close\r\n"

	for key, value := range response.Headers {
		headers += fmt.Sprintf("%s: %s\r\n", key, value)
	}

	headers += "\r\n"

	if _, err := conn.Write([]byte(headers)); err != nil {
		return err
	}

	if len(response.Body) > 0 {
		if _, err := conn.Write(response.Body); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) sendErrorResponse(conn net.Conn, status, message string) {
	response := s.createErrorResponse(status, message)
	s.sendResponse(conn, response)
}

func (s *Server) createErrorResponse(status, message string) *HTTPResponse {
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 50px; }
        h1 { color: #d32f2f; }
        hr { border: none; border-top: 1px solid #ccc; }
        .footer { font-style: italic; color: #666; }
    </style>
</head>
<body>
    <h1>%s</h1>
    <p>%s</p>
    <hr>
    <div class="footer">%s</div>
</body>
</html>`, status, status, message, ServerName)

	return &HTTPResponse{
		Status:      status,
		ContentType: "text/html",
		Body:        []byte(body),
		Headers:     make(map[string]string),
	}
}

func (s *Server) getMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	if mimeType, exists := mimeTypes[ext]; exists {
		return mimeType
	}
	return "application/octet-stream"
}

func (s *Server) isSafePath(path string) bool {
	return !strings.Contains(path, "..") && !strings.Contains(path, "~")
}

func (s *Server) logRequest(request *HTTPRequest, status string) {
	log.Printf("[%s] %s %s - %s",
		time.Now().Format("2006/01/02 15:04:05"),
		request.Method,
		request.Path,
		status)
}

func (s *Server) printStats() {
	uptime := time.Since(s.Stats.StartTime)
	successRate := float64(0)
	if s.Stats.TotalRequests > 0 {
		successRate = float64(s.Stats.SuccessfulRequests) / float64(s.Stats.TotalRequests) * 100
	}

	fmt.Println("\n=== Server Statistics ===")
	fmt.Printf("Uptime: %v\n", uptime.Round(time.Second))
	fmt.Printf("Total requests: %d\n", s.Stats.TotalRequests)
	fmt.Printf("Successful requests: %d\n", s.Stats.SuccessfulRequests)
	fmt.Printf("Error requests: %d\n", s.Stats.ErrorRequests)
	fmt.Printf("Success rate: %.1f%%\n", successRate)
	fmt.Println("========================")
}

func (s *Server) handleShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\nShutting down server...")

	if s.listener != nil {
		s.listener.Close()
	}

	s.printStats()
	os.Exit(0)
}

func setupSampleWebsite() {
	if err := os.MkdirAll(DocumentRoot, 0755); err != nil {
		log.Printf("Error creating document root: %v", err)
		return
	}

	indexHTML := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SimpleHTTP Server</title>
    <link rel="stylesheet" href="/style.css">
</head>
<body>
    <div class="container">
        <h1>🚀 Welcome to SimpleHTTP Server!</h1>
        <p>Your Go web server is working perfectly!</p>

        <div class="features">
            <h2>Features:</h2>
            <ul>
                <li>✅ HTTP/1.1 Support</li>
                <li>✅ Static File Serving</li>
                <li>✅ MIME Type Detection</li>
                <li>✅ Security Protection</li>
                <li>✅ Request Logging</li>
                <li>✅ Server Statistics</li>
            </ul>
        </div>

        <nav>
            <h2>Test Pages:</h2>
            <ul>
                <li><a href="/test.html">Test Page</a></li>
                <li><a href="/api.json">JSON API</a></li>
            </ul>
        </nav>
    </div>
    <script src="/app.js"></script>
</body>
</html>`

	testHTML := `<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
    <link rel="stylesheet" href="/style.css">
</head>
<body>
    <div class="container">
        <h1>Test Page</h1>
        <p>This is a test page to verify the server is working correctly.</p>
        <a href="/">← Back to Home</a>
    </div>
</body>
</html>`

	styleCSS := `body {
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    line-height: 1.6;
    margin: 0;
    padding: 20px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    min-height: 100vh;
    color: #333;
}

.container {
    max-width: 800px;
    margin: 0 auto;
    background: white;
    padding: 2rem;
    border-radius: 10px;
    box-shadow: 0 4px 6px rgba(0,0,0,0.1);
}

h1, h2 {
    color: #2c3e50;
}

h1 {
    text-align: center;
    margin-bottom: 1.5rem;
}

.features ul, nav ul {
    list-style-type: none;
    padding: 0;
}

.features li, nav li {
    padding: 0.5rem 0;
    border-bottom: 1px solid #eee;
}

.features li:last-child, nav li:last-child {
    border-bottom: none;
}

a {
    color: #3498db;
    text-decoration: none;
}

a:hover {
    text-decoration: underline;
}`

	appJS := `console.log('SimpleHTTP Server is running!');

document.addEventListener('DOMContentLoaded', function() {
    const title = document.querySelector('h1');
    if (title) {
        title.addEventListener('click', function() {
            title.style.color = title.style.color === 'red' ? '#2c3e50' : 'red';
        });
    }
});`

	apiJSON := `{
    "server": "SimpleHTTP/1.0",
    "status": "running",
    "message": "API endpoint is working!",
    "timestamp": "` + time.Now().Format(time.RFC3339) + `",
    "endpoints": [
        "/",
        "/test.html",
        "/api.json"
    ]
}`

	files := map[string]string{
		"index.html": indexHTML,
		"test.html":  testHTML,
		"style.css":  styleCSS,
		"app.js":     appJS,
		"api.json":   apiJSON,
	}

	for filename, content := range files {
		filePath := filepath.Join(DocumentRoot, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			log.Printf("Error creating %s: %v", filename, err)
		} else {
			log.Printf("Created: %s", filePath)
		}
	}
}

func main() {
	port := DefaultPort
	root := DocumentRoot

	if len(os.Args) > 1 {
		for i, arg := range os.Args[1:] {
			switch arg {
			case "-p", "--port":
				if i+2 < len(os.Args) {
					port = os.Args[i+2]
				}
			case "-r", "--root":
				if i+2 < len(os.Args) {
					root = os.Args[i+2]
				}
			case "--setup":
				setupSampleWebsite()
				fmt.Println("Sample website created in", DocumentRoot)
				return
			case "-h", "--help":
				fmt.Println("SimpleHTTP Server")
				fmt.Println("Usage:")
				fmt.Println("  go run main.go [options]")
				fmt.Println("Options:")
				fmt.Println("  -p, --port PORT    Server port (default: 8080)")
				fmt.Println("  -r, --root PATH    Document root (default: ./www)")
				fmt.Println("  --setup            Create sample website")
				fmt.Println("  -h, --help         Show this help")
				return
			}
		}
	}

	server := NewServer(port, root)
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}