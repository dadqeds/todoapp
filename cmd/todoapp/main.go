package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	// База часовых поясов внутри бинарника: в alpine-образе её нет, а она
	// нужна для повторов и уведомлений по местному времени пользователей.
	_ "time/tzdata"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_pgx_pool "github.com/dadqeds/todoapp/internal/core/repository/postgres/pool/pgx"
	core_telegram "github.com/dadqeds/todoapp/internal/core/telegram"
	core_http_middleware "github.com/dadqeds/todoapp/internal/core/transport/http/middleware"
	core_http_server "github.com/dadqeds/todoapp/internal/core/transport/http/server"
	lists_postgres_repository "github.com/dadqeds/todoapp/internal/features/lists/repository/postgres"
	lists_service "github.com/dadqeds/todoapp/internal/features/lists/service"
	lists_transport_http "github.com/dadqeds/todoapp/internal/features/lists/transport/http"
	notifications_postgres_repository "github.com/dadqeds/todoapp/internal/features/notifications/repository/postgres"
	notifications_service "github.com/dadqeds/todoapp/internal/features/notifications/service"
	statistics_postgres_repository "github.com/dadqeds/todoapp/internal/features/statistics/repository/postgres"
	statistics_service "github.com/dadqeds/todoapp/internal/features/statistics/service"
	statistics_transport_http "github.com/dadqeds/todoapp/internal/features/statistics/transport/http"
	tasks_postgres_repository "github.com/dadqeds/todoapp/internal/features/tasks/repository/postgres"
	tasks_service "github.com/dadqeds/todoapp/internal/features/tasks/service"
	tasks_transport_http "github.com/dadqeds/todoapp/internal/features/tasks/transport/http"
	users_postgres_repository "github.com/dadqeds/todoapp/internal/features/users/repository/postgres"
	users_service "github.com/dadqeds/todoapp/internal/features/users/service"
	users_transport_http "github.com/dadqeds/todoapp/internal/features/users/transport/http"
	web_fs_repository "github.com/dadqeds/todoapp/internal/features/web/repository/file_system"
	web_service "github.com/dadqeds/todoapp/internal/features/web/service"
	web_transport_http "github.com/dadqeds/todoapp/internal/features/web/transport/http"
	"github.com/dadqeds/todoapp/public"
	"go.uber.org/zap"

	_ "github.com/dadqeds/todoapp/docs"
)

var (
	timeZone = time.UTC
)

// @title			Golang Todo API
// @version  		1.0
// @description		Todo Application REST-API scheme
// @BasePath		/api/v1
// @securityDefinitions.apikey	TelegramInitData
// @in							header
// @name						Authorization
// @description				"tma <initData>" из Telegram.WebApp.initData. На локальном адресе (AUTH_LOCAL_ADDR) не требуется
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "todoapp:", err)
		os.Exit(1)
	}
}

