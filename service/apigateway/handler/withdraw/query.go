package withdrawhandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pbhelpers "github.com/MamangRust/monolith-payment-gateway-pb/card"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/withdraw"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/withdraw"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	withdraw_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/withdraw"
)

type withdrawQueryHandleApi struct {
	client pb.WithdrawQueryServiceClient

	logger logger.LoggerInterface

	mapper apimapper.WithdrawQueryResponseMapper

	cache withdraw_cache.WithdrawMencache

	apiHandler apierror.ApiHandler
}

type withdrawQueryHandleDeps struct {
	client pb.WithdrawQueryServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.WithdrawQueryResponseMapper

	cache withdraw_cache.WithdrawMencache

	apiHandler apierror.ApiHandler
}

func NewWithdrawQueryHandleApi(params *withdrawQueryHandleDeps) *withdrawQueryHandleApi {

	withdrawQueryHandleApi := &withdrawQueryHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/withdraw-query", func(routerWithdraw chi.Router) {

		routerWithdraw.Get("/", params.apiHandler.Handle("find-all-withdraws", withdrawQueryHandleApi.FindAll))
		routerWithdraw.Get("/card-number/{card_number}", params.apiHandler.Handle("find-all-withdraws-by-card-number", withdrawQueryHandleApi.FindAllByCardNumber))
		routerWithdraw.Get("/{id}", params.apiHandler.Handle("find-withdraw-by-id", withdrawQueryHandleApi.FindById))
		routerWithdraw.Get("/active", params.apiHandler.Handle("find-active-withdraws", withdrawQueryHandleApi.FindByActive))
		routerWithdraw.Get("/trashed", params.apiHandler.Handle("find-trashed-withdraws", withdrawQueryHandleApi.FindByTrashed))

	})
	return withdrawQueryHandleApi
}

// @Summary Find all withdraw records
// @Tags Withdraw Query
// @Security Bearer
// @Description Retrieve a list of all withdraw records with pagination and search
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Param search query string false "Search query"
// @Success 200 {object} response.ApiResponsePaginationWithdraw "List of withdraw records"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve withdraw data"
// @Router /api/withdraw-query [get]
func (h *withdrawQueryHandleApi) FindAll(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.FindAllWithdraws{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCachedWithdrawsCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindAllWithdrawRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindAllWithdraw(ctx, reqGrpc)
	if err != nil {
		h.logger.Debug("Failed to retrieve withdraw data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsePaginationWithdraw(res)
	h.cache.SetCachedWithdrawsCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// @Summary Find all withdraw records by card number
// @Tags Withdraw Query
// @Security Bearer
// @Description Retrieve a list of withdraw records for a specific card number with pagination and search
// @Accept json
// @Produce json
// @Param card_number path string true "Card Number"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Param search query string false "Search query"
// @Success 200 {object} response.ApiResponsePaginationWithdraw "List of withdraw records"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve withdraw data"
// @Router /api/withdraw-query/card-number/{card_number} [get]
func (h *withdrawQueryHandleApi) FindAllByCardNumber(w http.ResponseWriter, r *http.Request) error {
	cardNumber := chi.URLParam(r, "card_number")
	if cardNumber == "" {
		return errors.NewBadRequestError("card_number is required")
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

	reqCache := &requests.FindAllWithdrawCardNumber{
		CardNumber: cardNumber,
		Page:       page,
		PageSize:   pageSize,
		Search:     search,
	}

	cachedData, found := h.cache.GetCachedWithdrawByCardCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindAllWithdrawByCardNumberRequest{
		CardNumber: cardNumber,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		Search:     search,
	}

	res, err := h.client.FindAllWithdrawByCardNumber(ctx, reqGrpc)
	if err != nil {
		h.logger.Debug("Failed to retrieve withdraw data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsePaginationWithdraw(res)
	h.cache.SetCachedWithdrawByCardCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// @Summary Find a withdraw by ID
// @Tags Withdraw Query
// @Security Bearer
// @Description Retrieve a withdraw record using its ID
// @Accept json
// @Produce json
// @Param id path int true "Withdraw ID"
// @Success 200 {object} response.ApiResponseWithdraw "Withdraw data"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve withdraw data"
// @Router /api/withdraw-query/{id} [get]
func (h *withdrawQueryHandleApi) FindById(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedWithdrawCache(ctx, id)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindByIdWithdrawRequest{
		WithdrawId: int32(id),
	}

	withdraw, err := h.client.FindByIdWithdraw(ctx, reqGrpc)
	if err != nil {
		h.logger.Debug("Failed to retrieve withdraw data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseWithdraw(withdraw)
	h.cache.SetCachedWithdrawCache(ctx, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// @Summary Find a withdraw by card number
// @Tags Withdraw Query
// @Security Bearer
// @Description Retrieve a withdraw record using its card number
// @Accept json
// @Produce json
// @Param card_number query string true "Card number"
// @Success 200 {object} response.ApiResponsesWithdraw "Withdraw data"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid card number"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve withdraw data"
// @Router /api/withdraw-query/card/{card_number} [get]
func (h *withdrawQueryHandleApi) FindByCardNumber(w http.ResponseWriter, r *http.Request) error {
	cardNumber := r.URL.Query().Get("card_number")

	ctx := r.Context()

	req := &pbhelpers.FindByCardNumberRequest{
		CardNumber: cardNumber,
	}

	withdraw, err := h.client.FindByCardNumber(ctx, req)
	if err != nil {
		h.logger.Debug("Failed to retrieve withdraw data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponsesWithdraw(withdraw)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Retrieve all active withdraw data
// @Tags Withdraw Query
// @Security Bearer
// @Description Retrieve a list of all active withdraw data
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponsesWithdraw "List of withdraw data"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve withdraw data"
// @Router /api/withdraw-query/active [get]
func (h *withdrawQueryHandleApi) FindByActive(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.FindAllWithdraws{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCachedWithdrawActiveCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindAllWithdrawRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByActive(ctx, reqGrpc)
	if err != nil {
		h.logger.Debug("Failed to retrieve withdraw data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsePaginationWithdrawDeleteAt(res)
	h.cache.SetCachedWithdrawActiveCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// @Summary Retrieve trashed withdraw data
// @Tags Withdraw Query
// @Security Bearer
// @Description Retrieve a list of trashed withdraw data
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponsesWithdraw "List of trashed withdraw data"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve withdraw data"
// @Router /api/withdraw-query/trashed [get]
func (h *withdrawQueryHandleApi) FindByTrashed(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.FindAllWithdraws{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCachedWithdrawTrashedCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindAllWithdrawRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByTrashed(ctx, reqGrpc)
	if err != nil {
		h.logger.Debug("Failed to retrieve withdraw data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsePaginationWithdrawDeleteAt(res)
	h.cache.SetCachedWithdrawTrashedCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
