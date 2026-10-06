package cardhandler

import (
	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/card"
	pbstats "github.com/MamangRust/monolith-payment-gateway-pb/card/stats"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/card"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"

	card_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/card"
)

type DepsCard struct {
	Client *grpc.ClientConn

	Router chi.Router

	Logger logger.LoggerInterface

	Cache *cache.CacheStore

	ApiHandler apierror.ApiHandler
}

func RegisterCardHandler(deps *DepsCard) {
	mapper := apimapper.NewCardResponseMapper()
	cache := card_cache.NewCardMencache(deps.Cache)

	handlers := []func(){
		setupCardQueryHandler(deps, mapper.QueryMapper(), cache),
		setupCardCommandHandler(deps, mapper.CommandMapper(), cache),
		setupCardDashboardHandler(deps, mapper.DashboardMapper(), cache),
		setupCardStatsBalanceHandler(deps, mapper.BalanceStatsMapper(), cache),
		setupCardStatsTransactionHandler(deps, mapper.AmountStatsMapper(), cache),
		setupCardStatsTopupHandler(deps, mapper.AmountStatsMapper(), cache),
		setupCardStatsWithdrawHandler(deps, mapper.AmountStatsMapper(), cache),
		setupCardStatsTransferHandler(deps, mapper.AmountStatsMapper(), cache),
	}

	for _, h := range handlers {
		h()
	}
}

func setupCardQueryHandler(deps *DepsCard, mapper apimapper.CardQueryResponseMapper, cache card_cache.CardMencache) func() {
	return func() {
		NewCardQueryHandleApi(&cardQueryHandleApiDeps{
			client:     pb.NewCardQueryServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}

func setupCardCommandHandler(deps *DepsCard, mapper apimapper.CardCommandResponseMapper, cache card_cache.CardMencache) func() {
	return func() {
		NewCardCommandHandleApi(&cardCommandHandleApiDeps{
			client:     pb.NewCardCommandServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}

func setupCardDashboardHandler(deps *DepsCard, mapper apimapper.CardDashboardResponseMapper, cache card_cache.CardMencache) func() {
	return func() {
		NewCardDashboardHandleApi(&cardDashboardHandleApiDeps{
			client:     pb.NewCardDashboardServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}

func setupCardStatsBalanceHandler(deps *DepsCard, mapper apimapper.CardStatsBalanceResponseMapper, cache card_cache.CardMencache) func() {
	return func() {
		NewCardStatsBalanceHandleApi(&cardStatsBalanceHandleApiDeps{
			client:     pbstats.NewCardStatsBalanceServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}

func setupCardStatsTopupHandler(deps *DepsCard, mapper apimapper.CardStatsAmountResponseMapper, cache card_cache.CardMencache) func() {
	return func() {
		NewCardStatsTopupHandleApi(&cardStatsTopupHandleApiDeps{
			client:     pbstats.NewCardStatsTopupServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}

func setupCardStatsTransactionHandler(deps *DepsCard, mapper apimapper.CardStatsAmountResponseMapper, cache card_cache.CardMencache) func() {
	return func() {
		NewCardStatsTransactionHandleApi(&cardStatsTransactionHandleApiDeps{
			client:     pbstats.NewCardStatsTransactionServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}

func setupCardStatsTransferHandler(deps *DepsCard, mapper apimapper.CardStatsAmountResponseMapper, cache card_cache.CardMencache) func() {
	return func() {
		NewCardStatsTransferHandleApi(&cardStatsTransferHandleApiDeps{
			client:     pbstats.NewCardStatsTransferServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}

func setupCardStatsWithdrawHandler(deps *DepsCard, mapper apimapper.CardStatsAmountResponseMapper, cache card_cache.CardMencache) func() {
	return func() {
		NewCardStatsWithdrawHandleApi(&cardStatsWithdrawHandleApiDeps{
			client:     pbstats.NewCardStatsWithdrawServiceClient(deps.Client),
			router:     deps.Router,
			logger:     deps.Logger,
			mapper:     mapper,
			cache:      cache,
			apiHandler: deps.ApiHandler,
		})
	}
}
