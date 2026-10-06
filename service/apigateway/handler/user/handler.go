package userhandler

import (
	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/user"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/user"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"

	user_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/user"
)

type DepsUser struct {
	Client *grpc.ClientConn

	Router chi.Router

	Logger logger.LoggerInterface

	Cache *cache.CacheStore

	ApiHandler apierror.ApiHandler
}

func RegisterUserHandler(deps *DepsUser) {
	mapper := apimapper.NewUserResponseMapper()

	cache := user_cache.NewUserMencache(deps.Cache)

	handlers := []func(){
		setupUserQueryHandler(deps, mapper.QueryMapper(), cache),
		setupUserCommandHandler(deps, mapper.CommandMapper(), cache),
	}

	for _, h := range handlers {
		h()
	}
}

func setupUserQueryHandler(deps *DepsUser, mapper apimapper.UserQueryResponseMapper, cache user_cache.UserMencache) func() {
	return func() {
		NewUserQueryHandleApi(&userQueryHandleDeps{
			client:     pb.NewUserQueryServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}

func setupUserCommandHandler(deps *DepsUser, mapper apimapper.UserCommandResponseMapper, cache user_cache.UserMencache) func() {
	return func() {
		NewUserCommandHandleApi(&userCommandHandleDeps{
			client:     pb.NewUserCommandServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}
