package cardhandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pbcard "github.com/MamangRust/monolith-payment-gateway-pb/card"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/card/stats"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/card"
	"github.com/go-chi/chi/v5"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	card_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/card"
)

type cardStatsTransactionHandleApi struct {
	card pb.CardStatsTransactionServiceClient

	logger     logger.LoggerInterface
	apiHandler apierror.ApiHandler

	cache  card_cache.CardMencache
	mapper apimapper.CardStatsAmountResponseMapper
}

type cardStatsTransactionHandleApiDeps struct {
	client pb.CardStatsTransactionServiceClient
	router chi.Router

	apiHandler apierror.ApiHandler

	cache  card_cache.CardMencache
	logger logger.LoggerInterface
	mapper apimapper.CardStatsAmountResponseMapper
}

func NewCardStatsTransactionHandleApi(
	params *cardStatsTransactionHandleApiDeps,
) *cardStatsTransactionHandleApi {

	cardHandler := &cardStatsTransactionHandleApi{
		card:       params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/card-stats-transaction", func(routerCard chi.Router) {

		routerCard.Get("/monthly-transaction-amount", params.apiHandler.Handle("find-monthly-transaction-amount", cardHandler.FindMonthlyTransactionAmount))
		routerCard.Get("/yearly-transaction-amount", params.apiHandler.Handle("find-yearly-transaction-amount", cardHandler.FindYearlyTransactionAmount))
		routerCard.Get("/monthly-transaction-amount-by-card", params.apiHandler.Handle("find-monthly-transaction-amount-by-card", cardHandler.FindMonthlyTransactionAmountByCardNumber))
		routerCard.Get("/yearly-transaction-amount-by-card", params.apiHandler.Handle("find-yearly-transaction-amount-by-card", cardHandler.FindYearlyTransactionAmountByCardNumber))

	})
	return cardHandler
}

// FindMonthlyTransactionAmount godoc
// @Summary Get monthly transaction amount data
// @Description Retrieve monthly transaction amount data for a specific year
// @Tags Card Stats Transaction
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseMonthlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-transaction/monthly-transaction-amount [get]
func (h *cardStatsTransactionHandleApi) FindMonthlyTransactionAmount(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetMonthlyTransactionCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmount{
		Year: int32(year),
	}

	res, err := h.card.FindMonthlyTransactionAmount(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthlyAmounts(res)
	h.cache.SetMonthlyTransactionCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransactionAmount godoc
// @Summary Get yearly transaction amount data
// @Description Retrieve yearly transaction amount data for a specific year
// @Tags Card Stats Transaction
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseYearlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-transaction/yearly-transaction-amount [get]
func (h *cardStatsTransactionHandleApi) FindYearlyTransactionAmount(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearlyTransactionCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmount{
		Year: int32(year),
	}

	res, err := h.card.FindYearlyTransactionAmount(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearlyAmounts(res)
	h.cache.SetYearlyTransactionCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTransactionAmountByCardNumber godoc
// @Summary Get monthly transaction amount data by card number
// @Description Retrieve monthly transaction amount data for a specific year and card number
// @Tags Card Stats Transaction
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseMonthlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-transaction/monthly-transaction-amount-by-card [get]
func (h *cardStatsTransactionHandleApi) FindMonthlyTransactionAmountByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	cachedData, found := h.cache.GetMonthlyTransactionByNumberCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmountCardNumber{
		Year:       int32(year),
		CardNumber: cardNumber,
	}

	res, err := h.card.FindMonthlyTransactionAmountByCardNumber(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthlyAmounts(res)
	h.cache.SetMonthlyTransactionByNumberCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransactionAmountByCardNumber godoc
// @Summary Get yearly transaction amount data by card number
// @Description Retrieve yearly transaction amount data for a specific year and card number
// @Tags Card Stats Transaction
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseYearlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-transaction/yearly-transaction-amount-by-card [get]
func (h *cardStatsTransactionHandleApi) FindYearlyTransactionAmountByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	cachedData, found := h.cache.GetYearlyTransactionByNumberCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmountCardNumber{
		Year:       int32(year),
		CardNumber: cardNumber,
	}

	res, err := h.card.FindYearlyTransactionAmountByCardNumber(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearlyAmounts(res)
	h.cache.SetYearlyTransactionByNumberCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
