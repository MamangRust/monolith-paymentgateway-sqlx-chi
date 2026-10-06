package transferhandler

import (
	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/transfer"
	pbstats "github.com/MamangRust/monolith-payment-gateway-pb/transfer/stats"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/transfer"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"

	transfer_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/transfer"
)

type DepsTransfer struct {
	Client *grpc.ClientConn

	Router chi.Router

	Logger logger.LoggerInterface

	Cache *cache.CacheStore

	ApiHandler apierror.ApiHandler
}

func RegisterTransferHandler(deps *DepsTransfer) {
	mapper := apimapper.NewTransferResponseMapper()
	cache := transfer_cache.NewTransferMencache(deps.Cache)

	handlers := []func(){
		setupTransferQueryHandler(deps, mapper.QueryMapper(), cache),
		setupTransferCommandHandler(deps, mapper.CommandMapper(), cache),
		setupTransferStatsAmountHandler(deps, mapper.AmountStatsMapper(), cache),
		setupTransferStatsStatusHandler(deps, mapper.StatusStatsMapper(), cache),
	}

	for _, h := range handlers {
		h()
	}
}

func setupTransferQueryHandler(deps *DepsTransfer, mapper apimapper.TransferQueryResponseMapper, cache transfer_cache.TransferMencache) func() {
	return func() {
		NewTransferQueryHandleApi(&transferQueryHandleDeps{
			client:     pb.NewTransferQueryServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}

func setupTransferCommandHandler(deps *DepsTransfer, mapper apimapper.TransferCommandResponseMapper, cache transfer_cache.TransferMencache) func() {
	return func() {
		NewTransferCommandHandleApi(&transferCommandHandleDeps{
			client:     pb.NewTransferCommandServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}

func setupTransferStatsAmountHandler(deps *DepsTransfer, mapper apimapper.TransferStatsAmountResponseMapper, cache transfer_cache.TransferMencache) func() {
	return func() {
		NewTransferStatsAmountHandleApi(&transferStatsAmountHandleDeps{
			client:     pbstats.NewTransferStatsAmountServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}

func setupTransferStatsStatusHandler(deps *DepsTransfer, mapper apimapper.TransferStatsStatusResponseMapper, cache transfer_cache.TransferMencache) func() {
	return func() {
		NewTransferStatsStatusHandleApi(&transferStatsStatusHandleDeps{
			client:     pbstats.NewTransferStatsStatusServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}
