package integration_tests

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
	appConfig "yourproject/xxx/config"
)

var testConfig struct {
	WaitForFormerTest bool   `mapstructure:"waitForFormerTest"`
	WaitTimeInSeconds int    `mapstructure:"waitTimeInSeconds"`
	ServerAddr        string `mapstructure:"serverAddr"`
	DebugMode         bool   `mapstructure:"debugMode"`
}

var (
	server     *http.Server
	serverLock sync.Mutex
)

func LoadTestConfig() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get current file path")
	}
	configDir := filepath.Dir(filename)

	viper.SetConfigName("test_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)
	viper.AddConfigPath(filepath.Join(configDir, ".."))       // parent directory
	viper.AddConfigPath(filepath.Join(configDir, "..", "..")) // project root

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("failed to read config file: %w", err))
	}

	if err := viper.Unmarshal(&testConfig); err != nil {
		panic(fmt.Errorf("failed to unmarshal config: %w", err))
	}
}

func init() {
	appConfig.Load()
	appConfig.InitDB()

	LoadTestConfig()

	if testConfig.DebugMode {
		checkServerIsRunning()
	}
}

func checkServerIsRunning() {
	client := &http.Client{Timeout: 5 * time.Second}
	pingURL := fmt.Sprintf("http://%s%s%s", testConfig.ServerAddr, controller.UrlV1, controller.UrlPing)
	resp, err := client.Get(pingURL)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to server: %v", err))
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(fmt.Sprintf("Failed to close response body: %v", err))
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("Server is not responding correctly. Status code: %d", resp.StatusCode))
	}
	fmt.Println("Server is running and responding to ping requests.")
}

func conditionallyWaitForFormerTest() {
	if testConfig.WaitForFormerTest {
		fmt.Printf("wait %d seconds for former test to finish\n", testConfig.WaitTimeInSeconds)
		time.Sleep(time.Duration(testConfig.WaitTimeInSeconds) * time.Second)
	}
}

func conditionallyStartServer(t *testing.T) {
	if testConfig.DebugMode {
		return
	}

	serverLock.Lock()
	defer serverLock.Unlock()

	// Ensure any existing server is shut down
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Errorf("Failed to shutdown previous server: %v", err)
		}
		server = nil
	}

	fmt.Printf("wait 2 seconds for port to be released\n")
	time.Sleep(2 * time.Second)

	gin.SetMode(gin.TestMode)
	appConfig.Load()
	appConfig.InitDB()
	userDAO := dao.NewUserDAO(appConfig.DB)
	userCache := cache.NewUserCache(appConfig.RedisClient)
	userRepo := repository.NewUserRepository(userDAO, userCache)
	userLogic := logic.NewUserLogic(userRepo)
	userHandler := controller.NewUserHandler(userLogic)

	router := gin.New()
	userHandler.RegisterRoutes(router)

	server = &http.Server{
		Addr:    testConfig.ServerAddr,
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Failed to run server: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(time.Second)
}

// Add this function to your test file
func shutdownServer(t *testing.T) {
	serverLock.Lock()
	defer serverLock.Unlock()

	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
		server = nil
	}
}
