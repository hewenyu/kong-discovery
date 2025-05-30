package registration

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/hewenyu/kong-discovery/internal/core/config"
	"github.com/hewenyu/kong-discovery/internal/registration/handler"
	"github.com/hewenyu/kong-discovery/internal/registration/service"
	"github.com/hewenyu/kong-discovery/internal/store/etcd"
	serviceStore "github.com/hewenyu/kong-discovery/internal/store/service"
)

// Server 表示服务注册API服务
type Server struct {
	e           *echo.Echo
	host        string
	port        int
	handler     *handler.RegistrationHandler
	shutdownCtx context.Context
	cancel      context.CancelFunc
}

// NewServer 创建一个新的服务注册API服务
func NewServer(etcdClient *etcd.Client, cfg *config.Config) *Server {
	// 创建Echo实例
	e := echo.New()
	e.HideBanner = true

	// 添加中间件
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// 创建服务存储
	store := serviceStore.NewEtcdServiceStore(etcdClient, cfg.Namespace.Default)

	// 创建服务注册服务
	registrationService := service.NewRegistrationService(
		store,
		cfg.Service.Heartbeat.Interval,
		cfg.Service.Heartbeat.Timeout,
	)

	// 创建服务注册处理器
	registrationHandler := handler.NewRegistrationHandler(registrationService)

	// 注册路由
	registrationHandler.RegisterRoutes(e)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建服务
	server := &Server{
		e:           e,
		host:        cfg.Server.Registration.Host,
		port:        cfg.Server.Registration.Port,
		handler:     registrationHandler,
		shutdownCtx: ctx,
		cancel:      cancel,
	}

	// 启动定期清理任务
	registrationHandler.StartCleanupTask(cfg.Service.Heartbeat.Interval)

	return server
}

// Start 启动服务
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	log.Printf("服务注册API服务启动在 %s", addr)

	// 以非阻塞方式启动服务
	go func() {
		if err := s.e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务注册API服务启动失败: %v", err)
		}
	}()

	return nil
}

// Shutdown 关闭服务
func (s *Server) Shutdown(ctx context.Context) error {
	s.cancel()
	return s.e.Shutdown(ctx)
}
