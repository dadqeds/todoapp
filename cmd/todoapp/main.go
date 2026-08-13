package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_pgx_pool "github.com/dadqeds/todoapp/internal/core/repository/postgres/pool/pgx"
	core_http_middleware "github.com/dadqeds/todoapp/internal/core/transport/http/middleware"
	core_http_server "github.com/dadqeds/todoapp/internal/core/transport/http/server"
	users_postgres_repository "github.com/dadqeds/todoapp/internal/features/users/repository/postgres"
	users_service "github.com/dadqeds/todoapp/internal/features/users/service"
	users_transport_http "github.com/dadqeds/todoapp/internal/features/users/transport/http"
	"go.uber.org/zap"
)

func main(){
	ctx,cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	logger, err := core_logger.NewLogger(core_logger.NewConfigMust())
	if err !=nil{
		fmt.Println("failed to init applecation logger:",err)
		os.Exit(1)
	}
	defer logger.Close()


	logger.Debug("initiazling postgres connection pool")

	pool,err := core_pgx_pool.NewPool(
		ctx,
		core_pgx_pool.NewConfigMust(),
	)
	if err != nil{
		log.Fatal("failed to init postgres connection pool",zap.Error(err))
	}
	defer pool.Close()
	
	logger.Debug("initiazling feature",zap.String("feature","users"))
	usersRepository := users_postgres_repository.NewUsersRepository(pool)
	usersService := users_service.NewUserService(usersRepository)
	usersTransportHTTP := users_transport_http.NewUsersHTTPHandler(usersService)
	
	logger.Debug("initiazling HTTP server")
	
	httpServer := core_http_server.NewHTTPServer(
		core_http_server.NewConfigMust(),
		logger,
		core_http_middleware.RequestID(),
		core_http_middleware.Logger(logger),
		core_http_middleware.Trace(),
		core_http_middleware.Panic(),
	)
	apiVersionRouter :=core_http_server.NewAPIVersionRouter(core_http_server.ApiVersion1)
	apiVersionRouter.RegisterRouters(usersTransportHTTP.Routes()...)
	httpServer.RegisterAPIRouters(apiVersionRouter)

	if err := httpServer.Run(ctx); err != nil{
		logger.Error("HTTP server run error", zap.Error(err))
	}
}