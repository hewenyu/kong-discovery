package admin

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/hewenyu/kong-discovery/internal/admin/handler"
	"github.com/hewenyu/kong-discovery/internal/admin/service"
	"github.com/hewenyu/kong-discovery/internal/core/config"
	"github.com/hewenyu/kong-discovery/internal/store/etcd"
	serviceStore "github.com/hewenyu/kong-discovery/internal/store/service"
)

// Server 表示管理API服务
type Server struct {
	e              *echo.Echo
	host           string
	port           int
	serviceHandler *handler.ServiceHandler
	shutdownCtx    context.Context
	cancel         context.CancelFunc
}

// NewServer 创建一个新的管理API服务
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

	// 创建管理服务
	adminService := service.NewAdminService(store)

	// 创建服务处理器
	serviceHandler := handler.NewServiceHandler(adminService)

	// 注册路由
	serviceHandler.RegisterRoutes(e)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建服务
	server := &Server{
		e:              e,
		host:           cfg.Server.Admin.Host,
		port:           cfg.Server.Admin.Port,
		serviceHandler: serviceHandler,
		shutdownCtx:    ctx,
		cancel:         cancel,
	}

	return server
}

// Start 启动服务
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	log.Printf("管理API服务启动在 %s", addr)

	// 以非阻塞方式启动服务
	go func() {
		if err := s.e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("管理API服务启动失败: %v", err)
		}
	}()

	return nil
}

// Shutdown 关闭服务
func (s *Server) Shutdown(ctx context.Context) error {
	s.cancel()
	return s.e.Shutdown(ctx)
}
