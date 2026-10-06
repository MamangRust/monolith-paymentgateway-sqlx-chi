package cardhandler

import (
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/card"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/card"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	card_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/card"
)

// cardQueryHandleApi handles card query HTTP APIs.
type cardQueryHandleApi struct {
	card pb.CardQueryServiceClient

	logger logger.LoggerInterface
	mapper apimapper.CardQueryResponseMapper

	cache card_cache.CardMencache

	apiHandler apierror.ApiHandler
}

// cardQueryHandleApiDeps defines dependencies for cardQueryHandleApi.
type cardQueryHandleApiDeps struct {
	client pb.CardQueryServiceClient
	router chi.Router

	logger logger.LoggerInterface
	mapper apimapper.CardQueryResponseMapper

	cache card_cache.CardMencache

	apiHandler apierror.ApiHandler
}

func NewCardQueryHandleApi(
	params *cardQueryHandleApiDeps,
) *cardQueryHandleApi {

	cardHandler := &cardQueryHandleApi{
		card:       params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/card-query", func(routerCard chi.Router) {

		routerCard.Get("/", params.apiHandler.Handle("find-all", cardHandler.FindAll))
		routerCard.Get("/{id}", params.apiHandler.Handle("find-by-id", cardHandler.FindById))
		routerCard.Get("/user", params.apiHandler.Handle("find-by-user-id", cardHandler.FindByUserID))
		routerCard.Get("/active", params.apiHandler.Handle("find-by-active", cardHandler.FindByActive))
		routerCard.Get("/trashed", params.apiHandler.Handle("find-by-trashed", cardHandler.FindByTrashed))
		routerCard.Get("/card_number/{card_number}", params.apiHandler.Handle("find-by-card-number", cardHandler.FindByCardNumber))

	})
	return cardHandler
}

// FindAll godoc
// @Summary Retrieve all cards
// @Tags Card Query
// @Security Bearer
// @Description Retrieve all cards with pagination
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param page_size query int false "Number of data per page"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationCard "Card data"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve card data"
// @Router /api/card-query [get]
func (h *cardQueryHandleApi) FindAll(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.FindAllCards{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetFindAllCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindAllCardRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	cards, err := h.card.FindAllCard(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsesCard(cards)
	h.cache.SetFindAllCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindById godoc
// @Summary Retrieve card by ID
// @Tags Card Query
// @Security Bearer
// @Description Retrieve a card by its ID
// @Accept json
// @Produce json
// @Param id path int true "Card ID"
// @Success 200 {object} response.ApiResponseCard "Card data"
// @Failure 400 {object} response.ErrorResponse "Invalid card ID"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve card record"
// @Router /api/card-query/{id} [get]
func (h *cardQueryHandleApi) FindById(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("id is required and must be an integer")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetByIdCache(ctx, id)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindByIdCardRequest{
		CardId: int32(id),
	}

	card, err := h.card.FindByIdCard(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseCard(card)
	h.cache.SetByIdCache(ctx, id, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindByUserID godoc
// @Summary Retrieve cards by user ID
// @Tags Card Query
// @Security Bearer
// @Description Retrieve a list of cards associated with a user by their ID
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseCard "Card data"
// @Failure 400 {object} response.ErrorResponse "Invalid user ID"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve card record"
// @Router /api/card-query/user [get]
func (h *cardQueryHandleApi) FindByUserID(w http.ResponseWriter, r *http.Request) error {
	userIDStr, ok := httpx.Get(r, "userId").(string)
	if !ok {
		return errors.NewBadRequestError("user_id is required")
	}

	uid, err := strconv.ParseInt(userIDStr, 10, 32)
	if err != nil {
		return errors.NewBadRequestError("invalid user ID format")
	}
	userID := int(uid)

	ctx := r.Context()

	cachedData, found := h.cache.GetByUserIDCache(ctx, userID)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindByUserIdCardRequest{
		UserId: int32(userID),
	}

	card, err := h.card.FindByUserIdCard(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseCard(card)
	h.cache.SetByUserIDCache(ctx, userID, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// @Security Bearer
// @Summary Retrieve active card by Saldo ID
// @Tags Card Query
// @Description Retrieve an active card associated with a Saldo ID
// @Accept json
// @Produce json
// @Success 200 {object} pb.ApiResponsePaginationCardDeleteAt "Card data"
// @Failure 400 {object} response.ErrorResponse "Invalid Saldo ID"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve card record"
// @Router /api/card-query/active [get]
func (h *cardQueryHandleApi) FindByActive(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.FindAllCards{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetByActiveCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindAllCardRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.card.FindByActiveCard(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsesCardDeletedAt(res)
	h.cache.SetByActiveCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// @Summary Retrieve trashed cards
// @Tags Card Query
// @Security Bearer
// @Description Retrieve a list of trashed cards
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Number of items per page (default: 10)"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationCardDeleteAt "Card data"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve card record"
// @Router /api/card-query/trashed [get]
func (h *cardQueryHandleApi) FindByTrashed(w http.ResponseWriter, r *http.Request) error {
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

	reqCache := &requests.FindAllCards{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetByTrashedCache(ctx, reqCache)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	reqGrpc := &pb.FindAllCardRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.card.FindByTrashedCard(ctx, reqGrpc)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsesCardDeletedAt(res)
	h.cache.SetByTrashedCache(ctx, reqCache, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// @Security Bearer
// @Summary Retrieve card by card number
// @Tags Card Query
// @Description Retrieve a card by its card number
// @Accept json
// @Produce json
// @Param card_number path string true "Card number"
// @Success 200 {object} response.ApiResponseCard "Card data"
// @Failure 400 {object} response.ErrorResponse "Failed to fetch card record"
// @Failure 500 {object} response.ErrorResponse "Failed to retrieve card record"
// @Router /api/card-query/{card_number} [get]
func (h *cardQueryHandleApi) FindByCardNumber(w http.ResponseWriter, r *http.Request) error {
	cardNumber := chi.URLParam(r, "card_number")

	ctx := r.Context()

	req := &pb.FindByCardNumberRequest{
		CardNumber: cardNumber,
	}

	res, err := h.card.FindByCardNumber(ctx, req)

	if err != nil {
		h.logger.Debug("Failed to fetch card record", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseCard(res)

	return httpx.JSON(w, http.StatusOK, so)
}
