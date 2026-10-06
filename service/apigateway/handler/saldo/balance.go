package saldohandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pbsaldo "github.com/MamangRust/monolith-payment-gateway-pb/saldo"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/saldo/stats"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/saldo"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	saldo_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/saldo"
)

type saldoStatsBalanceHandleApi struct {
	saldo pb.SaldoStatsBalanceServiceClient

	logger logger.LoggerInterface

	mapper apimapper.SaldoStatsBalanceResponseMapper

	cache saldo_cache.SaldoMencache

	apiHandler apierror.ApiHandler
}

type saldoStatsBalanceHandleDeps struct {
	client pb.SaldoStatsBalanceServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.SaldoStatsBalanceResponseMapper

	cache saldo_cache.SaldoMencache

	apiHandler apierror.ApiHandler
}

func NewSaldoStatsBalanceHandleApi(params *saldoStatsBalanceHandleDeps) *saldoStatsBalanceHandleApi {

	saldoHandler := &saldoStatsBalanceHandleApi{
		saldo:      params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/saldo-stats-balance", func(routerSaldo chi.Router) {

		routerSaldo.Get("/monthly-balances", params.apiHandler.Handle("find-monthly-saldo-balances", saldoHandler.FindMonthlySaldoBalances))
		routerSaldo.Get("/yearly-balances", params.apiHandler.Handle("find-yearly-saldo-balances", saldoHandler.FindYearlySaldoBalances))

	})
	return saldoHandler
}

// FindMonthlySaldoBalances retrieves monthly saldo balances for a specific year.
// @Summary Get monthly saldo balances
// @Tags Saldo Stats Balance
// @Security Bearer
// @Description Retrieve monthly saldo balances for a specific year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseMonthSaldoBalances "Monthly saldo balances"
// @Failure 400 {object} response.ErrorResponse "Invalid year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly saldo balances"
// @Router /api/saldo-stats-balances/monthly-balances [get]
func (h *saldoStatsBalanceHandleApi) FindMonthlySaldoBalances(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetMonthlySaldoBalanceCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.saldo.FindMonthlySaldoBalances(ctx, &pbsaldo.FindYearlySaldo{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly saldo balances", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthSaldoBalances(res)
	h.cache.SetMonthlySaldoBalanceCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlySaldoBalances retrieves yearly saldo balances for a specific year.
// @Summary Get yearly saldo balances
// @Tags Saldo Stats Balance
// @Security Bearer
// @Description Retrieve yearly saldo balances for a specific year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseYearSaldoBalances "Yearly saldo balances"
// @Failure 400 {object} response.ErrorResponse "Invalid year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly saldo balances"
// @Router /api/saldo-stats-balance/yearly-balances [get]
func (h *saldoStatsBalanceHandleApi) FindYearlySaldoBalances(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearlySaldoBalanceCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.saldo.FindYearlySaldoBalances(ctx, &pbsaldo.FindYearlySaldo{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly saldo balances", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearSaldoBalances(res)
	h.cache.SetYearlySaldoBalanceCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
