package topuphandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pbtopup "github.com/MamangRust/monolith-payment-gateway-pb/topup"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/topup/stats"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/topup"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	topup_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/topup"
)

type topupStatsAmountHandleApi struct {
	client pb.TopupStatsAmountServiceClient

	logger logger.LoggerInterface

	mapper apimapper.TopupStatsAmountResponseMapper

	cache topup_cache.TopupMencach

	apiHandler apierror.ApiHandler
}

type topupStatsAmountHandleDeps struct {
	client pb.TopupStatsAmountServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.TopupStatsAmountResponseMapper

	cache topup_cache.TopupMencach

	apiHandler apierror.ApiHandler
}

func NewTopupStatsAmountHandleApi(params *topupStatsAmountHandleDeps) *topupStatsAmountHandleApi {

	topupHandler := &topupStatsAmountHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/topup-stats-amount", func(routerTopup chi.Router) {

		routerTopup.Get("/monthly-amounts", params.apiHandler.Handle("find-monthly-topup-amounts", topupHandler.FindMonthlyTopupAmounts))
		routerTopup.Get("/yearly-amounts", params.apiHandler.Handle("find-yearly-topup-amounts", topupHandler.FindYearlyTopupAmounts))

		routerTopup.Get("/monthly-amounts-by-card", params.apiHandler.Handle("find-monthly-topup-amounts-by-card", topupHandler.FindMonthlyTopupAmountsByCardNumber))
		routerTopup.Get("/yearly-amounts-by-card", params.apiHandler.Handle("find-yearly-topup-amounts-by-card", topupHandler.FindYearlyTopupAmountsByCardNumber))

	})
	return topupHandler
}

// FindMonthlyTopupAmounts retrieves the monthly top-up amounts for a specific year.
// @Summary Get monthly top-up amounts
// @Tags Topup Amount
// @Security Bearer
// @Description Retrieve the monthly top-up amounts for a specific year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTopupMonthAmount "Monthly top-up amounts"
// @Failure 400 {object} response.ErrorResponse "Invalid year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly top-up amounts"
// @Router /api/topup-stats-amount/monthly-amounts [get]
func (h *topupStatsAmountHandleApi) FindMonthlyTopupAmounts(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetMonthlyTopupAmountsCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTopupAmounts(ctx, &pbtopup.FindYearTopupStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly topup amounts", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTopupMonthAmount(res)
	h.cache.SetMonthlyTopupAmountsCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTopupAmounts retrieves the yearly top-up amounts for a specific year.
// @Summary Get yearly top-up amounts
// @Tags Topup Amount
// @Security Bearer
// @Description Retrieve the yearly top-up amounts for a specific year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTopupYearAmount "Yearly top-up amounts"
// @Failure 400 {object} response.ErrorResponse "Invalid year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly top-up amounts"
// @Router /api/topup-stats-amounts/yearly-amounts [get]
func (h *topupStatsAmountHandleApi) FindYearlyTopupAmounts(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearlyTopupAmountsCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTopupAmounts(ctx, &pbtopup.FindYearTopupStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly topup amounts", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTopupYearAmount(res)
	h.cache.SetYearlyTopupAmountsCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTopupAmountsByCardNumber retrieves the monthly top-up amounts for a specific card number and year.
// @Summary Get monthly top-up amounts by card number
// @Tags Topup Amount
// @Security Bearer
// @Description Retrieve the monthly top-up amounts for a specific card number and year.
// @Accept json
// @Produce json
// @Param card_number query string true "Card Number"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTopupMonthAmount "Monthly top-up amounts by card number"
// @Failure 400 {object} response.ErrorResponse "Invalid card number or year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly top-up amounts by card number"
// @Router /api/topup-stats-amounts/monthly-amounts-by-card [get]
func (h *topupStatsAmountHandleApi) FindMonthlyTopupAmountsByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.YearMonthMethod{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetMonthlyTopupAmountsByCardNumberCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTopupAmountsByCardNumber(ctx, &pbtopup.FindYearTopupCardNumber{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly topup amounts by card number", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTopupMonthAmount(res)
	h.cache.SetMonthlyTopupAmountsByCardNumberCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTopupAmountsByCardNumber retrieves the yearly top-up amounts for a specific card number and year.
// @Summary Get yearly top-up amounts by card number
// @Tags Topup Amount
// @Security Bearer
// @Description Retrieve the yearly top-up amounts for a specific card number and year.
// @Accept json
// @Produce json
// @Param card_number query string true "Card Number"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTopupYearAmount "Yearly top-up amounts by card number"
// @Failure 400 {object} response.ErrorResponse "Invalid card number or year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly top-up amounts by card number"
// @Router /api/topup-stats-amounts/yearly-amounts-by-card [get]
func (h *topupStatsAmountHandleApi) FindYearlyTopupAmountsByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.YearMonthMethod{
		CardNumber: cardNumber,
		Year:       year,
	}

	cachedData, found := h.cache.GetYearlyTopupAmountsByCardNumberCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTopupAmountsByCardNumber(ctx, &pbtopup.FindYearTopupCardNumber{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly topup amounts by card number", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTopupYearAmount(res)
	h.cache.SetYearlyTopupAmountsByCardNumberCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
