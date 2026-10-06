package rolehandler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/role"
	"github.com/MamangRust/monolith-payment-gateway-pkg/kafka"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/role"
	"github.com/go-chi/chi/v5"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	"github.com/MamangRust/monolith-payment-gateway-apigateway/middlewares"
	mencache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis"
	role_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/role"
)

type roleQueryHandlerApi struct {
	kafka *kafka.Kafka

	role pb.RoleServiceClient

	logger logger.LoggerInterface

	mapper apimapper.RoleQueryResponseMapper

	cache role_cache.RoleMencache

	apiHandler apierror.ApiHandler
}

type roleQueryHandleDeps struct {
	client     pb.RoleServiceClient
	router     chi.Router
	logger     logger.LoggerInterface
	mapper     apimapper.RoleQueryResponseMapper
	kafka      *kafka.Kafka
	cache_role mencache.RoleCache

	cache role_cache.RoleMencache

	apiHandler apierror.ApiHandler

	validator *middlewares.RoleValidator
}

func NewRoleQueryHandleApi(params *roleQueryHandleDeps) *roleQueryHandlerApi {

	roleQueryHandler := &roleQueryHandlerApi{
		role:       params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		kafka:      params.kafka,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	roleMiddleware := params.validator
	if roleMiddleware == nil {
		// Fallback: keep old behavior when the shared validator is not provided.
		roleMiddleware = middlewares.NewRoleValidator(params.kafka, "request-role", "response-role", 5*time.Second, params.logger, params.cache_role)
	}

	params.router.Route("/api/role-query", func(routerRole chi.Router) {

		roleMiddlewareChain := roleMiddleware.Middleware()
		requireAdmin := middlewares.RequireRoles("Admin_Role_10")

		routerRole.With(roleMiddlewareChain, requireAdmin).Get("/", httpx.Handler(roleQueryHandler.FindAll))

		routerRole.With(roleMiddlewareChain, requireAdmin).Get("/{id}", httpx.Handler(roleQueryHandler.FindById))

		routerRole.With(roleMiddlewareChain, requireAdmin).Get("/active", httpx.Handler(roleQueryHandler.FindAll))

		routerRole.With(roleMiddlewareChain, requireAdmin).Get("/trashed", httpx.Handler(roleQueryHandler.FindByTrashed))

		routerRole.With(roleMiddlewareChain, requireAdmin).Get("/user/{user_id}", httpx.Handler(roleQueryHandler.FindByUserId))

	})
	return roleQueryHandler
}

// FindAll godoc.
// @Summary Get all roles
// @Tags Role Command
// @Security Bearer
// @Description Retrieve a paginated list of roles with optional search and pagination parameters.
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Number of items per page (default: 10)"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationRole "List of roles"
// @Failure 400 {object} response.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} response.ErrorResponse "Failed to fetch roles"
// @Router /api/role-query [get]
func (h *roleQueryHandlerApi) FindAll(w http.ResponseWriter, r *http.Request) error {
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

	req := &requests.FindAllRoles{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCachedRoles(ctx, req)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	grpcReq := &pb.FindAllRoleRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.role.FindAllRole(ctx, grpcReq)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsePaginationRole(res)

	h.cache.SetCachedRoles(ctx, req, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindById godoc.
// @Summary Get a role by ID
// @Tags Role Command
// @Security Bearer
// @Description Retrieve a role by its ID.
// @Accept json
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {object} response.ApiResponseRole "Role data"
// @Failure 400 {object} response.ErrorResponse "Invalid role ID"
// @Failure 500 {object} response.ErrorResponse "Failed to fetch role"
// @Router /api/role-query/{id} [get]
func (h *roleQueryHandlerApi) FindById(w http.ResponseWriter, r *http.Request) error {
	roleID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || roleID <= 0 {
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedRoleById(ctx, roleID)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	req := &pb.FindByIdRoleRequest{
		RoleId: int32(roleID),
	}

	res, err := h.role.FindByIdRole(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponseRole(res)

	h.cache.SetCachedRoleById(ctx, roleID, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindByActive godoc.
// @Summary Get active roles
// @Tags Role Command
// @Security Bearer
// @Description Retrieve a paginated list of active roles with optional search and pagination parameters.
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Number of items per page (default: 10)"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationRoleDeleteAt "List of active roles"
// @Failure 400 {object} response.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} response.ErrorResponse "Failed to fetch active roles"
// @Router /api/role-query/active [get]
func (h *roleQueryHandlerApi) FindByActive(w http.ResponseWriter, r *http.Request) error {
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

	req := &requests.FindAllRoles{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCachedRoleActive(ctx, req)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	grpcReq := &pb.FindAllRoleRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.role.FindByActive(ctx, grpcReq)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsePaginationRoleDeleteAt(res)

	h.cache.SetCachedRoleActive(ctx, req, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindByTrashed godoc.
// @Summary Get trashed roles
// @Tags Role Command
// @Security Bearer
// @Description Retrieve a paginated list of trashed roles with optional search and pagination parameters.
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Number of items per page (default: 10)"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationRoleDeleteAt "List of trashed roles"
// @Failure 400 {object} response.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} response.ErrorResponse "Failed to fetch trashed roles"
// @Router /api/role-query/trashed [get]
func (h *roleQueryHandlerApi) FindByTrashed(w http.ResponseWriter, r *http.Request) error {
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

	req := &requests.FindAllRoles{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
	}

	cachedData, found := h.cache.GetCachedRoleTrashed(ctx, req)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	grpcReq := &pb.FindAllRoleRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.role.FindByTrashed(ctx, grpcReq)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsePaginationRoleDeleteAt(res)

	h.cache.SetCachedRoleTrashed(ctx, req, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}

// FindByUserId godoc.
// @Summary Get role by user ID
// @Tags Role Command
// @Security Bearer
// @Description Retrieve a role by the associated user ID.
// @Accept json
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {object} response.ApiResponseRole "Role data"
// @Failure 400 {object} response.ErrorResponse "Invalid user ID"
// @Failure 500 {object} response.ErrorResponse "Failed to fetch role by user ID"
// @Router /api/role-query/user/{user_id} [get]
func (h *roleQueryHandlerApi) FindByUserId(w http.ResponseWriter, r *http.Request) error {
	userID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil || userID <= 0 {
		return errors.NewBadRequestError("user_id is required")
	}

	ctx := r.Context()

	cachedData, found := h.cache.GetCachedRoleByUserId(ctx, userID)
	if found {
		return httpx.JSON(w, http.StatusOK, cachedData)
	}

	req := &pb.FindByIdUserRoleRequest{
		UserId: int32(userID),
	}

	res, err := h.role.FindByUserId(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	apiResponse := h.mapper.ToApiResponsesRole(res)

	h.cache.SetCachedRoleByUserId(ctx, userID, apiResponse)

	return httpx.JSON(w, http.StatusOK, apiResponse)
}
