package saldohandler

import (
	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/saldo"
	pbstats "github.com/MamangRust/monolith-payment-gateway-pb/saldo/stats"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/saldo"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"

	saldo_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/saldo"
)

type DepsSaldo struct {
	Client *grpc.ClientConn

	Router chi.Router

	Logger logger.LoggerInterface

	Cache *cache.CacheStore

	ApiHandler apierror.ApiHandler
}

func RegisterSaldoHandler(deps *DepsSaldo) {
	mapper := apimapper.NewSaldoResponseMapper()

	cache := saldo_cache.NewSaldoMencache(deps.Cache)

	handlers := []func(){
		setupSaldoQueryHandler(deps, mapper.QueryMapper(), cache),
		setupSaldoCommandHandler(deps, mapper.CommandMapper(), cache),
		setupSaldoStatsBalanceHandler(deps, mapper.BalanceStatsMapper(), cache),
		setupStatsSaldoTotalBalanceHandler(deps, mapper.TotalStatsMapper(), cache),
	}

	for _, h := range handlers {
		h()
	}
}

func setupSaldoQueryHandler(deps *DepsSaldo, mapper apimapper.SaldoQueryResponseMapper, cache saldo_cache.SaldoMencache) func() {
	return func() {
		NewSaldoQueryHandleApi(
			&saldoQueryHandleDeps{
				client:     pb.NewSaldoQueryServiceClient(deps.Client),
				router:     deps.Router,
				logger:     deps.Logger,
				mapper:     mapper,
				cache:      cache,
				apiHandler: deps.ApiHandler,
			},
		)
	}
}

func setupSaldoCommandHandler(deps *DepsSaldo, mapper apimapper.SaldoCommandResponseMapper, cache saldo_cache.SaldoMencache) func() {
	return func() {
		NewSaldoCommandHandleApi(
			&saldoCommandHandleDeps{
				client:     pb.NewSaldoCommandServiceClient(deps.Client),
				router:     deps.Router,
				logger:     deps.Logger,
				mapper:     mapper,
				apiHandler: deps.ApiHandler,
				cache:      cache,
			},
		)
	}
}

func setupSaldoStatsBalanceHandler(deps *DepsSaldo, mapper apimapper.SaldoStatsBalanceResponseMapper, cache saldo_cache.SaldoMencache) func() {
	return func() {
		NewSaldoStatsBalanceHandleApi(
			&saldoStatsBalanceHandleDeps{
				client:     pbstats.NewSaldoStatsBalanceServiceClient(deps.Client),
				router:     deps.Router,
				logger:     deps.Logger,
				mapper:     mapper,
				apiHandler: deps.ApiHandler,
				cache:      cache,
			},
		)
	}
}

func setupStatsSaldoTotalBalanceHandler(deps *DepsSaldo, mapper apimapper.SaldoStatsTotalResponseMapper, cache saldo_cache.SaldoMencache) func() {
	return func() {
		NewSaldoTotalBalanceHandleApi(
			&saldoTotalBalanceHandleDeps{
				client:     pbstats.NewSaldoStatsTotalBalanceClient(deps.Client),
				router:     deps.Router,
				logger:     deps.Logger,
				mapper:     mapper,
				cache:      cache,
				apiHandler: deps.ApiHandler,
			},
		)
	}
}
