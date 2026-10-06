package user_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	user_handler "github.com/MamangRust/monolith-payment-gateway-apigateway/handler/user"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pbrole "github.com/MamangRust/monolith-payment-gateway-pb/role"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/user"
	"github.com/MamangRust/monolith-payment-gateway-pkg/hash"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	"github.com/MamangRust/monolith-payment-gateway-shared/observability"
	tests "github.com/MamangRust/monolith-payment-gateway-test"
	"github.com/MamangRust/monolith-payment-gateway-user/handler"
	"github.com/MamangRust/monolith-payment-gateway-user/repository"
	"github.com/MamangRust/monolith-payment-gateway-user/service"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	redis_client "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type UserApiTestSuite struct {
	suite.Suite
	ts         *tests.TestSuite
	dbPool     *sqlx.DB
	echo       chi.Router
	grpcServer *grpc.Server
	lis        *bufconn.Listener
	identity   *tests.IdentityStack
	userID     int32
}

func (s *UserApiTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	pool, err := sqlx.ConnectContext(s.ts.Ctx, "pgx", s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	opts, err := redis_client.ParseURL(s.ts.RedisURL)
	s.Require().NoError(err)
	redisClient := redis_client.NewClient(opts)

	queries := pool

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("test", lp)
	hasher := hash.NewHashingPassword()
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(redisClient, log, cacheMetrics)

	s.identity, err = tests.StartIdentityStack(pool, cacheStore, log)
	s.Require().NoError(err)

	repos := repository.NewRepositories(
		queries,
		pbrole.NewRoleServiceClient(s.identity.Role.Conn),
		pbrole.NewRoleCommandServiceClient(s.identity.Role.Conn),
	)

	svc := service.NewService(&service.Deps{
		Repositories:    repos,
		Logger:          log,
		Cache:           cacheStore,
		Hash:            hasher,
		RoleAdapter:     s.identity.Role.RoleAdapter,
		UserRoleAdapter: s.identity.Role.UserRoleAdapter,
	})

	userHandler := handler.NewHandler(svc)

	// Setup gRPC server
	s.lis = bufconn.Listen(1024 * 1024)
	s.grpcServer = grpc.NewServer()
	pb.RegisterUserQueryServiceServer(s.grpcServer, userHandler)
	pb.RegisterUserCommandServiceServer(s.grpcServer, userHandler)

	go func() {
		if err := s.grpcServer.Serve(s.lis); err != nil {
		}
	}()

	s.echo = chi.NewRouter()

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return s.lis.Dial()
		}),
		grpc.WithInsecure())
	s.Require().NoError(err)

	obs, err := observability.NewObservability("test", log)
	s.Require().NoError(err)
	apiHandler := apierror.NewApiHandler(obs, log)

	user_handler.RegisterUserHandler(&user_handler.DepsUser{
		Client:     conn,
		Router:     s.echo,
		Logger:     log,
		Cache:      cacheStore,
		ApiHandler: apiHandler,
	})
}

func (s *UserApiTestSuite) TearDownSuite() {
	if s.identity != nil {
		s.identity.Close()
	}
	if s.grpcServer != nil {
		s.grpcServer.Stop()
	}
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	if s.ts != nil {
		s.ts.Teardown()
	}
}

func (s *UserApiTestSuite) Test1_UserLifecycle() {
	// Create
	email := fmt.Sprintf("api.%d@example.com", time.Now().UnixNano())
	reqJSON := fmt.Sprintf(`{"firstname": "JohnApi", "lastname": "DoeApi", "email": "%s", "password": "Password123!", "confirm_password": "Password123!"}`, email)
	req := httptest.NewRequest(http.MethodPost, "/api/user-command/create", strings.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	var res map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	s.NoError(err)
	data := res["data"].(map[string]interface{})
	s.userID = int32(data["id"].(float64))
	s.Equal("JohnApi", data["firstname"])

	// FindById
	req = httptest.NewRequest(http.MethodGet, "/api/user-query/"+strconv.Itoa(int(s.userID)), nil)
	rec = httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	s.Equal(http.StatusOK, rec.Code)

	// Update
	updateJSON := fmt.Sprintf(`{"firstname": "JohnUpdated", "lastname": "DoeApi", "email": "%s", "password": "Password123!", "confirm_password": "Password123!"}`, email)
	req = httptest.NewRequest(http.MethodPost, "/api/user-command/update/"+strconv.Itoa(int(s.userID)), strings.NewReader(updateJSON))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *UserApiTestSuite) Test2_QueryOperations() {
	// FindAll
	req := httptest.NewRequest(http.MethodGet, "/api/user-query?page=1&page_size=10", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	s.Equal(http.StatusOK, rec.Code)

	// FindActive
	req = httptest.NewRequest(http.MethodGet, "/api/user-query/active?page=1&page_size=10", nil)
	rec = httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	s.Equal(http.StatusOK, rec.Code)
}

func TestUserApiSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(UserApiTestSuite))
}
