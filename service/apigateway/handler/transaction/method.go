package transactionhandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pbstats "github.com/MamangRust/monolith-payment-gateway-pb/transaction"
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

type transactionStatsMethodHandleApi struct {
	client pb.TransactionStatsMethodServiceClient

	logger logger.LoggerInterface

	mapper apimapper.TransactionStatsMethodResponseMapper

	cache transaction_cache.TransactionMencache

	apiHandler apierror.ApiHandler
}

type transactionStatsMethodHandleDeps struct {
	client pb.TransactionStatsMethodServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.TransactionStatsMethodResponseMapper

	cache transaction_cache.TransactionMencache

	apiHandler apierror.ApiHandler
}

func NewTransactionStatsMethodHandleApi(params *transactionStatsMethodHandleDeps) *transactionStatsMethodHandleApi {

	transactionStatsMethodHandleApi := &transactionStatsMethodHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/transaction-stats-method", func(routerTransaction chi.Router) {

		routerTransaction.Get("/monthly-methods", httpx.Handler(transactionStatsMethodHandleApi.FindMonthlyPaymentMethods))
		routerTransaction.Get("/yearly-methods", httpx.Handler(transactionStatsMethodHandleApi.FindYearlyPaymentMethods))
		routerTransaction.Get("/monthly-methods-by-card", httpx.Handler(transactionStatsMethodHandleApi.FindMonthlyPaymentMethodsByCardNumber))
		routerTransaction.Get("/yearly-methods-by-card", httpx.Handler(transactionStatsMethodHandleApi.FindYearlyPaymentMethodsByCardNumber))

	})
	return transactionStatsMethodHandleApi
}

// FindMonthlyPaymentMethods retrieves the monthly payment methods for transactions.
// @Summary Get monthly payment methods
// @Tags Transaction Stats Method
// @Security Bearer
// @Description Retrieve the monthly payment methods for transactions by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransactionMonthMethod "Monthly payment methods"
// @Failure 400 {object} response.ErrorResponse "Invalid year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly payment methods"
// @Router /api/transaction-stats-method/monthly-payment-methods [get]
func (h *transactionStatsMethodHandleApi) FindMonthlyPaymentMethods(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetMonthlyPaymentMethodsCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyPaymentMethods(ctx, &pbstats.FindYearTransactionStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly payment methods", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionMonthMethod(res)
	h.cache.SetMonthlyPaymentMethodsCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyPaymentMethods retrieves the yearly payment methods for transactions.
// @Summary Get yearly payment methods
// @Tags Transaction Stats Method
// @Security Bearer
// @Description Retrieve the yearly payment methods for transactions by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransactionYearMethod "Yearly payment methods"
// @Failure 400 {object} response.ErrorResponse "Invalid year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly payment methods"
// @Router /api/transaction-stats-method/yearly-payment-methods [get]
func (h *transactionStatsMethodHandleApi) FindYearlyPaymentMethods(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearlyPaymentMethodsCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyPaymentMethods(ctx, &pbstats.FindYearTransactionStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly payment methods", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionYearMethod(res)
	h.cache.SetYearlyPaymentMethodsCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyPaymentMethodsByCardNumber retrieves the monthly payment methods for transactions by card number and year.
// @Summary Get monthly payment methods by card number
// @Tags Transaction Stats Method
// @Security Bearer
// @Description Retrieve the monthly payment methods for transactions by card number and year.
// @Accept json
// @Produce json
// @Param card_number query string true "Card Number"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransactionMonthMethod "Monthly payment methods by card number"
// @Failure 400 {object} response.ErrorResponse "Invalid card number or year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly payment methods by card number"
// @Router /api/transaction-stats-method/monthly-payment-methods-by-card [get]
func (h *transactionStatsMethodHandleApi) FindMonthlyPaymentMethodsByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	cachedData, found := h.cache.GetMonthlyPaymentMethodsByCardCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyPaymentMethodsByCardNumber(ctx, &pbstats.FindByYearCardNumberTransactionRequest{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly payment methods by card number", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionMonthMethod(res)
	h.cache.SetMonthlyPaymentMethodsByCardCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyPaymentMethodsByCardNumber retrieves the yearly payment methods for transactions by card number and year.
// @Summary Get yearly payment methods by card number
// @Tags Transaction Stats Method
// @Security Bearer
// @Description Retrieve the yearly payment methods for transactions by card number and year.
// @Accept json
// @Produce json
// @Param card_number query string true "Card Number"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransactionYearMethod "Yearly payment methods by card number"
// @Failure 400 {object} response.ErrorResponse "Invalid card number or year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly payment methods by card number"
// @Router /api/transaction-stats-method/yearly-payment-methods-by-card [get]
func (h *transactionStatsMethodHandleApi) FindYearlyPaymentMethodsByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	cachedData, found := h.cache.GetYearlyPaymentMethodsByCardCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyPaymentMethodsByCardNumber(ctx, &pbstats.FindByYearCardNumberTransactionRequest{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly payment methods by card number", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransactionYearMethod(res)
	h.cache.SetYearlyPaymentMethodsByCardCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
