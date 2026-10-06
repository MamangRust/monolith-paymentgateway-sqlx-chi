package merchanthandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pbmerchant "github.com/MamangRust/monolith-payment-gateway-pb/merchant"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/merchant/stats"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/merchant"
	"github.com/go-chi/chi/v5"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	merchant_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/merchant"
)

type merchantStatsMethodHandleApi struct {
	client pb.MerchantStatsMethodServiceClient

	logger logger.LoggerInterface

	mapper apimapper.MerchantStatsMethodResponseMapper

	cache merchant_cache.MerchantMencache

	apiHandler apierror.ApiHandler
}

type merchantStatsMethodHandleDeps struct {
	client pb.MerchantStatsMethodServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.MerchantStatsMethodResponseMapper

	cache merchant_cache.MerchantMencache

	apiHandler apierror.ApiHandler
}

func NewMerchantStatsMethodHandleApi(params *merchantStatsMethodHandleDeps) *merchantStatsMethodHandleApi {

	merchantHandler := &merchantStatsMethodHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		apiHandler: params.apiHandler,
		cache:      params.cache,
	}

	params.router.Route("/api/merchant-stats-method", func(routerMerchant chi.Router) {

		routerMerchant.Get("/monthly-payment-methods", params.apiHandler.Handle("find-monthly-payment-methods", merchantHandler.FindMonthlyPaymentMethodsMerchant))
		routerMerchant.Get("/yearly-payment-methods", params.apiHandler.Handle("find-yearly-payment-methods", merchantHandler.FindYearlyPaymentMethodMerchant))

		routerMerchant.Get("/monthly-payment-methods-by-merchant", params.apiHandler.Handle("find-monthly-payment-methods-by-merchant", merchantHandler.FindMonthlyPaymentMethodByMerchants))
		routerMerchant.Get("/yearly-payment-methods-by-merchant", params.apiHandler.Handle("find-yearly-payment-methods-by-merchant", merchantHandler.FindYearlyPaymentMethodByMerchants))
		routerMerchant.Get("/monthly-payment-methods-by-apikey", params.apiHandler.Handle("find-monthly-payment-methods-by-apikey", merchantHandler.FindMonthlyPaymentMethodByApikeys))
		routerMerchant.Get("/yearly-payment-methods-by-apikey", params.apiHandler.Handle("find-yearly-payment-methods-by-apikey", merchantHandler.FindYearlyPaymentMethodByApikeys))
	})
	return merchantHandler
}

