package saldohandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pbsaldo "github.com/MamangRust/monolith-payment-gateway-pb/saldo"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/saldo/stats"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/saldo"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	saldo_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/saldo"
)

type saldoTotalBalanceHandleApi struct {
	saldo pb.SaldoStatsTotalBalanceClient

	logger logger.LoggerInterface

	mapper apimapper.SaldoStatsTotalResponseMapper

	cache saldo_cache.SaldoMencache

	apiHandler apierror.ApiHandler
}

type saldoTotalBalanceHandleDeps struct {
	client pb.SaldoStatsTotalBalanceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.SaldoStatsTotalResponseMapper

	cache saldo_cache.SaldoMencache

	apiHandler apierror.ApiHandler
}

func NewSaldoTotalBalanceHandleApi(params *saldoTotalBalanceHandleDeps) *saldoTotalBalanceHandleApi {
	saldoHandler := &saldoTotalBalanceHandleApi{
		saldo:      params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/saldo-stats-total-balance", func(routerSaldo chi.Router) {

		routerSaldo.Get("/monthly-total-balance", params.apiHandler.Handle("find-monthly-total-saldo-balance", saldoHandler.FindMonthlyTotalSaldoBalance))
		routerSaldo.Get("/yearly-total-balance", params.apiHandler.Handle("find-yearly-total-saldo-balance", saldoHandler.FindYearTotalSaldoBalance))

	})
	return saldoHandler
}

// FindMonthlyTotalSaldoBalance retrieves the total saldo balance for a specific month and year.
// @Summary Get monthly total saldo balance
// @Tags Saldo
// @Security Bearer
// @Description Retrieve the total saldo balance for a specific month and year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param month query int true "Month"
// @Success 200 {object} response.ApiResponseMonthTotalSaldo "Monthly total saldo balance"
// @Failure 400 {object} response.ErrorResponse "Invalid year or month parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly total saldo balance"
// @Router /api/saldo-stats-total-balances/monthly-total-balance [get]
func (h *saldoTotalBalanceHandleApi) FindMonthlyTotalSaldoBalance(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month <= 0 || month > 12 {
		return errors.NewBadRequestError("invalid month parameter")
	}

	ctx := r.Context()

	reqCache := &requests.MonthTotalSaldoBalance{
		Year:  year,
		Month: month,
	}

	cachedData, found := h.cache.GetMonthlyTotalSaldoBalanceCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.saldo.FindMonthlyTotalSaldoBalance(ctx, &pbsaldo.FindMonthlySaldoTotalBalance{
		Year:  int32(year),
		Month: int32(month),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly total saldo balance", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthTotalSaldo(res)
	h.cache.SetMonthlyTotalSaldoCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearTotalSaldoBalance retrieves the total saldo balance for a specific year.
// @Summary Get yearly total saldo balance
// @Tags Saldo
// @Security Bearer
// @Description Retrieve the total saldo balance for a specific year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseYearTotalSaldo "Yearly total saldo balance"
// @Failure 400 {object} response.ErrorResponse "Invalid year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly total saldo balance"
// @Router /api/saldo-stats-total-balance/yearly-total-balance [get]
func (h *saldoTotalBalanceHandleApi) FindYearTotalSaldoBalance(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearTotalSaldoBalanceCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.saldo.FindYearTotalSaldoBalance(ctx, &pbsaldo.FindYearlySaldo{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve year total saldo balance", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearTotalSaldo(res)
	h.cache.SetYearTotalSaldoBalanceCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
