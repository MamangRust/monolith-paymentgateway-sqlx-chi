package transferhandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pbtransfer "github.com/MamangRust/monolith-payment-gateway-pb/transfer"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/transfer/stats"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/transfer"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	transfer_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/transfer"
)

type transferStatsStatusHandleApi struct {
	client pb.TransferStatsStatusServiceClient

	logger logger.LoggerInterface

	mapper apimapper.TransferStatsStatusResponseMapper

	cache transfer_cache.TransferMencache

	apiHandler apierror.ApiHandler
}

type transferStatsStatusHandleDeps struct {
	client pb.TransferStatsStatusServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.TransferStatsStatusResponseMapper

	cache transfer_cache.TransferMencache

	apiHandler apierror.ApiHandler
}

func NewTransferStatsStatusHandleApi(params *transferStatsStatusHandleDeps) *transferStatsStatusHandleApi {

	transferStatsStatusHandleApi := &transferStatsStatusHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/transfer-stats-status", func(routerTransfer chi.Router) {

		routerTransfer.Get("/monthly-success", params.apiHandler.Handle("find-monthly-transfer-status-success", transferStatsStatusHandleApi.FindMonthlyTransferStatusSuccess))
		routerTransfer.Get("/yearly-success", params.apiHandler.Handle("find-yearly-transfer-status-success", transferStatsStatusHandleApi.FindYearlyTransferStatusSuccess))
		routerTransfer.Get("/monthly-failed", params.apiHandler.Handle("find-monthly-transfer-status-failed", transferStatsStatusHandleApi.FindMonthlyTransferStatusFailed))
		routerTransfer.Get("/yearly-failed", params.apiHandler.Handle("find-yearly-transfer-status-failed", transferStatsStatusHandleApi.FindYearlyTransferStatusFailed))

		routerTransfer.Get("/monthly-success-by-card", params.apiHandler.Handle("find-monthly-transfer-status-success-by-card", transferStatsStatusHandleApi.FindMonthlyTransferStatusSuccessByCardNumber))
		routerTransfer.Get("/yearly-success-by-card", params.apiHandler.Handle("find-yearly-transfer-status-success-by-card", transferStatsStatusHandleApi.FindYearlyTransferStatusSuccessByCardNumber))
		routerTransfer.Get("/monthly-failed-by-card", params.apiHandler.Handle("find-monthly-transfer-status-failed-by-card", transferStatsStatusHandleApi.FindMonthlyTransferStatusFailedByCardNumber))
		routerTransfer.Get("/yearly-failed-by-card", params.apiHandler.Handle("find-yearly-transfer-status-failed-by-card", transferStatsStatusHandleApi.FindYearlyTransferStatusFailedByCardNumber))

	})
	return transferStatsStatusHandleApi
}

