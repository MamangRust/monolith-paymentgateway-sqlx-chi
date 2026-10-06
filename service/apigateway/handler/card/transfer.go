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

type cardStatsTransferHandleApi struct {
	card pb.CardStatsTransferServiceClient

	logger logger.LoggerInterface
	mapper apimapper.CardStatsAmountResponseMapper

	apiHandler apierror.ApiHandler

	cache card_cache.CardMencache
}

type cardStatsTransferHandleApiDeps struct {
	client     pb.CardStatsTransferServiceClient
	router     chi.Router
	apiHandler apierror.ApiHandler

	cache  card_cache.CardMencache
	logger logger.LoggerInterface
	mapper apimapper.CardStatsAmountResponseMapper
}

func NewCardStatsTransferHandleApi(
	params *cardStatsTransferHandleApiDeps,
) *cardStatsTransferHandleApi {

	cardHandler := &cardStatsTransferHandleApi{
		card:       params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/card-stats-transfer", func(routerCard chi.Router) {

		routerCard.Get("/monthly-transfer-sender-amount", params.apiHandler.Handle("find-monthly-transfer-sender-amount", cardHandler.FindMonthlyTransferSenderAmount))
		routerCard.Get("/yearly-transfer-sender-amount", params.apiHandler.Handle("find-yearly-transfer-sender-amount", cardHandler.FindYearlyTransferSenderAmount))
		routerCard.Get("/monthly-transfer-receiver-amount", params.apiHandler.Handle("find-monthly-transfer-receiver-amount", cardHandler.FindMonthlyTransferReceiverAmount))
		routerCard.Get("/yearly-transfer-receiver-amount", params.apiHandler.Handle("find-yearly-transfer-receiver-amount", cardHandler.FindYearlyTransferReceiverAmount))

		routerCard.Get("/monthly-transfer-sender-amount-by-card", params.apiHandler.Handle("find-monthly-transfer-sender-amount-by-card", cardHandler.FindMonthlyTransferSenderAmountByCardNumber))
		routerCard.Get("/yearly-transfer-sender-amount-by-card", params.apiHandler.Handle("find-yearly-transfer-sender-amount-by-card", cardHandler.FindYearlyTransferSenderAmountByCardNumber))
		routerCard.Get("/monthly-transfer-receiver-amount-by-card", params.apiHandler.Handle("find-monthly-transfer-receiver-amount-by-card", cardHandler.FindMonthlyTransferReceiverAmountByCardNumber))
		routerCard.Get("/yearly-transfer-receiver-amount-by-card", params.apiHandler.Handle("find-yearly-transfer-receiver-amount-by-card", cardHandler.FindYearlyTransferReceiverAmountByCardNumber))

	})
	return cardHandler
}

