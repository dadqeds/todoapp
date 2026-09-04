package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_pgx_pool "github.com/dadqeds/todoapp/internal/core/repository/postgres/pool/pgx"
	core_http_middleware "github.com/dadqeds/todoapp/internal/core/transport/http/middleware"
	core_http_server "github.com/dadqeds/todoapp/internal/core/transport/http/server"
	statistics_postgres_repository "github.com/dadqeds/todoapp/internal/features/statistics/repository/postgres"
	statistics_service "github.com/dadqeds/todoapp/internal/features/statistics/service"
	statistics_transport_http "github.com/dadqeds/todoapp/internal/features/statistics/transport/http"
	tasks_postgres_repository "github.com/dadqeds/todoapp/internal/features/tasks/repository/postgres"
	tasks_service "github.com/dadqeds/todoapp/internal/features/tasks/service"
	tasks_transport_http "github.com/dadqeds/todoapp/internal/features/tasks/transport/http"
	users_postgres_repository "github.com/dadqeds/todoapp/internal/features/users/repository/postgres"
	users_service "github.com/dadqeds/todoapp/internal/features/users/service"
	users_transport_http "github.com/dadqeds/todoapp/internal/features/users/transport/http"
	"go.uber.org/zap"
)

var (
	timeZone = time.UTC
)

func main() {
	time.Local = timeZone

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	logger, err := core_logger.NewLogger(core_logger.NewConfigMust())
	if err != nil {
		fmt.Println("failed to init applecation logger:", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Debug("application time zone", zap.Any("zone", timeZone))

	logger.Debug("initiazling postgres connection pool")

	pool, err := core_pgx_pool.NewPool(
		ctx,
		core_pgx_pool.NewConfigMust(),
	)
	if err != nil {
		log.Fatal("failed to init postgres connection pool", zap.Error(err))
	}
	defer pool.Close()

	logger.Debug("initiazling feature", zap.String("feature", "users"))
	usersRepository := users_postgres_repository.NewUsersRepository(pool)
	usersService := users_service.NewUserService(usersRepository)
	usersTransportHTTP := users_transport_http.NewUsersHTTPHandler(usersService)

	logger.Debug("initiazling feature", zap.String("feature", "tasks"))
	tasksRepository := tasks_postgres_repository.NewTasksRepository(pool)
	tasksService := tasks_service.NewTasksService(tasksRepository)
	tasksTransportHTTP := tasks_transport_http.NewTasksHTTPHandler(tasksService)

	logger.Debug("initiazling feature", zap.String("feature", "statistics"))
	statisticsRepository := statistics_postgres_repository.NewStatisticsRepository(pool)
	statisticsService := statistics_service.NewStatisticsService(statisticsRepository)
	statisticsTransportHTTP := statistics_transport_http.NewStatisticsHTTPHandler(statisticsService)


	logger.Debug("initiazling HTTP server")

	httpServer := core_http_server.NewHTTPServer(
		core_http_server.NewConfigMust(),
		logger,
		core_http_middleware.RequestID(),
		core_http_middleware.Logger(logger),
		core_http_middleware.Trace(),
		core_http_middleware.Panic(),
	)
	apiVersionRouterV1 := core_http_server.NewAPIVersionRouter(core_http_server.ApiVersion1)
	apiVersionRouterV1.RegisterRouters(usersTransportHTTP.Routes()...)
	apiVersionRouterV1.RegisterRouters(tasksTransportHTTP.Routes()...)
	apiVersionRouterV1.RegisterRouters(statisticsTransportHTTP.Routes()...)

	httpServer.RegisterAPIRouters(
		apiVersionRouterV1,
	)

	if err := httpServer.Run(ctx); err != nil {
		logger.Error("HTTP server run error", zap.Error(err))
	}
}
