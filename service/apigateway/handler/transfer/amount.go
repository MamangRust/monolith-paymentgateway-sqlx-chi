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

type transferStatsAmountHandleApi struct {
	client pb.TransferStatsAmountServiceClient

	logger logger.LoggerInterface

	mapper apimapper.TransferStatsAmountResponseMapper

	cache transfer_cache.TransferMencache

	apiHandler apierror.ApiHandler
}

type transferStatsAmountHandleDeps struct {
	client pb.TransferStatsAmountServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.TransferStatsAmountResponseMapper

	cache transfer_cache.TransferMencache

	apiHandler apierror.ApiHandler
}

func NewTransferStatsAmountHandleApi(params *transferStatsAmountHandleDeps) *transferStatsAmountHandleApi {

	transferStatsAmountHandleApi := &transferStatsAmountHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/transfer-stats-amount", func(routerTransfer chi.Router) {

		routerTransfer.Get("/monthly-amount", params.apiHandler.Handle("find-monthly-transfer-amounts", transferStatsAmountHandleApi.FindMonthlyTransferAmounts))
		routerTransfer.Get("/yearly-amount", params.apiHandler.Handle("find-yearly-transfer-amounts", transferStatsAmountHandleApi.FindYearlyTransferAmounts))
		routerTransfer.Get("/monthly-by-sender", params.apiHandler.Handle("find-monthly-transfer-amounts-by-sender", transferStatsAmountHandleApi.FindMonthlyTransferAmountsBySenderCardNumber))
		routerTransfer.Get("/monthly-by-receiver", params.apiHandler.Handle("find-monthly-transfer-amounts-by-receiver", transferStatsAmountHandleApi.FindMonthlyTransferAmountsByReceiverCardNumber))
		routerTransfer.Get("/yearly-by-sender", params.apiHandler.Handle("find-yearly-transfer-amounts-by-sender", transferStatsAmountHandleApi.FindYearlyTransferAmountsBySenderCardNumber))
		routerTransfer.Get("/yearly-by-receiver", params.apiHandler.Handle("find-yearly-transfer-amounts-by-receiver", transferStatsAmountHandleApi.FindYearlyTransferAmountsByReceiverCardNumber))

	})
	return transferStatsAmountHandleApi
}

