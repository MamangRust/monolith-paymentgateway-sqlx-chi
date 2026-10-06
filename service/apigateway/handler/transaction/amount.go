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

type transactionStatsAmountHandleApi struct {
	client pb.TransactionStatsAmountServiceClient

	logger logger.LoggerInterface

	mapper apimapper.TransactionStatsAmountResponseMapper

	cache transaction_cache.TransactionMencache

	apiHandler apierror.ApiHandler
}

type transactionStatsAmountHandleDeps struct {
	client pb.TransactionStatsAmountServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.TransactionStatsAmountResponseMapper

	cache transaction_cache.TransactionMencache

	apiHandler apierror.ApiHandler
}

func NewTransactionStatsAmountHandleApi(params *transactionStatsAmountHandleDeps) *transactionStatsAmountHandleApi {

	transactionStatsAmountHandleApi := &transactionStatsAmountHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/transaction-stats-amount", func(routerTransaction chi.Router) {

		routerTransaction.Get("/monthly-amounts-by-card", params.apiHandler.Handle("find-monthly-amounts-by-card", transactionStatsAmountHandleApi.FindMonthlyAmountsByCardNumber))
		routerTransaction.Get("/yearly-amounts-by-card", params.apiHandler.Handle("find-yearly-amounts-by-card", transactionStatsAmountHandleApi.FindYearlyAmountsByCardNumber))
		routerTransaction.Get("/monthly-amounts", params.apiHandler.Handle("find-monthly-amounts", transactionStatsAmountHandleApi.FindMonthlyAmounts))
		routerTransaction.Get("/yearly-amounts", params.apiHandler.Handle("find-yearly-amounts", transactionStatsAmountHandleApi.FindYearlyAmounts))

	})
	return transactionStatsAmountHandleApi
}

// FindMonthlyAmounts retrieves the monthly transaction amounts for a specific year.
// @Summary Get monthly transaction amounts
// @Tags Transaction Stats Amount
// @Security Bearer
// @Description Retrieve the monthly transaction amounts for a specific year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransactionMonthAmount "Monthly transaction amounts"
// @Failure 400 {object} response.ErrorResponse "Invalid year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly transaction amounts"
// @Router /api/transaction-stats-amount/monthly-amounts [get]
func (h *transactionStatsAmountHandleApi) FindMonthlyAmounts(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetMonthlyAmountsCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyAmounts(ctx, &pbtransaction.FindYearTransactionStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly amounts", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionMonthAmount(res)
	h.cache.SetMonthlyAmountsCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyAmounts retrieves the yearly transaction amounts for a specific year.
// @Summary Get yearly transaction amounts
// @Tags Transaction Stats Amount
// @Security Bearer
// @Description Retrieve the yearly transaction amounts for a specific year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransactionYearAmount "Yearly transaction amounts"
// @Failure 400 {object} response.ErrorResponse "Invalid year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly transaction amounts"
// @Router /api/transaction-stats-amount/yearly-amounts [get]
func (h *transactionStatsAmountHandleApi) FindYearlyAmounts(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearlyAmountsCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyAmounts(ctx, &pbtransaction.FindYearTransactionStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly amounts", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionYearAmount(res)
	h.cache.SetYearlyAmountsCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyAmountsByCardNumber retrieves the monthly transaction amounts for a specific card number and year.
// @Summary Get monthly transaction amounts by card number
// @Tags Transaction Stats Amount
// @Security Bearer
// @Description Retrieve the monthly transaction amounts for a specific card number and year.
// @Accept json
// @Produce json
// @Param card_number query string true "Card Number"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransactionMonthAmount "Monthly transaction amounts by card number"
// @Failure 400 {object} response.ErrorResponse "Invalid card number or year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly transaction amounts by card number"
// @Router /api/transaction-stats-amount/monthly-amounts-by-card [get]
func (h *transactionStatsAmountHandleApi) FindMonthlyAmountsByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.MonthYearPaymentMethod{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetMonthlyAmountsByCardCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyAmountsByCardNumber(ctx, &pbtransaction.FindByYearCardNumberTransactionRequest{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly amounts by card number", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionMonthAmount(res)
	h.cache.SetMonthlyAmountsByCardCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyAmountsByCardNumber retrieves the yearly transaction amounts for a specific card number and year.
// @Summary Get yearly transaction amounts by card number
// @Tags Transaction Stats Amount
// @Security Bearer
// @Description Retrieve the yearly transaction amounts for a specific card number and year.
// @Accept json
// @Produce json
// @Param card_number query string true "Card Number"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransactionYearAmount "Yearly transaction amounts by card number"
// @Failure 400 {object} response.ErrorResponse "Invalid card number or year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly transaction amounts by card number"
// @Router /api/transaction-stats-amount/yearly-amounts-by-card [get]
func (h *transactionStatsAmountHandleApi) FindYearlyAmountsByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.MonthYearPaymentMethod{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetYearlyAmountsByCardCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyAmountsByCardNumber(ctx, &pbtransaction.FindByYearCardNumberTransactionRequest{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly amounts by card number", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionYearAmount(res)
	h.cache.SetYearlyAmountsByCardCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