// FindMonthlyPaymentMethodsMerchant godoc
// @Summary Find monthly payment methods for a merchant
// @Tags Merchant Stats Method
// @Security Bearer
// @Description Retrieve monthly payment methods for a merchant by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseMerchantMonthlyPaymentMethod "Monthly payment methods"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly payment methods"
// @Router /api/merchant-stats-method/monthly-payment-methods [get]
func (h *merchantStatsMethodHandleApi) FindMonthlyPaymentMethodsMerchant(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetMonthlyPaymentMethodsMerchantCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	req := &pbmerchant.FindYearMerchant{
		Year: int32(year),
	}

	res, err := h.client.FindMonthlyPaymentMethodsMerchant(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthlyPaymentMethods(res)
	h.cache.SetMonthlyPaymentMethodsMerchantCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyPaymentMethodMerchant godoc.
// @Summary Find yearly payment methods for a merchant
// @Tags Merchant Stats Method
// @Security Bearer
// @Description Retrieve yearly payment methods for a merchant by year.
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseMerchantYearlyPaymentMethod "Yearly payment methods"
// @Failure 400 {object} response.ErrorResponse "Invalid year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly payment methods"
// @Router /api/merchant-stats-method/monthly-amount [get]
func (h *merchantStatsMethodHandleApi) FindYearlyPaymentMethodMerchant(w http.ResponseWriter, r *http.Request) error {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetYearlyPaymentMethodMerchantCache(ctx, year)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	req := &pbmerchant.FindYearMerchant{
		Year: int32(year),
	}

	res, err := h.client.FindYearlyPaymentMethodMerchant(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearlyPaymentMethods(res)
	h.cache.SetYearlyPaymentMethodMerchantCache(ctx, year, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyPaymentMethodByMerchants godoc.
// @Summary Find monthly payment methods for a specific merchant
// @Tags Merchant Stats Method
// @Security Bearer
// @Description Retrieve monthly payment methods for a specific merchant by year.
// @Accept json
// @Produce json
// @Param merchant_id query int true "Merchant ID"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseMerchantMonthlyPaymentMethod "Monthly payment methods"
// @Failure 400 {object} response.ErrorResponse "Invalid merchant ID or year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly payment methods"
// @Router /api/merchant-stats-method/monthly-payment-methods-by-merchant [get]
func (h *merchantStatsMethodHandleApi) FindMonthlyPaymentMethodByMerchants(w http.ResponseWriter, r *http.Request) error {
	merchantIDStr := r.URL.Query().Get("merchant_id")
	yearStr := r.URL.Query().Get("year")

	merchantID, err := strconv.Atoi(merchantIDStr)
	if err != nil || merchantID <= 0 {
		return errors.NewBadRequestError("merchant_id is required and must be a positive integer")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	reqCache := &requests.MonthYearPaymentMethodMerchant{
		MerchantID: merchantID,
		Year:       year,
	}

	cachedData, found := h.cache.GetMonthlyPaymentMethodByMerchantsCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbmerchant.FindYearMerchantById{
		MerchantId: int32(merchantID),
		Year:       int32(year),
	}

	res, err := h.client.FindMonthlyPaymentMethodByMerchants(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthlyPaymentMethods(res)
	h.cache.SetMonthlyPaymentMethodByMerchantsCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyPaymentMethodByMerchants godoc.
// @Summary Find yearly payment methods for a specific merchant
// @Tags Merchant Stats Method
// @Security Bearer
// @Description Retrieve yearly payment methods for a specific merchant by year.
// @Accept json
// @Produce json
// @Param merchant_id query int true "Merchant ID"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseMerchantYearlyPaymentMethod "Yearly payment methods"
// @Failure 400 {object} response.ErrorResponse "Invalid merchant ID or year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly payment methods"
// @Router /api/merchant-stats-method/yearly-payment-methods-by-merchant [get]
func (h *merchantStatsMethodHandleApi) FindYearlyPaymentMethodByMerchants(w http.ResponseWriter, r *http.Request) error {
	merchantIDStr := r.URL.Query().Get("merchant_id")
	yearStr := r.URL.Query().Get("year")

	merchantID, err := strconv.Atoi(merchantIDStr)
	if err != nil || merchantID <= 0 {
		return errors.NewBadRequestError("merchant_id is required and must be a positive integer")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	reqCache := &requests.MonthYearPaymentMethodMerchant{
		MerchantID: merchantID,
		Year:       year,
	}

	cachedData, found := h.cache.GetYearlyPaymentMethodByMerchantsCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbmerchant.FindYearMerchantById{
		MerchantId: int32(merchantID),
		Year:       int32(year),
	}

	res, err := h.client.FindYearlyPaymentMethodByMerchants(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearlyPaymentMethods(res)
	h.cache.SetYearlyPaymentMethodByMerchantsCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindMonthlyPaymentMethodByApikeys godoc.
// @Summary Find monthly payment methods for a specific merchant
// @Tags Merchant Stats Method
// @Security Bearer
// @Description Retrieve monthly payment methods for a specific merchant by year.
// @Accept json
// @Produce json
// @Param merchant_id query int true "Merchant ID"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseMerchantMonthlyPaymentMethod "Monthly payment methods"
// @Failure 400 {object} response.ErrorResponse "Invalid merchant ID or year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve monthly payment methods"
// @Router /api/merchant-stats-method/monthly-payment-methods-by-apikey [get]
func (h *merchantStatsMethodHandleApi) FindMonthlyPaymentMethodByApikeys(w http.ResponseWriter, r *http.Request) error {
	api_key := r.URL.Query().Get("api_key")
	yearStr := r.URL.Query().Get("year")

	if api_key == "" {
		return errors.NewBadRequestError("api_key is required")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	reqCache := &requests.MonthYearPaymentMethodApiKey{
		Apikey: api_key,
		Year:   year,
	}

	cachedData, found := h.cache.GetMonthlyPaymentMethodByApikeysCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbmerchant.FindYearMerchantByApikey{
		ApiKey: api_key,
		Year:   int32(year),
	}

	res, err := h.client.FindMonthlyPaymentMethodByApikey(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMonthlyPaymentMethods(res)
	h.cache.SetMonthlyPaymentMethodByApikeysCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindYearlyPaymentMethodByApikeys godoc.
// @Summary Find yearly payment methods for a specific merchant
// @Tags Merchant Stats Method
// @Security Bearer
// @Description Retrieve yearly payment methods for a specific merchant by year.
// @Accept json
// @Produce json
// @Param merchant_id query int true "Merchant ID"
// @Param year query int true "Year"
// @Success 200 {object} response.ApiResponseMerchantYearlyPaymentMethod "Yearly payment methods"
// @Failure 400 {object} response.ErrorResponse "Invalid merchant ID or year"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve yearly payment methods"
// @Router /api/merchant-stats-method/yearly-payment-methods-by-apikey [get]
func (h *merchantStatsMethodHandleApi) FindYearlyPaymentMethodByApikeys(w http.ResponseWriter, r *http.Request) error {
	api_key := r.URL.Query().Get("api_key")
	yearStr := r.URL.Query().Get("year")

	if api_key == "" {
		return errors.NewBadRequestError("api_key is required")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		return errors.NewBadRequestError("year is required and must be a positive integer")
	}

	ctx := r.Context()

	reqCache := &requests.MonthYearPaymentMethodApiKey{
		Apikey: api_key,
		Year:   year,
	}

	cachedData, found := h.cache.GetYearlyPaymentMethodByApikeysCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pbmerchant.FindYearMerchantByApikey{
		ApiKey: api_key,
		Year:   int32(year),
	}

	res, err := h.client.FindYearlyPaymentMethodByApikey(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseYearlyPaymentMethods(res)
	h.cache.SetYearlyPaymentMethodByApikeysCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
