package merchanthandler

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/merchant"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/merchant"
	"github.com/go-chi/chi/v5"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	merchant_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/merchant"
)

type merchantQueryHandleApi struct {
	client pb.MerchantQueryServiceClient

	logger logger.LoggerInterface
	mapper apimapper.MerchantQueryResponseMapper

	cache merchant_cache.MerchantMencache

	apiHandler apierror.ApiHandler
}

type merchantQueryHandleDeps struct {
	client pb.MerchantQueryServiceClient
	router chi.Router

	logger logger.LoggerInterface
	mapper apimapper.MerchantQueryResponseMapper

	cache merchant_cache.MerchantMencache

	apiHandler apierror.ApiHandler
}

func NewMerchantQueryHandleApi(params *merchantQueryHandleDeps) *merchantQueryHandleApi {

	merchantHandler := &merchantQueryHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/merchant-query", func(routerMerchant chi.Router) {

		routerMerchant.Get("/", httpx.Handler(merchantHandler.FindAll))
		routerMerchant.Get("/{id}", httpx.Handler(merchantHandler.FindById))
		routerMerchant.Get("/api-key", httpx.Handler(merchantHandler.FindByApiKey))
		routerMerchant.Get("/merchant-user", httpx.Handler(merchantHandler.FindByMerchantUserId))

		routerMerchant.Get("/active", httpx.Handler(merchantHandler.FindByActive))
		routerMerchant.Get("/trashed", httpx.Handler(merchantHandler.FindByTrashed))

	})
	return merchantHandler
}

// FindAll godoc
// @Summary Find all merchants
// @Tags Merchant Query
// @Security Bearer
// @Description Retrieve a list of all merchants
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Number of items per page" default(10)
// @Param search query string false "Search query"
// @Success 200 {object} response.ApiResponsePaginationMerchant "List of merchants"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve merchant data"
// @Router /api/merchant-query [get]
func (h *merchantQueryHandleApi) FindAll(w http.ResponseWriter, r *http.Request) error {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page <= 0 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("page_size"))
	if err != nil || pageSize <= 0 {
		pageSize = 10
	}

	search := r.URL.Query().Get("search")
	ctx := r.Context()

	reqCache := &requests.FindAllMerchants{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCachedMerchants(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindAllMerchantRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindAllMerchant(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsesMerchant(res)

	h.cache.SetCachedMerchants(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindById godoc
// @Summary Find a merchant by ID
// @Tags Merchant Query
// @Security Bearer
// @Description Retrieve a merchant by its ID.
// @Accept json
// @Produce json
// @Param id path int true "Merchant ID"
// @Success 200 {object} response.ApiResponseMerchant "Merchant data"
// @Failure 400 {object} response.ErrorResponse "Invalid merchant ID"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve merchant data"
// @Router /api/merchant-query/{id} [get]
func (h *merchantQueryHandleApi) FindById(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("id is required and must be an integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedMerchant(ctx, id)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindByIdMerchantRequest{
		MerchantId: int32(id),
	}

	res, err := h.client.FindByIdMerchant(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMerchant(res)

	h.cache.SetCachedMerchant(ctx, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindByApiKey godoc
// @Summary Find a merchant by API key
// @Tags Merchant Query
// @Security Bearer
// @Description Retrieve a merchant by its API key
// @Accept json
// @Produce json
// @Param api_key query string true "API key"
// @Success 200 {object} response.ApiResponseMerchant "Merchant data"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve merchant data"
// @Router /api/merchant-query/api-key [get]
func (h *merchantQueryHandleApi) FindByApiKey(w http.ResponseWriter, r *http.Request) error {
	apiKey := r.URL.Query().Get("api_key")
	if apiKey == "" {
		return errors.NewBadRequestError("api_key is required")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedMerchantByApiKey(ctx, apiKey)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindByApiKeyRequest{
		ApiKey: apiKey,
	}

	res, err := h.client.FindByApiKey(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMerchant(res)

	h.cache.SetCachedMerchantByApiKey(ctx, apiKey, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindByMerchantUserId godoc.
// @Summary Find a merchant by user ID
// @Tags Merchant Query
// @Security Bearer
// @Description Retrieve a merchant by its user ID
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} response.ApiResponsesMerchant "Merchant data"
// @Failure 400 {object} response.ErrorResponse "Invalid merchant ID"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve merchant data"
// @Router /api/merchant-query/merchant-user [get]
func (h *merchantQueryHandleApi) FindByMerchantUserId(w http.ResponseWriter, r *http.Request) error {
	userID, err := contextUserID(r)
	if err != nil {
		return errors.NewBadRequestError("user_id is required and must be valid")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedMerchantsByUserId(ctx, int(userID))
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindByMerchantUserIdRequest{
		UserId: int32(userID),
	}

	res, err := h.client.FindByMerchantUserId(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseMerchants(res)

	h.cache.SetCachedMerchantsByUserId(ctx, int(userID), apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

func contextUserID(r *http.Request) (int, error) {
	value := httpx.Get(r, "user_id")
	if value == nil {
		value = httpx.Get(r, "userId")
	}

	switch userID := value.(type) {
	case int:
		return userID, nil
	case int32:
		return int(userID), nil
	case int64:
		return int(userID), nil
	case float64:
		if math.IsNaN(userID) || math.IsInf(userID, 0) || userID < 0 || userID != math.Trunc(userID) || userID > float64(^uint(0)>>1) {
			return 0, fmt.Errorf("invalid numeric user id")
		}
		return int(userID), nil
	case string:
		parsed, err := strconv.Atoi(userID)
		if err != nil {
			return 0, fmt.Errorf("invalid user id: %w", err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported user id type %T", value)
	}
}

// FindByActive godoc
// @Summary Find active merchants
// @Tags Merchant Query
// @Security Bearer
// @Description Retrieve a list of active merchants
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Number of items per page" default(10)
// @Param search query string false "Search query"
// @Success 200 {object} response.ApiResponsesMerchant "List of active merchants"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve merchant data"
// @Router /api/merchant-query/active [get]
func (h *merchantQueryHandleApi) FindByActive(w http.ResponseWriter, r *http.Request) error {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page <= 0 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("page_size"))
	if err != nil || pageSize <= 0 {
		pageSize = 10
	}

	search := r.URL.Query().Get("search")
	ctx := r.Context()

	reqCache := &requests.FindAllMerchants{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCachedMerchantActive(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindAllMerchantRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByActive(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsesMerchantDeleteAt(res)

	h.cache.SetCachedMerchantActive(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindByTrashed godoc
// @Summary Find trashed merchants
// @Tags Merchant Query
// @Security Bearer
// @Description Retrieve a list of trashed merchants
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Number of items per page" default(10)
// @Param search query string false "Search query"
// @Success 200 {object} response.ApiResponsesMerchant "List of trashed merchants"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve merchant data"
// @Router /api/merchant-query/trashed [get]
func (h *merchantQueryHandleApi) FindByTrashed(w http.ResponseWriter, r *http.Request) error {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page <= 0 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("page_size"))
	if err != nil || pageSize <= 0 {
		pageSize = 10
	}

	search := r.URL.Query().Get("search")
	ctx := r.Context()

	reqCache := &requests.FindAllMerchants{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCachedMerchantTrashed(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindAllMerchantRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByTrashed(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsesMerchantDeleteAt(res)

	h.cache.SetCachedMerchantTrashed(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
