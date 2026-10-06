package transactionhandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pbtransaction "github.com/MamangRust/monolith-payment-gateway-pb/transaction"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/transaction/stats"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/transaction"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	transaction_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/transaction"
)

type transactionStatsStatusHandleApi struct {
	client pb.TransactionStatsStatusServiceClient

	logger logger.LoggerInterface

	mapper apimapper.TransactionStatsStatusResponseMapper

	cache transaction_cache.TransactionMencache

	apiHandler apierror.ApiHandler
}

type transactionStatsStatusHandleDeps struct {
	client pb.TransactionStatsStatusServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.TransactionStatsStatusResponseMapper

	cache transaction_cache.TransactionMencache

	apiHandler apierror.ApiHandler
}

func NewTransactionStatsStatusHandleApi(params *transactionStatsStatusHandleDeps) *transactionStatsStatusHandleApi {

	transactionStatsStatusHandleApi := &transactionStatsStatusHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/transaction-stats-status", func(routerTransaction chi.Router) {

		routerTransaction.Get("/monthly-success", httpx.Handler(transactionStatsStatusHandleApi.FindMonthlyTransactionStatusSuccess))
		routerTransaction.Get("/yearly-success", httpx.Handler(transactionStatsStatusHandleApi.FindYearlyTransactionStatusSuccess))
		routerTransaction.Get("/monthly-failed", httpx.Handler(transactionStatsStatusHandleApi.FindMonthlyTransactionStatusFailed))
		routerTransaction.Get("/yearly-failed", httpx.Handler(transactionStatsStatusHandleApi.FindYearlyTransactionStatusFailed))

		routerTransaction.Get("/monthly-success-by-card", httpx.Handler(transactionStatsStatusHandleApi.FindMonthlyTransactionStatusSuccessByCardNumber))
		routerTransaction.Get("/yearly-success-by-card", httpx.Handler(transactionStatsStatusHandleApi.FindYearlyTransactionStatusSuccessByCardNumber))
		routerTransaction.Get("/monthly-failed-by-card", httpx.Handler(transactionStatsStatusHandleApi.FindMonthlyTransactionStatusFailedByCardNumber))
		routerTransaction.Get("/yearly-failed-by-card", httpx.Handler(transactionStatsStatusHandleApi.FindYearlyTransactionStatusFailedByCardNumber))

	})
	return transactionStatsStatusHandleApi
}

