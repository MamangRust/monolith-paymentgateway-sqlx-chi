package withdrawhandler

import (
	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/withdraw"
	pbstats "github.com/MamangRust/monolith-payment-gateway-pb/withdraw/stats"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/withdraw"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"

	withdraw_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/withdraw"
)

type DepsWithdraw struct {
	Client *grpc.ClientConn

	Router chi.Router

	Logger logger.LoggerInterface

	Cache *cache.CacheStore

	ApiHandler apierror.ApiHandler
}

func RegisterWithdrawHandler(deps *DepsWithdraw) {
	if deps.Client == nil {
		panic("RegisterWithdrawHandler: deps.Client is nil")
	}
	mapper := apimapper.NewWithdrawResponseMapper()

	cache := withdraw_cache.NewWithdrawMencache(deps.Cache)

	handlers := []func(){
		setupWithdrawQueryHandler(deps, mapper.QueryMapper(), cache),
		setupWithdrawCommandHandler(deps, mapper.CommandMapper(), cache),
		setupWithdrawStatsAmountHandler(deps, mapper.AmountStatsMapper(), cache),
		setupWithdrawStatsStatusHandler(deps, mapper.StatusStatsMapper(), cache),
	}

	for _, h := range handlers {
		h()
	}
}

func setupWithdrawQueryHandler(deps *DepsWithdraw, mapper apimapper.WithdrawQueryResponseMapper, cache withdraw_cache.WithdrawMencache) func() {
	return func() {
		NewWithdrawQueryHandleApi(&withdrawQueryHandleDeps{
			client:     pb.NewWithdrawQueryServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			apiHandler: deps.ApiHandler,
			cache:      cache,
		})
	}
}

func setupWithdrawCommandHandler(deps *DepsWithdraw, mapper apimapper.WithdrawCommandResponseMapper, cache withdraw_cache.WithdrawMencache) func() {
	return func() {
		NewWithdrawCommandHandleApi(&withdrawCommandHandleDeps{
			client:     pb.NewWithdrawCommandServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			apiHandler: deps.ApiHandler,
			cache:      cache,
		})
	}
}

func setupWithdrawStatsAmountHandler(deps *DepsWithdraw, mapper apimapper.WithdrawStatsAmountResponseMapper, cache withdraw_cache.WithdrawMencache) func() {
	return func() {
		NewWithdrawStatsAmountHandleApi(&withdrawStatsAmountHandleDeps{
			client:     pbstats.NewWithdrawStatsAmountServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			apiHandler: deps.ApiHandler,
			cache:      cache,
		})
	}
}

func setupWithdrawStatsStatusHandler(deps *DepsWithdraw, mapper apimapper.WithdrawStatsStatusResponseMapper, cache withdraw_cache.WithdrawMencache) func() {
	return func() {
		NewWithdrawStatsStatusHandleApi(&withdrawStatsStatusHandleDeps{
			client:     pbstats.NewWithdrawStatsStatusServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			apiHandler: deps.ApiHandler,
			cache:      cache,
		})
	}
}
