package cardhandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/card/stats"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/card"
	"github.com/go-chi/chi/v5"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	card_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/card"
)

type cardStatsBalanceHandleApi struct {
	card pb.CardStatsBalanceServiceClient

	logger logger.LoggerInterface

	apiHandler apierror.ApiHandler

	cache card_cache.CardMencache

	mapper apimapper.CardStatsBalanceResponseMapper
}

type cardStatsBalanceHandleApiDeps struct {
	client pb.CardStatsBalanceServiceClient

	router chi.Router

	logger logger.LoggerInterface

	cache card_cache.CardMencache

	apiHandler apierror.ApiHandler

	mapper apimapper.CardStatsBalanceResponseMapper
}

func NewCardStatsBalanceHandleApi(
	params *cardStatsBalanceHandleApiDeps,
) *cardStatsBalanceHandleApi {

	cardHandler := &cardStatsBalanceHandleApi{
		card:       params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/card-stats-balance", func(routerCard chi.Router) {

		routerCard.Get("/monthly-balance", params.apiHandler.Handle("find-monthly-balance", cardHandler.FindMonthlyBalance))
		routerCard.Get("/yearly-balance", params.apiHandler.Handle("find-yearly-balance", cardHandler.FindYearlyBalance))
		routerCard.Get("/monthly-balance-by-card", params.apiHandler.Handle("find-monthly-balance-by-card", cardHandler.FindMonthlyBalanceByCardNumber))
		routerCard.Get("/yearly-balance-by-card", params.apiHandler.Handle("find-yearly-balance-by-card", cardHandler.FindYearlyBalanceByCardNumber))

	})
	return cardHandler
}

// FindMonthlyBalance godoc
// @Summary Get monthly balance data
// @Description Retrieve monthly balance data for a specific year
// @Tags Card Stats Balance
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseMonthlyBalance
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-balance/monthly-balance [get]
func (h *cardStatsBalanceHandleApi) FindMonthlyBalance(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetMonthlyBalanceCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindYearBalance{
		Year: int32(year),
	}

	res, err := h.card.FindMonthlyBalance(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthlyBalances(res)
	h.cache.SetMonthlyBalanceCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyBalance godoc
// @Summary Get yearly balance data
// @Description Retrieve yearly balance data for a specific year
// @Tags Card Stats Balance
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseYearlyBalance
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-balance/yearly-balance [get]
func (h *cardStatsBalanceHandleApi) FindYearlyBalance(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearlyBalanceCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindYearBalance{
		Year: int32(year),
	}

	res, err := h.card.FindYearlyBalance(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearlyBalances(res)
	h.cache.SetYearlyBalanceCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyBalanceByCardNumber godoc
// @Summary Get monthly balance data by card number
// @Description Retrieve monthly balance data for a specific year and card number
// @Tags Card Stats Balance
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseMonthlyBalance
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-balance/monthly-balance-by-card [get]
func (h *cardStatsBalanceHandleApi) FindMonthlyBalanceByCardNumber(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	cardNumber := r.URL.Query().Get("card_number")
	if cardNumber == "" {
		return errors.NewBadRequestError("card_number is required")
	}

	ctx := r.Context()

	reqCache := &requests.MonthYearCardNumberCard{
		Year:       year,
		CardNumber: cardNumber,
	}

	cachedData, found := h.cache.GetMonthlyBalanceByNumberCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindYearBalanceCardNumber{
		Year:       int32(year),
		CardNumber: cardNumber,
	}

	res, err := h.card.FindMonthlyBalanceByCardNumber(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthlyBalances(res)
	h.cache.SetMonthlyBalanceByNumberCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyBalanceByCardNumber godoc
// @Summary Get yearly balance data by card number
// @Description Retrieve yearly balance data for a specific year and card number
// @Tags Card Stats Balance
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseYearlyBalance
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-balance/yearly-balance-by-card [get]
func (h *cardStatsBalanceHandleApi) FindYearlyBalanceByCardNumber(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	cardNumber := r.URL.Query().Get("card_number")
	if cardNumber == "" {
		return errors.NewBadRequestError("card_number is required")
	}

	ctx := r.Context()

	reqCache := &requests.MonthYearCardNumberCard{
		Year:       year,
		CardNumber: cardNumber,
	}

	cachedData, found := h.cache.GetYearlyBalanceByNumberCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindYearBalanceCardNumber{
		Year:       int32(year),
		CardNumber: cardNumber,
	}

	res, err := h.card.FindYearlyBalanceByCardNumber(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearlyBalances(res)
	h.cache.SetYearlyBalanceByNumberCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