// run держит всю инициализацию отдельно от main, чтобы defer-ы
// (закрытие пула, логгера) гарантированно выполнялись до os.Exit.
func run() error {
	time.Local = timeZone

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	logger, err := core_logger.NewLogger(core_logger.NewConfigMust())
	if err != nil {
		return fmt.Errorf("init application logger: %w", err)
	}
	defer logger.Close()

	logger.Debug("application time zone", zap.Any("zone", timeZone))

	logger.Debug("initializing postgres connection pool")

	pool, err := core_pgx_pool.NewPool(
		ctx,
		core_pgx_pool.NewConfigMust(),
	)
	if err != nil {
		logger.Error("failed to init postgres connection pool", zap.Error(err))
		return fmt.Errorf("init postgres connection pool: %w", err)
	}
	defer pool.Close()

	authConfig := core_auth.NewConfigMust()

	logger.Debug("initializing feature", zap.String("feature", "users"))
	usersRepository := users_postgres_repository.NewUsersRepository(pool)
	usersService := users_service.NewUserService(usersRepository)
	usersTransportHTTP := users_transport_http.NewUsersHTTPHandler(usersService)

	logger.Debug("initializing feature", zap.String("feature", "lists"))
	listsRepository := lists_postgres_repository.NewListsRepository(pool)
	listsService := lists_service.NewListsService(listsRepository)
	listsTransportHTTP := lists_transport_http.NewListsHTTPHandler(listsService, authConfig.TelegramBotUsername)

	logger.Debug("initializing feature", zap.String("feature", "tasks"))
	tasksRepository := tasks_postgres_repository.NewTasksRepository(pool)
	tasksService := tasks_service.NewTasksService(tasksRepository, listsRepository)
	tasksTransportHTTP := tasks_transport_http.NewTasksHTTPHandler(tasksService)

	logger.Debug("initializing feature", zap.String("feature", "statistics"))
	statisticsRepository := statistics_postgres_repository.NewStatisticsRepository(pool)
	statisticsService := statistics_service.NewStatisticsService(statisticsRepository)
	statisticsTransportHTTP := statistics_transport_http.NewStatisticsHTTPHandler(statisticsService)

	logger.Debug("initializing feature", zap.String("feature", "web"))
	webRepository := web_fs_repository.NewWebRepository(public.FS)
	webService := web_service.NewWebService(webRepository)
	webTransportHTTP := web_transport_http.NewWebHTTPHandler(webService, public.FS)

	logger.Debug("initializing HTTP servers")

	httpConfig := core_http_server.NewConfigMust()

	if authConfig.TelegramBotToken == "" {
		logger.Warn("AUTH_TELEGRAM_BOT_TOKEN is not set: API on public address will reject all requests")
	}

	apiRoutes := [][]core_http_server.Route{
		usersTransportHTTP.Routes(),
		listsTransportHTTP.Routes(),
		tasksTransportHTTP.Routes(),
		statisticsTransportHTTP.Routes(),
	}

	newServer := func(config core_http_server.Config, auth core_http_middleware.Middleware) *core_http_server.HTTPServer {
		server := core_http_server.NewHTTPServer(
			config,
			logger,
			core_http_middleware.CORS(config.CORSAllowedOrigins),
			core_http_middleware.RequestID(),
			core_http_middleware.Logger(logger),
			core_http_middleware.Trace(),
			core_http_middleware.Panic(),
		)

		apiVersionRouterV1 := core_http_server.NewAPIVersionRouter(core_http_server.ApiVersion1, auth)
		for _, routes := range apiRoutes {
			apiVersionRouterV1.RegisterRouters(routes...)
		}

		server.RegisterAPIRouters(apiVersionRouterV1)
		server.RegisterRoutes(webTransportHTTP.Routes()...)
		server.RegisterSwagger()

		return server
	}

	// Публичный адрес: сюда смотрит туннель, доступ только с initData Telegram.
	servers := []*core_http_server.HTTPServer{
		newServer(httpConfig, core_http_middleware.TelegramAuth(authConfig, usersService)),
	}

	// Локальный адрес: без Telegram, от имени AUTH_LOCAL_TELEGRAM_ID с правами
	// администратора. Swagger здесь включён всегда.
	if authConfig.LocalAddr != "" {
		localConfig := httpConfig
		localConfig.Addr = authConfig.LocalAddr
		localConfig.SwaggerEnabled = true

		servers = append(servers, newServer(localConfig, core_http_middleware.LocalAuth(authConfig.LocalTelegramID, usersService)))
	}

	// Напоминания и утренняя сводка: без токена бота писать некому.
	if authConfig.TelegramBotToken != "" {
		notifier := notifications_service.NewNotifier(
			notifications_postgres_repository.NewNotificationsRepository(pool),
			core_telegram.NewClient(authConfig.TelegramAPIURL, authConfig.TelegramBotToken),
			authConfig.TelegramBotUsername,
			logger.With(zap.String("component", "notifier")),
		)
		go notifier.Run(ctx)
	}

	if err := runServers(ctx, servers); err != nil {
		logger.Error("HTTP server run error", zap.Error(err))
		return fmt.Errorf("run HTTP servers: %w", err)
	}

	return nil
}

// runServers запускает все серверы и останавливает их вместе: при ошибке
// одного или по сигналу завершения.
func runServers(ctx context.Context, servers []*core_http_server.HTTPServer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errs := make(chan error, len(servers))
	for _, server := range servers {
		go func() {
			err := server.Run(ctx)
			cancel()
			errs <- err
		}()
	}

	var errList []error
	for range servers {
		if err := <-errs; err != nil {
			errList = append(errList, err)
		}
	}

	return errors.Join(errList...)
}