// FindMonthlyTransferStatusSuccess retrieves the monthly transfer status for successful transactions.
// @Summary Get monthly transfer status for successful transactions
// @Tags Transfer Status
// @Security Bearer
// @Description Retrieve the monthly transfer status for successful transactions by year and month.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param month query int true "Month"
// @Success 200 {object} response.ApiResponseTransferMonthStatusSuccess "Monthly transfer status for successful transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year or month"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly transfer status for successful transactions"
// @Router /api/transfer-stats-status/monthly-success [get]
func (h *transferStatsStatusHandleApi) FindMonthlyTransferStatusSuccess(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.MonthStatusTransfer{
		Year:  year,
		Month: month,
	}

	cachedData, found := h.cache.GetCachedMonthTransferStatusSuccess(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTransferStatusSuccess(ctx, &pbtransfer.FindMonthlyTransferStatus{
		Year:  int32(year),
		Month: int32(month),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly transfer status success", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferMonthStatusSuccess(res)
	h.cache.SetCachedMonthTransferStatusSuccess(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransferStatusSuccess retrieves the yearly transfer status for successful transactions.
// @Summary Get yearly transfer status for successful transactions
// @Tags Transfer Status
// @Security Bearer
// @Description Retrieve the yearly transfer status for successful transactions by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransferYearStatusSuccess "Yearly transfer status for successful transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly transfer status for successful transactions"
// @Router /api/transfer-stats-status/yearly-success [get]
func (h *transferStatsStatusHandleApi) FindYearlyTransferStatusSuccess(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedYearlyTransferStatusSuccess(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTransferStatusSuccess(ctx, &pbtransfer.FindYearTransferStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly transfer status success", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferYearStatusSuccess(res)
	h.cache.SetCachedYearlyTransferStatusSuccess(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTransferStatusFailed retrieves the monthly transfer status for failed transactions.
// @Summary Get monthly transfer status for failed transactions
// @Tags Transfer Status
// @Security Bearer
// @Description Retrieve the monthly transfer status for failed transactions by year and month.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param month query int true "Month"
// @Success 200 {object} response.ApiResponseTransferMonthStatusFailed "Monthly transfer status for failed transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year or month"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly transfer status for failed transactions"
// @Router /api/transfer-stats-status/monthly-failed [get]
func (h *transferStatsStatusHandleApi) FindMonthlyTransferStatusFailed(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.MonthStatusTransfer{
		Year:  year,
		Month: month,
	}

	cachedData, found := h.cache.GetCachedMonthTransferStatusFailed(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTransferStatusFailed(ctx, &pbtransfer.FindMonthlyTransferStatus{
		Year:  int32(year),
		Month: int32(month),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly transfer status failed", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferMonthStatusFailed(res)
	h.cache.SetCachedMonthTransferStatusFailed(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransferStatusFailed retrieves the yearly transfer status for failed transactions.
// @Summary Get yearly transfer status for failed transactions
// @Tags Transfer Status
// @Security Bearer
// @Description Retrieve the yearly transfer status for failed transactions by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransferYearStatusFailed "Yearly transfer status for failed transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly transfer status for failed transactions"
// @Router /api/transfer-stats-status/yearly-failed [get]
func (h *transferStatsStatusHandleApi) FindYearlyTransferStatusFailed(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedYearlyTransferStatusFailed(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTransferStatusFailed(ctx, &pbtransfer.FindYearTransferStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly transfer status failed", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferYearStatusFailed(res)
	h.cache.SetCachedYearlyTransferStatusFailed(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTransferStatusSuccessByCardNumber retrieves the monthly transfer status for successful transactions.
// @Summary Get monthly transfer status for successful transactions
// @Tags Transfer Status
// @Security Bearer
// @Description Retrieve the monthly transfer status for successful transactions by year and month.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param month query int true "Month"
// @Success 200 {object} response.ApiResponseTransferMonthStatusSuccess "Monthly transfer status for successful transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year or month"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly transfer status for successful transactions"
// @Router /api/transfer-stats-status/monthly-success-by-card [get]
func (h *transferStatsStatusHandleApi) FindMonthlyTransferStatusSuccessByCardNumber(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	cardNumber := r.URL.Query().Get("card_number")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month <= 0 || month > 12 {
		return errors.NewBadRequestError("invalid month parameter")
	}

	if cardNumber == "" {
		return errors.NewBadRequestError("card_number is required")
	}

	ctx := r.Context()

	reqCache := &requests.MonthStatusTransferCardNumber{
		CardNumber: cardNumber,
		Year:       year,
		Month:      month,
	}

	cachedData, found := h.cache.GetMonthTransferStatusSuccessByCard(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTransferStatusSuccessByCardNumber(ctx, &pbtransfer.FindMonthlyTransferStatusCardNumber{
		Year:       int32(year),
		Month:      int32(month),
		CardNumber: cardNumber,
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly transfer status success", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferMonthStatusSuccess(res)
	h.cache.SetMonthTransferStatusSuccessByCard(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransferStatusSuccessByCardNumber retrieves the yearly transfer status for successful transactions.
// @Summary Get yearly transfer status for successful transactions
// @Tags Transfer Status
// @Security Bearer
// @Description Retrieve the yearly transfer status for successful transactions by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseTransferYearStatusSuccess "Yearly transfer status for successful transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly transfer status for successful transactions"
// @Router /api/transfer-stats-status/yearly-success-by-card [get]
func (h *transferStatsStatusHandleApi) FindYearlyTransferStatusSuccessByCardNumber(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	cardNumber := r.URL.Query().Get("card_number")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	if cardNumber == "" {
		return errors.NewBadRequestError("card_number is required")
	}

	ctx := r.Context()

	reqCache := &requests.YearStatusTransferCardNumber{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetYearlyTransferStatusSuccessByCard(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTransferStatusSuccessByCardNumber(ctx, &pbtransfer.FindYearTransferStatusCardNumber{
		Year:       int32(year),
		CardNumber: cardNumber,
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly transfer status success", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferYearStatusSuccess(res)
	h.cache.SetYearlyTransferStatusSuccessByCard(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTransferStatusFailedByCardNumber retrieves the monthly transfer status for failed transactions.
// @Summary Get monthly transfer status for failed transactions
// @Tags Transfer Status
// @Security Bearer
// @Description Retrieve the monthly transfer status for failed transactions by year and month.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param month query int true "Month"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseTransferMonthStatusFailed "Monthly transfer status for failed transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year or month"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly transfer status for failed transactions"
// @Router /api/transfer-stats-status/monthly-failed-by-card [get]
func (h *transferStatsStatusHandleApi) FindMonthlyTransferStatusFailedByCardNumber(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	cardNumber := r.URL.Query().Get("card_number")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month <= 0 || month > 12 {
		return errors.NewBadRequestError("invalid month parameter")
	}

	if cardNumber == "" {
		return errors.NewBadRequestError("card_number is required")
	}

	ctx := r.Context()

	reqCache := &requests.MonthStatusTransferCardNumber{
		CardNumber: cardNumber,
		Year:       year,
		Month:      month,
	}

	cachedData, found := h.cache.GetMonthTransferStatusFailedByCard(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTransferStatusFailedByCardNumber(ctx, &pbtransfer.FindMonthlyTransferStatusCardNumber{
		Year:       int32(year),
		Month:      int32(month),
		CardNumber: cardNumber,
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly transfer status failed", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferMonthStatusFailed(res)
	h.cache.SetMonthTransferStatusFailedByCard(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransferStatusFailedByCardNumber retrieves the yearly transfer status for failed transactions.
// @Summary Get yearly transfer status for failed transactions
// @Tags Transfer Status
// @Security Bearer
// @Description Retrieve the yearly transfer status for failed transactions by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseTransferYearStatusFailed "Yearly transfer status for failed transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly transfer status for failed transactions"
// @Router /api/transfer-stats-status/yearly-failed-by-card [get]
func (h *transferStatsStatusHandleApi) FindYearlyTransferStatusFailedByCardNumber(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	cardNumber := r.URL.Query().Get("card_number")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	if cardNumber == "" {
		return errors.NewBadRequestError("card_number is required")
	}

	ctx := r.Context()

	reqCache := &requests.YearStatusTransferCardNumber{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetYearlyTransferStatusFailedByCard(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTransferStatusFailedByCardNumber(ctx, &pbtransfer.FindYearTransferStatusCardNumber{
		Year:       int32(year),
		CardNumber: cardNumber,
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly transfer status failed", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferYearStatusFailed(res)
	h.cache.SetYearlyTransferStatusFailedByCard(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
