package withdrawhandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pbwithdraw "github.com/MamangRust/monolith-payment-gateway-pb/withdraw"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/withdraw/stats"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/withdraw"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	withdraw_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/withdraw"
)

type withdrawStatsStatusHandleApi struct {
	client pb.WithdrawStatsStatusServiceClient

	logger logger.LoggerInterface

	mapper apimapper.WithdrawStatsStatusResponseMapper

	cache withdraw_cache.WithdrawMencache

	apiHandler apierror.ApiHandler
}

type withdrawStatsStatusHandleDeps struct {
	client pb.WithdrawStatsStatusServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.WithdrawStatsStatusResponseMapper

	cache withdraw_cache.WithdrawMencache

	apiHandler apierror.ApiHandler
}

func NewWithdrawStatsStatusHandleApi(params *withdrawStatsStatusHandleDeps) *withdrawStatsStatusHandleApi {
	withdrawStatsStatusHandleApi := &withdrawStatsStatusHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/withdraw-stats-status", func(routerWithdraw chi.Router) {

		routerWithdraw.Get("/monthly-success", params.apiHandler.Handle("find-monthly-withdraw-status-success", withdrawStatsStatusHandleApi.FindMonthlyWithdrawStatusSuccess))
		routerWithdraw.Get("/yearly-success", params.apiHandler.Handle("find-yearly-withdraw-status-success", withdrawStatsStatusHandleApi.FindYearlyWithdrawStatusSuccess))
		routerWithdraw.Get("/monthly-failed", params.apiHandler.Handle("find-monthly-withdraw-status-failed", withdrawStatsStatusHandleApi.FindMonthlyWithdrawStatusFailed))
		routerWithdraw.Get("/yearly-failed", params.apiHandler.Handle("find-yearly-withdraw-status-failed", withdrawStatsStatusHandleApi.FindYearlyWithdrawStatusFailed))

		routerWithdraw.Get("/monthly-success-by-card", params.apiHandler.Handle("find-monthly-withdraw-status-success-by-card", withdrawStatsStatusHandleApi.FindMonthlyWithdrawStatusSuccessByCardNumber))
		routerWithdraw.Get("/yearly-success-by-card", params.apiHandler.Handle("find-yearly-withdraw-status-success-by-card", withdrawStatsStatusHandleApi.FindYearlyWithdrawStatusSuccessByCardNumber))
		routerWithdraw.Get("/monthly-failed-by-card", params.apiHandler.Handle("find-monthly-withdraw-status-failed-by-card", withdrawStatsStatusHandleApi.FindMonthlyWithdrawStatusFailedByCardNumber))
		routerWithdraw.Get("/yearly-failed-by-card", params.apiHandler.Handle("find-yearly-withdraw-status-failed-by-card", withdrawStatsStatusHandleApi.FindYearlyWithdrawStatusFailedByCardNumber))

	})
	return withdrawStatsStatusHandleApi
}

