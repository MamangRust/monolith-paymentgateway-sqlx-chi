package transferhandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"

	pb "github.com/MamangRust/monolith-payment-gateway-pb/transfer"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/transfer"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	transfer_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/transfer"
)

type transferHandleApi struct {
	client pb.TransferQueryServiceClient

	logger logger.LoggerInterface

	mapper apimapper.TransferQueryResponseMapper

	cache transfer_cache.TransferMencache

	apiHandler apierror.ApiHandler
}

type transferQueryHandleDeps struct {
	client pb.TransferQueryServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.TransferQueryResponseMapper

	cache transfer_cache.TransferMencache

	apiHandler apierror.ApiHandler
}

func NewTransferQueryHandleApi(params *transferQueryHandleDeps) *transferHandleApi {

	transferHandleApi := &transferHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/transfer-query", func(routerTransfer chi.Router) {

		routerTransfer.Get("/", params.apiHandler.Handle("find-all-transfers", transferHandleApi.FindAll))
		routerTransfer.Get("/{id}", params.apiHandler.Handle("find-transfer-by-id", transferHandleApi.FindById))
		routerTransfer.Get("/transfer_from/{transfer_from}", params.apiHandler.Handle("find-transfers-by-transfer-from", transferHandleApi.FindByTransferByTransferFrom))
		routerTransfer.Get("/transfer_to/{transfer_to}", params.apiHandler.Handle("find-transfers-by-transfer-to", transferHandleApi.FindByTransferByTransferTo))

		routerTransfer.Get("/active", params.apiHandler.Handle("find-active-transfers", transferHandleApi.FindByActiveTransfer))
		routerTransfer.Get("/trashed", params.apiHandler.Handle("find-trashed-transfers", transferHandleApi.FindByTrashedTransfer))

	})
	return transferHandleApi
}

// @Summary Find all transfer records
// @Tags Transfer Query
// @Security Bearer
// @Description Retrieve a list of all transfer records with pagination
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Number of items per page" default(10)
// @Param search query string false "Search query"
// @Success 200 {object} response.ApiResponsePaginationTransfer "List of transfer records"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve transfer data"
// @Router /api/transfer-query [get]
func (h *transferHandleApi) FindAll(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.FindAllTransfers{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCachedTransfersCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindAllTransferRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindAllTransfer(ctx, reqGrpc)
	if err != nil {
		h.logger.Debug("Failed to retrieve transfer data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsePaginationTransfer(res)
	h.cache.SetCachedTransfersCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// @Summary Find a transfer by ID
// @Tags Transfer Query
// @Security Bearer
// @Description Retrieve a transfer record using its ID
// @Accept json
// @Produce json
// @Param id path string true "Transfer ID"
// @Success 200 {object} response.ApiResponseTransfer "Transfer data"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve transfer data"
// @Router /api/transfer-query/{id} [get]
func (h *transferHandleApi) FindById(w http.ResponseWriter, r *http.Request) error {
	idStr := chi.URLParam(r, "id")
	idInt, err := strconv.Atoi(idStr)
	if err != nil {
		h.logger.Debug("Bad Request: Invalid ID", zap.Error(err))
		return errors.NewBadRequestError("invalid id parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedTransferCache(ctx, idInt)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindByIdTransfer(ctx, &pb.FindByIdTransferRequest{
		TransferId: int32(idInt),
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve transfer data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransfer(res)
	h.cache.SetCachedTransferCache(ctx, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// @Summary Find transfers by transfer_from
// @Tags Transfer Query
// @Security Bearer
// @Description Retrieve a list of transfer records using the transfer_from parameter
// @Accept json
// @Produce json
// @Param transfer_from path string true "Transfer From"
// @Success 200 {object} response.ApiResponseTransfers "Transfer data"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve transfer data"
// @Router /api/transfer-query/transfer_from/{transfer_from} [get]
func (h *transferHandleApi) FindByTransferByTransferFrom(w http.ResponseWriter, r *http.Request) error {
	transferFrom := chi.URLParam(r, "transfer_from")
	if transferFrom == "" {
		return errors.NewBadRequestError("invalid card_number parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedTransferByFrom(ctx, transferFrom)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindTransferByTransferFrom(ctx, &pb.FindTransferByTransferFromRequest{
		TransferFrom: transferFrom,
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve transfer data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransfers(res)
	h.cache.SetCachedTransferByFrom(ctx, transferFrom, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// @Summary Find transfers by transfer_to
// @Tags Transfer Query
// @Security Bearer
// @Description Retrieve a list of transfer records using the transfer_to parameter
// @Accept json
// @Produce json
// @Param transfer_to path string true "Transfer To"
// @Success 200 {object} response.ApiResponseTransfers "Transfer data"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve transfer data"
// @Router /api/transfer-query/transfer_to/{transfer_to} [get]
func (h *transferHandleApi) FindByTransferByTransferTo(w http.ResponseWriter, r *http.Request) error {
	transferTo := chi.URLParam(r, "transfer_to")
	if transferTo == "" {
		return errors.NewBadRequestError("invalid card_number parameter")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedTransferByTo(ctx, transferTo)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	res, err := h.client.FindTransferByTransferTo(ctx, &pb.FindTransferByTransferToRequest{
		TransferTo: transferTo,
	})
	if err != nil {
		h.logger.Debug("Failed to retrieve transfer data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseTransfers(res)
	h.cache.SetCachedTransferByTo(ctx, transferTo, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// @Summary Find active transfers
// @Tags Transfer Query
// @Security Bearer
// @Description Retrieve a list of active transfer records
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Number of items per page (default: 10)"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponseTransfers "Active transfer data"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve transfer data"
// @Router /api/transfer-query/active [get]
func (h *transferHandleApi) FindByActiveTransfer(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.FindAllTransfers{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCachedTransferActiveCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindAllTransferRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByActiveTransfer(ctx, reqGrpc)
	if err != nil {
		h.logger.Debug("Failed to retrieve transfer data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsePaginationTransferDeleteAt(res)
	h.cache.SetCachedTransferActiveCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// @Summary Retrieve trashed transfers
// @Tags Transfer Query
// @Security Bearer
// @Description Retrieve a list of trashed transfer records
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Number of items per page (default: 10)"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponseTransfers "List of trashed transfer records"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve transfer data"
// @Router /api/transfer-query/trashed [get]
func (h *transferHandleApi) FindByTrashedTransfer(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.FindAllTransfers{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCachedTransferTrashedCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindAllTransferRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByTrashedTransfer(ctx, reqGrpc)
	if err != nil {
		h.logger.Debug("Failed to retrieve transfer data", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsePaginationTransferDeleteAt(res)
	h.cache.SetCachedTransferTrashedCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