// FindMonthlyTransferSenderAmount godoc
// @Summary Get monthly transfer sender amount data
// @Description Retrieve monthly transfer sender amount data for a specific year
// @Tags Card Stats Transfer
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseMonthlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-transfer-stats-transfer/monthly-transfer-sender-amount [get]
func (h *cardStatsTransferHandleApi) FindMonthlyTransferSenderAmount(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetMonthlyTransferSenderCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmount{
		Year: int32(year),
	}

	res, err := h.card.FindMonthlyTransferSenderAmount(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthlyAmounts(res)
	h.cache.SetMonthlyTransferSenderCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransferSenderAmount godoc
// @Summary Get yearly transfer sender amount data
// @Description Retrieve yearly transfer sender amount data for a specific year
// @Tags Card Stats Transfer
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseYearlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transfer/yearly-transfer-sender-amount [get]
func (h *cardStatsTransferHandleApi) FindYearlyTransferSenderAmount(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearlyTransferSenderCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmount{
		Year: int32(year),
	}

	res, err := h.card.FindYearlyTransferSenderAmount(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearlyAmounts(res)
	h.cache.SetYearlyTransferSenderCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTransferReceiverAmount godoc
// @Summary Get monthly transfer receiver amount data
// @Description Retrieve monthly transfer receiver amount data for a specific year
// @Tags Card Stats Transfer
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseMonthlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-transfer-stats-transfer/monthly-transfer-receiver-amount [get]
func (h *cardStatsTransferHandleApi) FindMonthlyTransferReceiverAmount(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetMonthlyTransferReceiverCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmount{
		Year: int32(year),
	}

	res, err := h.card.FindMonthlyTransferReceiverAmount(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthlyAmounts(res)
	h.cache.SetMonthlyTransferReceiverCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransferReceiverAmount godoc
// @Summary Get yearly transfer receiver amount data
// @Description Retrieve yearly transfer receiver amount data for a specific year
// @Tags Card Stats Transfer
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseYearlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-transfer-stats-transfer/yearly-transfer-receiver-amount [get]
func (h *cardStatsTransferHandleApi) FindYearlyTransferReceiverAmount(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearlyTransferReceiverCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmount{
		Year: int32(year),
	}

	res, err := h.card.FindYearlyTransferReceiverAmount(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearlyAmounts(res)
	h.cache.SetYearlyTransferReceiverCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTransferSenderAmountByCardNumber godoc
// @Summary Get monthly transfer sender amount data by card number
// @Description Retrieve monthly transfer sender amount data for a specific year and card number
// @Tags Card Stats Transfer
// @Security Bearer
// @Produce json
// @Param year query int true "Year"
// @Param card_number query string true "Card Number"
// @Success 200 {object} response.ApiResponseMonthlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-transfer/monthly-transfer-sender-amount-by-card [get]
func (h *cardStatsTransferHandleApi) FindMonthlyTransferSenderAmountByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	cachedData, found := h.cache.GetMonthlyTransferBySenderCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmountCardNumber{
		Year:       int32(year),
		CardNumber: cardNumber,
	}

	res, err := h.card.FindMonthlyTransferSenderAmountByCardNumber(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthlyAmounts(res)
	h.cache.SetMonthlyTransferBySenderCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransferSenderAmountByCardNumber godoc
// @Summary Get yearly transfer sender amount by card number
// @Description Retrieve the total amount sent by a specific card number in a given year
// @Tags Card Stats Transfer
// @Security Bearer
// @Accept json
// @Produce json
// @Param year query int true "Year for which the data is requested"
// @Param card_number query string true "Card number for which the data is requested"
// @Success 200 {object} response.ApiResponseYearlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-transfer/yearly-transfer-sender-amount-by-card [get]
func (h *cardStatsTransferHandleApi) FindYearlyTransferSenderAmountByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	cachedData, found := h.cache.GetYearlyTransferBySenderCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmountCardNumber{
		Year:       int32(year),
		CardNumber: cardNumber,
	}

	res, err := h.card.FindYearlyTransferSenderAmountByCardNumber(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearlyAmounts(res)
	h.cache.SetYearlyTransferBySenderCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTransferReceiverAmountByCardNumber godoc
// @Summary Get monthly transfer receiver amount by card number
// @Description Retrieve the total amount received by a specific card number in a given year, broken down by month
// @Tags Card Stats Transfer
// @Security Bearer
// @Accept json
// @Produce json
// @Param year query int true "Year for which the data is requested"
// @Param card_number query string true "Card number for which the data is requested"
// @Success 200 {object} response.ApiResponseYearlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-transfer/monthly-transfer-receiver-amount-by-card [get]
func (h *cardStatsTransferHandleApi) FindMonthlyTransferReceiverAmountByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	cachedData, found := h.cache.GetMonthlyTransferByReceiverCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmountCardNumber{
		Year:       int32(year),
		CardNumber: cardNumber,
	}

	res, err := h.card.FindMonthlyTransferReceiverAmountByCardNumber(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthlyAmounts(res)
	h.cache.SetMonthlyTransferByReceiverCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTransferReceiverAmountByCardNumber godoc
// @Summary Get yearly transfer receiver amount by card number
// @Description Retrieve the total amount received by a specific card number in a given year
// @Tags Card Stats Transfer
// @Security Bearer
// @Accept json
// @Produce json
// @Param year query int true "Year for which the data is requested"
// @Param card_number query string true "Card number for which the data is requested"
// @Success 200 {object} response.ApiResponseYearlyAmount
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/card-stats-transfer/yearly-transfer-receiver-amount-by-card [get]
func (h *cardStatsTransferHandleApi) FindYearlyTransferReceiverAmountByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	cachedData, found := h.cache.GetYearlyTransferByReceiverCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbcard.FindYearAmountCardNumber{
		Year:       int32(year),
		CardNumber: cardNumber,
	}

	res, err := h.card.FindYearlyTransferReceiverAmountByCardNumber(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearlyAmounts(res)
	h.cache.SetYearlyTransferByReceiverCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