// FindMonthlyWithdrawStatusSuccess retrieves the monthly withdraw status for successful transactions.
// @Summary Get monthly withdraw status for successful transactions
// @Tags Withdraw Stats Withdraw
// @Security Bearer
// @Description Retrieve the monthly withdraw status for successful transactions by year and month.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param month query int true "Month"
// @Success 200 {object} response.ApiResponseWithdrawMonthStatusSuccess "Monthly withdraw status for successful transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year or month"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly withdraw status for successful transactions"
// @Router /api/withdraw-stats-status/monthly-success [get]
func (h *withdrawStatsStatusHandleApi) FindMonthlyWithdrawStatusSuccess(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.MonthStatusWithdraw{
		Year:  year,
		Month: month,
	}

	cachedData, found := h.cache.GetCachedMonthWithdrawStatusSuccessCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyWithdrawStatusSuccess(ctx, &pbwithdraw.FindMonthlyWithdrawStatus{
		Year:  int32(year),
		Month: int32(month),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly withdraw status success", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseWithdrawMonthStatusSuccess(res)
	h.cache.SetCachedMonthWithdrawStatusSuccessCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyWithdrawStatusSuccess retrieves the yearly withdraw status for successful transactions.
// @Summary Get yearly withdraw status for successful transactions
// @Tags Withdraw Stats Withdraw
// @Security Bearer
// @Description Retrieve the yearly withdraw status for successful transactions by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseWithdrawYearStatusSuccess "Yearly withdraw status for successful transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly withdraw status for successful transactions"
// @Router /api/withdraw-stats-status/yearly-success [get]
func (h *withdrawStatsStatusHandleApi) FindYearlyWithdrawStatusSuccess(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedYearlyWithdrawStatusSuccessCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyWithdrawStatusSuccess(ctx, &pbwithdraw.FindYearWithdrawStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly withdraw status success", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseWithdrawYearStatusSuccess(res)
	h.cache.SetCachedYearlyWithdrawStatusSuccessCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyWithdrawStatusFailed retrieves the monthly withdraw status for failed transactions.
// @Summary Get monthly withdraw status for failed transactions
// @Tags Withdraw Stats Withdraw
// @Security Bearer
// @Description Retrieve the monthly withdraw status for failed transactions by year and month.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param month query int true "Month"
// @Success 200 {object} response.ApiResponseWithdrawMonthStatusFailed "Monthly withdraw status for failed transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year or month"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly withdraw status for failed transactions"
// @Router /api/withdraw-stats-status/monthly-failed [get]
func (h *withdrawStatsStatusHandleApi) FindMonthlyWithdrawStatusFailed(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.MonthStatusWithdraw{
		Year:  year,
		Month: month,
	}

	cachedData, found := h.cache.GetCachedMonthWithdrawStatusFailedCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyWithdrawStatusFailed(ctx, &pbwithdraw.FindMonthlyWithdrawStatus{
		Year:  int32(year),
		Month: int32(month),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly withdraw status failed", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseWithdrawMonthStatusFailed(res)
	h.cache.SetCachedMonthWithdrawStatusFailedCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyWithdrawStatusFailed retrieves the yearly withdraw status for failed transactions.
// @Summary Get yearly withdraw status for failed transactions
// @Tags Withdraw Stats Withdraw
// @Security Bearer
// @Description Retrieve the yearly withdraw status for failed transactions by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseWithdrawYearStatusSuccess "Yearly withdraw status for failed transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly withdraw status for failed transactions"
// @Router /api/withdraw-stats-status/yearly-failed [get]
func (h *withdrawStatsStatusHandleApi) FindYearlyWithdrawStatusFailed(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedYearlyWithdrawStatusFailedCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyWithdrawStatusFailed(ctx, &pbwithdraw.FindYearWithdrawStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly withdraw status failed", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseWithdrawYearStatusFailed(res)
	h.cache.SetCachedYearlyWithdrawStatusFailedCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyWithdrawStatusSuccessByCardNumber retrieves the monthly withdraw status for successful transactions.
// @Summary Get monthly withdraw status for successful transactions
// @Tags Withdraw Stats Withdraw
// @Security Bearer
// @Description Retrieve the monthly withdraw status for successful transactions by year and month.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param month query int true "Month"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseWithdrawMonthStatusSuccess "Monthly withdraw status for successful transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year or month"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly withdraw status for successful transactions"
// @Router /api/withdraw-stats-status/monthly-success-by-card [get]
func (h *withdrawStatsStatusHandleApi) FindMonthlyWithdrawStatusSuccessByCardNumber(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	cardNumber := r.URL.Query().Get("card_number")

	if cardNumber == "" {
		return errors.NewBadRequestError("invalid card_number parameter")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month <= 0 || month > 12 {
		return errors.NewBadRequestError("invalid month parameter")
	}

	ctx := r.Context()

	reqCache := &requests.MonthStatusWithdrawCardNumber{
		CardNumber: cardNumber,
		Year:       year,
		Month:      month,
	}

	cachedData, found := h.cache.GetCachedMonthWithdrawStatusSuccessByCardNumber(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyWithdrawStatusSuccessCardNumber(ctx, &pbwithdraw.FindMonthlyWithdrawStatusCardNumber{
		Year:       int32(year),
		Month:      int32(month),
		CardNumber: cardNumber,
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly withdraw status success", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseWithdrawMonthStatusSuccess(res)
	h.cache.SetCachedMonthWithdrawStatusSuccessByCardNumber(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyWithdrawStatusSuccessByCardNumber retrieves the yearly withdraw status for successful transactions.
// @Summary Get yearly withdraw status for successful transactions
// @Tags Withdraw Stats Withdraw
// @Security Bearer
// @Description Retrieve the yearly withdraw status for successful transactions by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseWithdrawYearStatusSuccess "Yearly withdraw status for successful transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly withdraw status for successful transactions"
// @Router /api/withdraw-stats-status/yearly-success-by-card-number [get]
func (h *withdrawStatsStatusHandleApi) FindYearlyWithdrawStatusSuccessByCardNumber(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	cardNumber := r.URL.Query().Get("card_number")

	if cardNumber == "" {
		return errors.NewBadRequestError("invalid card_number parameter")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	reqCache := &requests.YearStatusWithdrawCardNumber{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetCachedYearlyWithdrawStatusSuccessByCardNumber(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyWithdrawStatusSuccessCardNumber(ctx, &pbwithdraw.FindYearWithdrawStatusCardNumber{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly withdraw status success", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseWithdrawYearStatusSuccess(res)
	h.cache.SetCachedYearlyWithdrawStatusSuccessByCardNumber(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyWithdrawStatusFailedByCardNumber retrieves the monthly withdraw status for failed transactions.
// @Summary Get monthly withdraw status for failed transactions
// @Tags Withdraw Stats Withdraw
// @Security Bearer
// @Description Retrieve the monthly withdraw status for failed transactions by year and month.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param month query int true "Month"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseWithdrawMonthStatusFailed "Monthly withdraw status for failed transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year or month"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly withdraw status for failed transactions"
// @Router /api/withdraw-stats-status/monthly-failed-by-card [get]
func (h *withdrawStatsStatusHandleApi) FindMonthlyWithdrawStatusFailedByCardNumber(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	cardNumber := r.URL.Query().Get("card_number")

	if cardNumber == "" {
		return errors.NewBadRequestError("invalid card_number parameter")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month <= 0 || month > 12 {
		return errors.NewBadRequestError("invalid month parameter")
	}

	ctx := r.Context()

	reqCache := &requests.MonthStatusWithdrawCardNumber{
		CardNumber: cardNumber,
		Year:       year,
		Month:      month,
	}

	cachedData, found := h.cache.GetCachedMonthWithdrawStatusFailedByCardNumber(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyWithdrawStatusFailedCardNumber(ctx, &pbwithdraw.FindMonthlyWithdrawStatusCardNumber{
		Year:       int32(year),
		Month:      int32(month),
		CardNumber: cardNumber,
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly withdraw status failed", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseWithdrawMonthStatusFailed(res)
	h.cache.SetCachedMonthWithdrawStatusFailedByCardNumber(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyWithdrawStatusFailedByCardNumber retrieves the yearly withdraw status for failed transactions.
// @Summary Get yearly withdraw status for failed transactions
// @Tags Withdraw Stats Withdraw
// @Security Bearer
// @Description Retrieve the yearly withdraw status for failed transactions by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseWithdrawYearStatusSuccess "Yearly withdraw status for failed transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly withdraw status for failed transactions"
// @Router /api/withdraw-stats-status/yearly-failed-by-card [get]
func (h *withdrawStatsStatusHandleApi) FindYearlyWithdrawStatusFailedByCardNumber(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	cardNumber := r.URL.Query().Get("card_number")

	if cardNumber == "" {
		return errors.NewBadRequestError("invalid card_number parameter")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	reqCache := &requests.YearStatusWithdrawCardNumber{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetCachedYearlyWithdrawStatusFailedByCardNumber(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyWithdrawStatusFailedCardNumber(ctx, &pbwithdraw.FindYearWithdrawStatusCardNumber{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly withdraw status failed", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseWithdrawYearStatusFailed(res)
	h.cache.SetCachedYearlyWithdrawStatusFailedByCardNumber(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
