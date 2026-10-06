package rolehandler

import (
	"time"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/role"
	"github.com/MamangRust/monolith-payment-gateway-pkg/kafka"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/role"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/middlewares"
	mencache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis"
	role_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/role"
)

type DepsRole struct {
	Client *grpc.ClientConn

	Kafka *kafka.Kafka

	Router chi.Router

	Logger logger.LoggerInterface

	Cache mencache.RoleCache

	CacheStore *cache.CacheStore

	ApiHandler apierror.ApiHandler
}

func RegisterRoleHandler(deps *DepsRole) {
	mapper := apimapper.NewRoleResponseMapper()
	cache := role_cache.NewRoleMencache(deps.CacheStore)

	// Single shared RoleValidator: it owns the only consumer of the
	// "response-role" topic (1 partition) and the response channel registry.
	// Creating one per router would split the response consumer group, so
	// responses could be consumed by the other instance's handler and get
	// dropped ("No waiting channel") -> 408 timeouts.
	roleValidator := middlewares.NewRoleValidator(deps.Kafka, "request-role", "response-role", 5*time.Second, deps.Logger, deps.Cache)

	handlers := []func(){
		setupRoleQueryHandler(deps, deps.Cache, mapper.QueryMapper(), cache, roleValidator),
		setupRoleCommandHandler(deps, deps.Cache, mapper.CommandMapper(), cache, roleValidator),
	}

	for _, h := range handlers {
		h()
	}
}

func setupRoleQueryHandler(deps *DepsRole, cache_role mencache.RoleCache, mapper apimapper.RoleQueryResponseMapper, cache role_cache.RoleMencache, validator *middlewares.RoleValidator) func() {
	return func() {
		NewRoleQueryHandleApi(&roleQueryHandleDeps{
			client:     pb.NewRoleServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache_role: cache_role,
			kafka:      deps.Kafka,
			cache:      cache,
			apiHandler: deps.ApiHandler,
			validator:  validator,
		})
	}
}

func setupRoleCommandHandler(deps *DepsRole, cache_role mencache.RoleCache, mapper apimapper.RoleCommandResponseMapper, cache role_cache.RoleMencache, validator *middlewares.RoleValidator) func() {
	return func() {
		NewRoleCommandHandleApi(&roleCommandHandleDeps{
			client:     pb.NewRoleCommandServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			kafka:      deps.Kafka,
			cache_role: cache_role,
			cache:      cache,
			apiHandler: deps.ApiHandler,
			validator:  validator,
		})
	}
}
