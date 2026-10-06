package merchanthandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"

	pb "github.com/MamangRust/monolith-payment-gateway-pb/merchant"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/merchant"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	merchant_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/merchant"
)

type merchantTransactionHandleApi struct {
	client pb.MerchantTransactionServiceClient

	logger logger.LoggerInterface

	mapper apimapper.MerchantTransactionResponseMapper

	cache merchant_cache.MerchantMencache

	apiHandler apierror.ApiHandler
}

type merchantTransactionHandleDeps struct {
	client pb.MerchantTransactionServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.MerchantTransactionResponseMapper

	cache merchant_cache.MerchantMencache

	apiHandler apierror.ApiHandler
}

func NewMerchantTransactionHandleApi(params *merchantTransactionHandleDeps) *merchantTransactionHandleApi {

	merchantHandler := &merchantTransactionHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/merchant-transactions", func(routerMerchant chi.Router) {

		routerMerchant.Get("/transactions", params.apiHandler.Handle("find-all-transactions", merchantHandler.FindAllTransactions))
		routerMerchant.Get("/transactions/{merchant_id}", params.apiHandler.Handle("find-all-transactions-by-merchant", merchantHandler.FindAllTransactionByMerchant))
		routerMerchant.Get("/transactions/api-key/{api_key}", params.apiHandler.Handle("find-all-transactions-by-apikey", merchantHandler.FindAllTransactionByApikey))

	})
	return merchantHandler
}

// FindAllTransactions godoc
// @Summary Find all transactions
// @Tags Merchant Transaction
// @Security Bearer
// @Description Retrieve a list of all transactions
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Number of items per page" default(10)
// @Param search query string false "Search query"
// @Success 200 {object} response.ApiResponsePaginationTransaction "List of transactions"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve transaction data"
// @Router /api/merchant-transactions [get]
func (h *merchantTransactionHandleApi) FindAllTransactions(w http.ResponseWriter, r *http.Request) error {
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

	cacheReq := &requests.FindAllMerchantTransactions{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCacheAllMerchantTransactions(ctx, cacheReq)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	req := &pb.FindAllMerchantRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindAllTransactionMerchant(ctx, req)

	if err != nil {
		h.logger.Debug("Failed to retrieve transaction data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseMerchantsTransactionResponse(res)

	h.cache.SetCacheAllMerchantTransactions(ctx, cacheReq, so)

	return httpx.JSON(w, http.StatusOK, so)
}

// FindAllTransactionByMerchant godoc
// @Summary Find all transactions by merchant ID
// @Tags Merchant Transaction
// @Security Bearer
// @Description Retrieve a list of transactions for a specific merchant
// @Accept json
// @Produce json
// @Param merchant_id path int true "Merchant ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Number of items per page" default(10)
// @Param search query string false "Search query"
// @Success 200 {object} response.ApiResponsePaginationTransaction "List of transactions"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve transaction data"
// @Router /api/merchant-transactionss/:merchant_id [get]
func (h *merchantTransactionHandleApi) FindAllTransactionByMerchant(w http.ResponseWriter, r *http.Request) error {
	merchantID, err := strconv.Atoi(chi.URLParam(r, "merchant_id"))
	if err != nil || merchantID <= 0 {
		return errors.NewBadRequestError("merchant_id is required and must be a positive integer")
	}

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

	cacheReq := &requests.FindAllMerchantTransactionsById{
		MerchantID: merchantID,
		Page:       page,
		PageSize:   pageSize,
		Search:     search,
	}

	cachedData, found := h.cache.GetCacheMerchantTransactions(ctx, cacheReq)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	req := &pb.FindAllMerchantTransaction{
		MerchantId: int32(merchantID),
		Page:       int32(page),
		PageSize:   int32(pageSize),
		Search:     search,
	}

	res, err := h.client.FindAllTransactionByMerchant(ctx, req)

	if err != nil {
		h.logger.Debug("Failed to retrieve transaction data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseMerchantsTransactionResponse(res)

	h.cache.SetCacheMerchantTransactions(ctx, cacheReq, so)

	return httpx.JSON(w, http.StatusOK, so)
}

// FindAllTransactionByApikey godoc
// @Summary Find all transactions by api_key
// @Tags Merchant Transaction
// @Security Bearer
// @Description Retrieve a list of transactions for a specific merchant
// @Accept json
// @Produce json
// @Param api_key path string true "Api key"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Number of items per page" default(10)
// @Param search query string false "Search query"
// @Success 200 {object} response.ApiResponsePaginationTransaction "List of transactions"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve transaction data"
// @Router /api/merchant-transactionss/api-key/:api_key [get]
func (h *merchantTransactionHandleApi) FindAllTransactionByApikey(w http.ResponseWriter, r *http.Request) error {
	api_key := chi.URLParam(r, "api_key")

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

	cacheReq := &requests.FindAllMerchantTransactionsByApiKey{
		ApiKey:   api_key,
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCacheMerchantTransactionApikey(ctx, cacheReq)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	req := &pb.FindAllMerchantApikey{
		ApiKey:   api_key,
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindAllTransactionByApikey(ctx, req)

	if err != nil {
		h.logger.Debug("Failed to retrieve transaction data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseMerchantsTransactionResponse(res)

	h.cache.SetCacheMerchantTransactionApikey(ctx, cacheReq, so)

	return httpx.JSON(w, http.StatusOK, so)
}