// FindMonthlyTransactionStatusSuccess retrieves the monthly transaction status for successful transactions.
// @Summary Get monthly transaction status for successful transactions
// @Tags Transaction Stats Status
// @Security Bearer
// @Description Retrieve the monthly transaction status for successful transactions by year and month.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param month query int true "Month"
// @Success 200 {object} response.ApiResponseTransactionMonthStatusSuccess "Monthly transaction status for successful transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year or month"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly transaction status for successful transactions"
// @Router "/api/transaction-stats-status/monthly-success [get]
func (h *transactionStatsStatusHandleApi) FindMonthlyTransactionStatusSuccess(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.MonthStatusTransaction{
		Year:  year,
		Month: month,
	}

	cachedData, found := h.cache.GetMonthTransactionStatusSuccessCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTransactionStatusSuccess(ctx, &pbtransaction.FindMonthlyTransactionStatus{
		Year:  int32(year),
		Month: int32(month),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly transaction status success", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionMonthStatusSuccess(res)
	h.cache.SetMonthTransactionStatusSuccessCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransactionStatusSuccess retrieves the yearly transaction status for successful transactions.
// @Summary Get yearly transaction status for successful transactions
// @Tags Transaction Stats Status
// @Security Bearer
// @Description Retrieve the yearly transaction status for successful transactions by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransactionYearStatusSuccess "Yearly transaction status for successful transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly transaction status for successful transactions"
// @Router "/api/transaction-stats-status/yearly-success [get]
func (h *transactionStatsStatusHandleApi) FindYearlyTransactionStatusSuccess(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearTransactionStatusSuccessCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTransactionStatusSuccess(ctx, &pbtransaction.FindYearTransactionStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly transaction status success", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionYearStatusSuccess(res)
	h.cache.SetYearTransactionStatusSuccessCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTransactionStatusFailed retrieves the monthly transaction status for failed transactions.
// @Summary Get monthly transaction status for failed transactions
// @Tags Transaction Stats Status
// @Security Bearer
// @Description Retrieve the monthly transaction status for failed transactions by year and month.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param month query int true "Month"
// @Success 200 {object} response.ApiResponseTransactionMonthStatusFailed "Monthly transaction status for failed transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year or month"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly transaction status for failed transactions"
// @Router "/api/transaction-stats-status/monthly-failed [get]
func (h *transactionStatsStatusHandleApi) FindMonthlyTransactionStatusFailed(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.MonthStatusTransaction{
		Year:  year,
		Month: month,
	}

	cachedData, found := h.cache.GetMonthTransactionStatusFailedCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTransactionStatusFailed(ctx, &pbtransaction.FindMonthlyTransactionStatus{
		Year:  int32(year),
		Month: int32(month),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly transaction status failed", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionMonthStatusFailed(res)
	h.cache.SetMonthTransactionStatusFailedCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransactionStatusFailed retrieves the yearly transaction status for failed transactions.
// @Summary Get yearly transaction status for failed transactions
// @Tags Transaction Stats Status
// @Security Bearer
// @Description Retrieve the yearly transaction status for failed transactions by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransactionYearStatusFailed "Yearly transaction status for failed transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly transaction status for failed transactions"
// @Router "/api/transaction-stats-status/yearly-failed [get]
func (h *transactionStatsStatusHandleApi) FindYearlyTransactionStatusFailed(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearTransactionStatusFailedCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTransactionStatusFailed(ctx, &pbtransaction.FindYearTransactionStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly transaction status failed", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionYearStatusFailed(res)
	h.cache.SetYearTransactionStatusFailedCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTransactionStatusSuccess retrieves the monthly transaction status for successful transactions.
// @Summary Get monthly transaction status for successful transactions
// @Tags Transaction Stats Status
// @Security Bearer
// @Description Retrieve the monthly transaction status for successful transactions by year and month.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param month query int true "Month"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseTransactionMonthStatusSuccess "Monthly transaction status for successful transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year or month"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly transaction status for successful transactions"
// @Router "/api/transaction-stats-status/monthly-success-by-card [get]
func (h *transactionStatsStatusHandleApi) FindMonthlyTransactionStatusSuccessByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.MonthStatusTransactionCardNumber{
		CardNumber: cardNumber,
		Year:       year,
		Month:      month,
	}

	cachedData, found := h.cache.GetMonthTransactionStatusSuccessByCardCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTransactionStatusSuccessByCardNumber(ctx, &pbtransaction.FindMonthlyTransactionStatusCardNumber{
		CardNumber: cardNumber,
		Year:       int32(year),
		Month:      int32(month),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly transaction status success", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionMonthStatusSuccess(res)
	h.cache.SetMonthTransactionStatusSuccessByCardCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransactionStatusSuccess retrieves the yearly transaction status for successful transactions.
// @Summary Get yearly transaction status for successful transactions
// @Tags Transaction Stats Status
// @Security Bearer
// @Description Retrieve the yearly transaction status for successful transactions by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param cardNumber query string true "Card Number"
// @Success 200 {object} response.ApiResponseTransactionYearStatusSuccess "Yearly transaction status for successful transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly transaction status for successful transactions"
// @Router "/api/transaction-stats-status/yearly-success-by-card [get]
func (h *transactionStatsStatusHandleApi) FindYearlyTransactionStatusSuccessByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.YearStatusTransactionCardNumber{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetYearTransactionStatusSuccessByCardCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTransactionStatusSuccessByCardNumber(ctx, &pbtransaction.FindYearTransactionStatusCardNumber{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly transaction status success", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionYearStatusSuccess(res)
	h.cache.SetYearTransactionStatusSuccessByCardCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTransactionStatusFailed retrieves the monthly transaction status for failed transactions.
// @Summary Get monthly transaction status for failed transactions
// @Tags Transaction Stats Status
// @Security Bearer
// @Description Retrieve the monthly transaction status for failed transactions by year and month.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param month query int true "Month"
// @Param cardNumber query string true "Card Number"
// @Success 200 {object} response.ApiResponseTransactionMonthStatusFailed "Monthly transaction status for failed transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year or month"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly transaction status for failed transactions"
// @Router "/api/transaction-stats-status/monthly-failed-by-card [get]
func (h *transactionStatsStatusHandleApi) FindMonthlyTransactionStatusFailedByCardNumber(w http.ResponseWriter, r *http.Request) error {
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
		return errors.NewBadRequestError("invalid card_number paramater")
	}

	ctx := r.Context()

	reqCache := &requests.MonthStatusTransactionCardNumber{
		CardNumber: cardNumber,
		Year:       year,
		Month:      month,
	}

	cachedData, found := h.cache.GetMonthTransactionStatusFailedByCardCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTransactionStatusFailedByCardNumber(ctx, &pbtransaction.FindMonthlyTransactionStatusCardNumber{
		CardNumber: cardNumber,
		Year:       int32(year),
		Month:      int32(month),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly transaction status failed", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionMonthStatusFailed(res)
	h.cache.SetMonthTransactionStatusFailedByCardCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransactionStatusFailedByCardNumber retrieves the yearly transaction status for failed transactions.
// @Summary Get yearly transaction status for failed transactions
// @Tags Transaction Stats Status
// @Security Bearer
// @Description Retrieve the yearly transaction status for failed transactions by year.
// @Accept json
// @Produce json
// @Param card_number query string true "Card Number"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransactionYearStatusFailed "Yearly transaction status for failed transactions"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly transaction status for failed transactions"
// @Router "/api/transaction-stats-status/yearly-failed-by-card [get]
func (h *transactionStatsStatusHandleApi) FindYearlyTransactionStatusFailedByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.YearStatusTransactionCardNumber{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetYearTransactionStatusFailedByCardCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTransactionStatusFailedByCardNumber(ctx, &pbtransaction.FindYearTransactionStatusCardNumber{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly transaction status failed", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionYearStatusFailed(res)
	h.cache.SetYearTransactionStatusFailedByCardCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
