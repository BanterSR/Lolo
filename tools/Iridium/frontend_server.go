package main

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
)

type eventBroker struct {
	mu          sync.RWMutex
	subscribers map[chan string]struct{}
}

var packetEvents = &eventBroker{
	subscribers: map[chan string]struct{}{},
}

//go:embed frontend/public
var staticFolder embed.FS

func startServer(autoStart bool) error {
	router := newRouter()
	if autoStart {
		if _, err := startLiveCapture(CaptureStartRequest{Label: "auto"}); err != nil {
			return fmt.Errorf("auto-start capture: %w", err)
		}
	}
	return router.Run(config.ListenAddr)
}

func newRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	if err := router.SetTrustedProxies(nil); err != nil {
		panic(fmt.Sprintf("configure trusted proxies: %v", err))
	}
	router.GET("/api/health", apiHealth)
	router.GET("/api/status", apiStatus)
	router.GET("/api/devices", apiDevices)
	router.GET("/api/packets", apiPackets)
	router.GET("/api/stream", stream)

	// Legacy GET routes remain for the bundled frontend.
	router.GET("/api/start", apiStart)
	router.GET("/api/stop", apiStop)
	router.POST("/api/capture/start", apiStart)
	router.POST("/api/capture/stop", apiStop)
	router.POST("/api/upload", apiUpload)

	router.Use(static.Serve("/", EmbedFolder(staticFolder, "frontend/public")))
	newProxy(router)
	return router
}

func apiHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ok":            true,
		"schemaVersion": captureSchemaVersion,
		"captureState":  captures.statusSnapshot().State,
	})
}

func apiStatus(c *gin.Context) {
	c.JSON(http.StatusOK, captures.statusSnapshot())
}

func apiDevices(c *gin.Context) {
	devices, err := listDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

func apiPackets(c *gin.Context) {
	afterID, err := parseUintQuery(c, "afterId", 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	limit, err := parseIntQuery(c, "limit", 200)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if limit < 1 || limit > 5000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 5000"})
		return
	}
	direction := c.Query("direction")
	if direction != "" && direction != "client_to_server" && direction != "server_to_client" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "direction must be client_to_server or server_to_client",
		})
		return
	}

	packets := captures.queryPackets(packetQuery{
		AfterID:   afterID,
		Limit:     limit,
		Name:      c.Query("name"),
		Direction: direction,
	})
	c.JSON(http.StatusOK, gin.H{
		"packets": packets,
		"status":  captures.statusSnapshot(),
	})
}

func apiStart(c *gin.Context) {
	request := CaptureStartRequest{}
	if c.Request.Method == http.MethodPost && c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	status, err := startLiveCapture(request)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, errCaptureActive) {
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, gin.H{"error": err.Error(), "status": status})
		return
	}
	c.JSON(http.StatusAccepted, status)
}

func apiStop(c *gin.Context) {
	status, err := captures.stop()
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "status": status})
		return
	}
	c.JSON(http.StatusAccepted, status)
}

func apiUpload(c *gin.Context) {
	upload, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	source, err := upload.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer source.Close()

	tempFile, err := os.CreateTemp("", "iridium-upload-*.pcap*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	tempPath := tempFile.Name()
	if _, err := io.Copy(tempFile, source); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := tempFile.Close(); err != nil {
		os.Remove(tempPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status, err := startOfflineCapture(tempPath, filepath.Base(upload.Filename), true)
	if err != nil {
		os.Remove(tempPath)
		statusCode := http.StatusInternalServerError
		if errors.Is(err, errCaptureActive) {
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, gin.H{"error": err.Error(), "status": status})
		return
	}
	c.JSON(http.StatusAccepted, status)
}

func stream(c *gin.Context) {
	subscriber := packetEvents.subscribe()
	defer packetEvents.unsubscribe(subscriber)

	c.Stream(func(w io.Writer) bool {
		select {
		case message := <-subscriber:
			c.SSEvent("packetNotify", message)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

func sendStreamMsg(message string) {
	packetEvents.publish(message)
}

func (b *eventBroker) subscribe() chan string {
	subscriber := make(chan string, 128)
	b.mu.Lock()
	b.subscribers[subscriber] = struct{}{}
	b.mu.Unlock()
	return subscriber
}

func (b *eventBroker) unsubscribe(subscriber chan string) {
	b.mu.Lock()
	delete(b.subscribers, subscriber)
	b.mu.Unlock()
}

func (b *eventBroker) publish(message string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for subscriber := range b.subscribers {
		select {
		case subscriber <- message:
		default:
			// A slow browser must not block capture or other subscribers.
		}
	}
}

func parseUintQuery(c *gin.Context, name string, fallback uint64) (uint64, error) {
	value := c.Query(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an unsigned integer", name)
	}
	return parsed, nil
}

func parseIntQuery(c *gin.Context, name string, fallback int) (int, error) {
	value := c.Query(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return parsed, nil
}

type embedFileSystem struct {
	http.FileSystem
}

func (e embedFileSystem) Exists(prefix string, path string) bool {
	_, err := e.Open(path)
	return err == nil
}

func EmbedFolder(fsEmbed embed.FS, targetPath string) static.ServeFileSystem {
	fsys, err := fs.Sub(fsEmbed, targetPath)
	if err != nil {
		panic(err)
	}
	return embedFileSystem{
		FileSystem: http.FS(fsys),
	}
}
