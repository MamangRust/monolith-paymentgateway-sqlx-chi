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

type cardStatsTopupHandleApi struct {
	card pb.CardStatsTopupServiceClient

	logger logger.LoggerInterface
	mapper apimapper.CardStatsAmountResponseMapper

	cache card_cache.CardMencache

	apiHandler apierror.ApiHandler
}

type cardStatsTopupHandleApiDeps struct {
	client pb.CardStatsTopupServiceClient
	router chi.Router

	logger logger.LoggerInterface
	mapper apimapper.CardStatsAmountResponseMapper

	cache card_cache.CardMencache

	apiHandler apierror.ApiHandler
}

func NewCardStatsTopupHandleApi(
	params *cardStatsTopupHandleApiDeps,
) *cardStatsTopupHandleApi {

	cardHandler := &cardStatsTopupHandleApi{
		card:       params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/card-stats-topup", func(routerCard chi.Router) {

		routerCard.Get("/monthly-topup-amount", params.apiHandler.Handle("find-monthly-topup-amount", cardHandler.FindMonthlyTopupAmount))
		routerCard.Get("/yearly-topup-amount", params.apiHandler.Handle("find-yearly-topup-amount", cardHandler.FindYearlyTopupAmount))

		routerCard.Get("/monthly-topup-amount-by-card", params.apiHandler.Handle("find-monthly-topup-amount-by-card", cardHandler.FindMonthlyTopupAmountByCardNumber))
		routerCard.Get("/yearly-topup-amount-by-card", params.apiHandler.Handle("find-yearly-topup-amount-by-card", cardHandler.FindYearlyTopupAmountByCardNumber))

	})
	return cardHandler
}

// FindMonthlyTopupAmount godoc
// @Summary Get monthly topup amount data
// @Description Retrieve monthly topup amount data for a specific year
// @Tags Card Stats Topup
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseMonthlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card/monthly-topup-amount [get]
func (h *cardStatsTopupHandleApi) FindMonthlyTopupAmount(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetMonthlyTopupCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmount{
		Year: int32(year),
	}

	res, err := h.card.FindMonthlyTopupAmount(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthlyAmounts(res)
	h.cache.SetMonthlyTopupCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTopupAmount godoc
// @Summary Get yearly topup amount data
// @Description Retrieve yearly topup amount data for a specific year
// @Tags Card Stats Topup
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseYearlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-topup/yearly-topup-amount [get]
func (h *cardStatsTopupHandleApi) FindYearlyTopupAmount(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearlyTopupCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmount{
		Year: int32(year),
	}

	res, err := h.card.FindYearlyTopupAmount(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearlyAmounts(res)
	h.cache.SetYearlyTopupCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTopupAmountByCardNumber godoc
// @Summary Get monthly topup amount data by card number
// @Description Retrieve monthly topup amount data for a specific year and card number
// @Tags Card Stats Topup
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseMonthlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card/monthly-topup-amount-by-card [get]
func (h *cardStatsTopupHandleApi) FindMonthlyTopupAmountByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	cachedData, found := h.cache.GetMonthlyTopupByNumberCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmountCardNumber{
		Year:       int32(year),
		CardNumber: cardNumber,
	}

	res, err := h.card.FindMonthlyTopupAmountByCardNumber(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthlyAmounts(res)
	h.cache.SetMonthlyTopupByNumberCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTopupAmountByCardNumber godoc
// @Summary Get yearly topup amount data by card number
// @Description Retrieve yearly topup amount data for a specific year and card number
// @Tags Card Stats Topup
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseYearlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card/yearly-topup-amount-by-card [get]
func (h *cardStatsTopupHandleApi) FindYearlyTopupAmountByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	cachedData, found := h.cache.GetYearlyTopupByNumberCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmountCardNumber{
		Year:       int32(year),
		CardNumber: cardNumber,
	}

	res, err := h.card.FindYearlyTopupAmountByCardNumber(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearlyAmounts(res)
	h.cache.SetYearlyTopupByNumberCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
