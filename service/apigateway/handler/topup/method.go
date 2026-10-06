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

type topupStatsMethodHandleApi struct {
	client pb.TopupStatsMethodServiceClient

	logger logger.LoggerInterface

	mapper apimapper.TopupStatsMethodResponseMapper

	cache topup_cache.TopupMencach

	apiHandler apierror.ApiHandler
}

type topupStatsMethodHandleDeps struct {
	client pb.TopupStatsMethodServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.TopupStatsMethodResponseMapper

	cache topup_cache.TopupMencach

	apiHandler apierror.ApiHandler
}

func NewTopupStatsMethodHandleApi(params *topupStatsMethodHandleDeps) *topupStatsMethodHandleApi {

	topupHandler := &topupStatsMethodHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/topup-stats-method", func(routerTopup chi.Router) {

		routerTopup.Get("/monthly-methods", params.apiHandler.Handle("find-monthly-topup-methods", topupHandler.FindMonthlyTopupMethods))
		routerTopup.Get("/yearly-methods", params.apiHandler.Handle("find-yearly-topup-methods", topupHandler.FindYearlyTopupMethods))
		routerTopup.Get("/monthly-methods-by-card", params.apiHandler.Handle("find-monthly-topup-methods-by-card", topupHandler.FindMonthlyTopupMethodsByCardNumber))
		routerTopup.Get("/yearly-methods-by-card", params.apiHandler.Handle("find-yearly-topup-methods-by-card", topupHandler.FindYearlyTopupMethodsByCardNumber))

	})
	return topupHandler
}

// FindMonthlyTopupMethods retrieves the monthly top-up methods for a specific year.
// @Summary Get monthly top-up methods
// @Tags Topup Method
// @Security Bearer
// @Description Retrieve the monthly top-up methods for a specific year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTopupMonthMethod "Monthly top-up methods"
// @Failure 400 {object} response.ErrorResponse "Invalid year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly top-up methods"
// @Router /api/topup-stats-method/monthly-methods [get]
func (h *topupStatsMethodHandleApi) FindMonthlyTopupMethods(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetMonthlyTopupMethodsCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTopupMethods(ctx, &pbtopup.FindYearTopupStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly topup methods", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTopupMonthMethod(res)
	h.cache.SetMonthlyTopupMethodsCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTopupMethods retrieves the yearly top-up methods for a specific year.
// @Summary Get yearly top-up methods
// @Tags Topup Method
// @Security Bearer
// @Description Retrieve the yearly top-up methods for a specific year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTopupYearMethod "Yearly top-up methods"
// @Failure 400 {object} response.ErrorResponse "Invalid year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly top-up methods"
// @Router /api/topup-stats-method/yearly-methods [get]
func (h *topupStatsMethodHandleApi) FindYearlyTopupMethods(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("invalid year parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearlyTopupMethodsCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTopupMethods(ctx, &pbtopup.FindYearTopupStatus{
		Year: int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly topup methods", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTopupYearMethod(res)
	h.cache.SetYearlyTopupMethodsCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyTopupMethodsByCardNumber retrieves the monthly top-up methods for a specific card number and year.
// @Summary Get monthly top-up methods by card number
// @Tags Topup Method
// @Security Bearer
// @Description Retrieve the monthly top-up methods for a specific card number and year.
// @Accept json
// @Produce json
// @Param card_number query string true "Card Number"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTopupMonthMethod "Monthly top-up methods by card number"
// @Failure 400 {object} response.ErrorResponse "Invalid card number or year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly top-up methods by card number"
// @Router /api/topup-stats-method/monthly-methods-by-card [get]
func (h *topupStatsMethodHandleApi) FindMonthlyTopupMethodsByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	cachedData, found := h.cache.GetMonthlyTopupMethodsByCardNumberCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindMonthlyTopupMethodsByCardNumber(ctx, &pbtopup.FindYearTopupCardNumber{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve monthly topup methods by card number", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTopupMonthMethod(res)
	h.cache.SetMonthlyTopupMethodsByCardNumberCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyTopupMethodsByCardNumber retrieves the yearly top-up methods for a specific card number and year.
// @Summary Get yearly top-up methods by card number
// @Tags Topup Method
// @Security Bearer
// @Description Retrieve the yearly top-up methods for a specific card number and year.
// @Accept json
// @Produce json
// @Param card_number query string true "Card Number"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseTopupYearMethod "Yearly top-up methods by card number"
// @Failure 400 {object} response.ErrorResponse "Invalid card number or year parameter"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly top-up methods by card number"
// @Router /api/topup-stats-method/yearly-methods-by-card [get]
func (h *topupStatsMethodHandleApi) FindYearlyTopupMethodsByCardNumber(w http.ResponseWriter, r *http.Request) error {
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

	cachedData, found := h.cache.GetYearlyTopupMethodsByCardNumberCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindYearlyTopupMethodsByCardNumber(ctx, &pbtopup.FindYearTopupCardNumber{
		CardNumber: cardNumber,
		Year:       int32(year),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve yearly topup methods by card number", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTopupYearMethod(res)
	h.cache.SetYearlyTopupMethodsByCardNumberCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
