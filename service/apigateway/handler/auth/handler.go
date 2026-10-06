package authhandler

import (
	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	authapimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/auth"

	auth_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/auth"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

type DepsAuth struct {
	Client     *grpc.ClientConn
	Router     chi.Router
	Logger     logger.LoggerInterface
	Cache      *cache.CacheStore
	ApiHandler apierror.ApiHandler
}

func RegisterAuthHandler(deps *DepsAuth) {
	mapper := authapimapper.NewAuthResponseMapper()

	cache := auth_cache.NewMencache(deps.Cache)

	NewHandlerAuth(&authHandleParams{
		client:     pb.NewAuthServiceClient(deps.Client),
		router:     deps.Router,
		logger:     deps.Logger,
		mapper:     mapper,
		cache:      cache,
		apiHandler: deps.ApiHandler,
	})
}