// FindMonthlyTransferAmounts retrieves the monthly transfer amounts for a specific year.
// @Summary Get monthly transfer amounts
// @Tags Transfer Amount
// @Security Bearer
// @Description Retrieve the monthly transfer amounts for a specific year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransferMonthAmount "Monthly transfer amounts"
// @Failure 400 {object} response.ErrorResponse "Invalid year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly transfer amounts"
// @Router /api/transfer-stats-amount/monthly-amounts [get]
func (h *transferStatsAmountHandleApi) FindMonthlyTransferAmounts(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedMonthTransferAmounts(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTransferAmounts(ctx, &pbtransfer.FindYearTransferStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly transfer amounts", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferMonthAmount(res)
	h.cache.SetCachedMonthTransferAmounts(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransferAmounts retrieves the yearly transfer amounts for a specific year.
// @Summary Get yearly transfer amounts
// @Tags Transfer Amount
// @Security Bearer
// @Description Retrieve the yearly transfer amounts for a specific year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransferYearAmount "Yearly transfer amounts"
// @Failure 400 {object} response.ErrorResponse "Invalid year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly transfer amounts"
// @Router /api/transfer-stats-amount/yearly-amounts [get]
func (h *transferStatsAmountHandleApi) FindYearlyTransferAmounts(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedYearlyTransferAmounts(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTransferAmounts(ctx, &pbtransfer.FindYearTransferStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly transfer amounts", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferYearAmount(res)
	h.cache.SetCachedYearlyTransferAmounts(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTransferAmountsBySenderCardNumber retrieves the monthly transfer amounts for a specific sender card number and year.
// @Summary Get monthly transfer amounts by sender card number
// @Tags Transfer Amount
// @Security Bearer
// @Description Retrieve the monthly transfer amounts for a specific sender card number and year.
// @Accept json
// @Produce json
// @Param card_number query string true "Sender Card Number"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransferMonthAmount "Monthly transfer amounts by sender card number"
// @Failure 400 {object} response.ErrorResponse "Invalid card number or year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly transfer amounts by sender card number"
// @Router /api/transfer-stats-amount/monthly-amounts-by-sender-card [get]
func (h *transferStatsAmountHandleApi) FindMonthlyTransferAmountsBySenderCardNumber(w http.ResponseWriter, r *http.Request) error {
	cardNumber := r.URL.Query().Get("card_number")
	yearStr := r.URL.Query().Get("year")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	if cardNumber == "" {
		return errors.NewBadRequestError("card_number is required")
	}

	ctx := r.Context()

	reqCache := &requests.MonthYearCardNumber{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetMonthlyTransferAmountsBySenderCard(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTransferAmountsBySenderCardNumber(ctx, &pbtransfer.FindByCardNumberTransferRequest{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly transfer amounts by sender card number", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferMonthAmount(res)
	h.cache.SetMonthlyTransferAmountsBySenderCard(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTransferAmountsByReceiverCardNumber retrieves the monthly transfer amounts for a specific receiver card number and year.
// @Summary Get monthly transfer amounts by receiver card number
// @Tags Transfer Amount
// @Security Bearer
// @Description Retrieve the monthly transfer amounts for a specific receiver card number and year.
// @Accept json
// @Produce json
// @Param card_number query string true "Receiver Card Number"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransferMonthAmount "Monthly transfer amounts by receiver card number"
// @Failure 400 {object} response.ErrorResponse "Invalid card number or year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly transfer amounts by receiver card number"
// @Router /api/transfer-stats-amount/monthly-amounts-by-receiver-card [get]
func (h *transferStatsAmountHandleApi) FindMonthlyTransferAmountsByReceiverCardNumber(w http.ResponseWriter, r *http.Request) error {
	cardNumber := r.URL.Query().Get("card_number")
	yearStr := r.URL.Query().Get("year")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	if cardNumber == "" {
		return errors.NewBadRequestError("card_number is required")
	}

	ctx := r.Context()

	reqCache := &requests.MonthYearCardNumber{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetMonthlyTransferAmountsByReceiverCard(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTransferAmountsByReceiverCardNumber(ctx, &pbtransfer.FindByCardNumberTransferRequest{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly transfer amounts by receiver card number", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferMonthAmount(res)
	h.cache.SetMonthlyTransferAmountsByReceiverCard(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransferAmountsBySenderCardNumber retrieves the yearly transfer amounts for a specific sender card number and year.
// @Summary Get yearly transfer amounts by sender card number
// @Tags Transfer Amount
// @Security Bearer
// @Description Retrieve the yearly transfer amounts for a specific sender card number and year.
// @Accept json
// @Produce json
// @Param card_number query string true "Sender Card Number"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransferYearAmount "Yearly transfer amounts by sender card number"
// @Failure 400 {object} response.ErrorResponse "Invalid card number or year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly transfer amounts by sender card number"
// @Router /api/transfer-stats-amount/yearly-amounts-by-sender-card [get]
func (h *transferStatsAmountHandleApi) FindYearlyTransferAmountsBySenderCardNumber(w http.ResponseWriter, r *http.Request) error {
	cardNumber := r.URL.Query().Get("card_number")
	yearStr := r.URL.Query().Get("year")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	if cardNumber == "" {
		return errors.NewBadRequestError("card_number is required")
	}

	ctx := r.Context()

	reqCache := &requests.MonthYearCardNumber{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetYearlyTransferAmountsBySenderCard(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTransferAmountsBySenderCardNumber(ctx, &pbtransfer.FindByCardNumberTransferRequest{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly transfer amounts by sender card number", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferYearAmount(res)
	h.cache.SetYearlyTransferAmountsBySenderCard(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransferAmountsByReceiverCardNumber retrieves the yearly transfer amounts for a specific receiver card number and year.
// @Summary Get yearly transfer amounts by receiver card number
// @Tags Transfer Amount
// @Security Bearer
// @Description Retrieve the yearly transfer amounts for a specific receiver card number and year.
// @Accept json
// @Produce json
// @Param card_number query string true "Receiver Card Number"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTransferYearAmount "Yearly transfer amounts by receiver card number"
// @Failure 400 {object} response.ErrorResponse "Invalid card number or year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly transfer amounts by receiver card number"
// @Router /api/transfer-stats-amount/yearly-amounts-by-receiver-card [get]
func (h *transferStatsAmountHandleApi) FindYearlyTransferAmountsByReceiverCardNumber(w http.ResponseWriter, r *http.Request) error {
	cardNumber := r.URL.Query().Get("card_number")
	yearStr := r.URL.Query().Get("year")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	if cardNumber == "" {
		return errors.NewBadRequestError("card_number is required")
	}

	ctx := r.Context()

	reqCache := &requests.MonthYearCardNumber{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetYearlyTransferAmountsByReceiverCard(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTransferAmountsByReceiverCardNumber(ctx, &pbtransfer.FindByCardNumberTransferRequest{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly transfer amounts by receiver card number", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransferYearAmount(res)
	h.cache.SetYearlyTransferAmountsByReceiverCard(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
