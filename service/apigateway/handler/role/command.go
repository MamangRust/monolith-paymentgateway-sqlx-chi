package rolehandler

import (
	"fmt"
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

	"github.com/MamangRust/monolith-payment-gateway-apigateway/middlewares"
	mencache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis"
	role_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/role"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
)

type roleCommandHandleApi struct {
	kafka *kafka.Kafka

	role pb.RoleCommandServiceClient

	logger logger.LoggerInterface

	mapper apimapper.RoleCommandResponseMapper

	cache role_cache.RoleMencache

	apiHandler apierror.ApiHandler
}

type roleCommandHandleDeps struct {
	client pb.RoleCommandServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.RoleCommandResponseMapper

	kafka *kafka.Kafka

	cache_role mencache.RoleCache

	cache role_cache.RoleMencache

	apiHandler apierror.ApiHandler

	validator *middlewares.RoleValidator
}

func NewRoleCommandHandleApi(params *roleCommandHandleDeps) *roleCommandHandleApi {
	roleHandler := &roleCommandHandleApi{
		role:       params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
		kafka:      params.kafka,
	}

	roleMiddleware := params.validator
	if roleMiddleware == nil {
		// Fallback: keep old behavior when the shared validator is not provided.
		roleMiddleware = middlewares.NewRoleValidator(params.kafka, "request-role", "response-role", 5*time.Second, params.logger, params.cache_role)
	}

	params.router.Route("/api/role", func(routerRole chi.Router) {

		roleMiddlewareChain := roleMiddleware.Middleware()
		requireAdmin := middlewares.RequireRoles("Admin_Admin_14")

		routerRole.With(roleMiddlewareChain, requireAdmin).Post("/", params.apiHandler.Handle("create-role", roleHandler.Create))

		routerRole.With(roleMiddlewareChain, requireAdmin).Post("/{id}", params.apiHandler.Handle("update-role", roleHandler.Update))

		routerRole.With(roleMiddlewareChain, requireAdmin).Post("/trashed/{id}", params.apiHandler.Handle("trash-role", roleHandler.Trashed))

		routerRole.With(roleMiddlewareChain, requireAdmin).Put("/restore/{id}", params.apiHandler.Handle("restore-role", roleHandler.Restore))

		routerRole.With(roleMiddlewareChain, requireAdmin).Delete("/{id}", params.apiHandler.Handle("delete-role", roleHandler.DeletePermanent))

		routerRole.With(roleMiddlewareChain, requireAdmin).Delete("/permanent/{id}", params.apiHandler.Handle("delete-role-permanent", roleHandler.DeletePermanent))

		routerRole.With(roleMiddlewareChain, requireAdmin).Post("/restore/all", params.apiHandler.Handle("restore-all-roles", roleHandler.RestoreAll))

		routerRole.With(roleMiddlewareChain, requireAdmin).Post("/permanent/all", params.apiHandler.Handle("delete-all-roles-permanent", roleHandler.DeleteAllPermanent))

	})
	return roleHandler
}

// Create godoc.
// @Summary Create a new role
// @Tags Role Command
// @Security Bearer
// @Description Create a new role with the provided details.
// @Accept json
// @Produce json
// @Param request body requests.CreateRoleRequest true "Role data"
// @Success 200 {object} response.ApiResponseRole "Created role data"
// @Failure 400 {object} response.ErrorResponse "Invalid request body"
// @Failure 500 {object} response.ErrorResponse "Failed to create role"
// @Router /api/role-command/create [post]
func (h *roleCommandHandleApi) Create(w http.ResponseWriter, r *http.Request) error {
	var body requests.CreateRoleRequest

	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("Invalid request format").WithInternal(err)
	}

	if err := body.Validate(); err != nil {
		validations := h.parseValidationErrors(err)
		return errors.NewValidationError(validations)
	}

	reqPb := &pb.CreateRoleRequest{
		Name: body.Name,
	}

	ctx := r.Context()

	res, err := h.role.CreateRole(ctx, reqPb)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseRole(res)

	return httpx.JSON(w, http.StatusOK, so)
}

// Update godoc.
// @Summary Update a role
// @Tags Role Command
// @Security Bearer
// @Description Update an existing role with the provided details.
// @Accept json
// @Produce json
// @Param id path int true "Role ID"
// @Param request body requests.UpdateRoleRequest true "Role data"
// @Success 200 {object} response.ApiResponseRole "Updated role data"
// @Failure 400 {object} response.ErrorResponse "Invalid role ID or request body"
// @Failure 500 {object} response.ErrorResponse "Failed to update role"
// @Router /api/role-command/update/{id} [post]
func (h *roleCommandHandleApi) Update(w http.ResponseWriter, r *http.Request) error {
	roleID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || roleID <= 0 {
		return errors.NewBadRequestError("id is required")
	}

	var body requests.UpdateRoleRequest

	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("Invalid request format").WithInternal(err)
	}

	if err := body.Validate(); err != nil {
		validations := h.parseValidationErrors(err)
		return errors.NewValidationError(validations)
	}

	reqPb := &pb.UpdateRoleRequest{
		Id:   int32(roleID),
		Name: body.Name,
	}

	ctx := r.Context()

	res, err := h.role.UpdateRole(ctx, reqPb)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseRole(res)

	return httpx.JSON(w, http.StatusOK, so)
}

// Trashed godoc.
// @Summary Soft-delete a role
// @Tags Role Command
// @Security Bearer
// @Description Soft-delete a role by its ID.
// @Accept json
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {object} response.ApiResponseRole "Soft-deleted role data"
// @Failure 400 {object} response.ErrorResponse "Invalid role ID"
// @Failure 500 {object} response.ErrorResponse "Failed to soft-delete role"
// @Router /api/role-command/trashed/{id} [post]
func (h *roleCommandHandleApi) Trashed(w http.ResponseWriter, r *http.Request) error {
	roleID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || roleID <= 0 {
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	req := &pb.FindByIdRoleRequest{
		RoleId: int32(roleID),
	}

	res, err := h.role.TrashedRole(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseRoleDeleteAt(res)

	return httpx.JSON(w, http.StatusOK, so)
}

// Restore godoc.
// @Summary Restore a soft-deleted role
// @Tags Role Command
// @Security Bearer
// @Description Restore a soft-deleted role by its ID.
// @Accept json
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {object} response.ApiResponseRole "Restored role data"
// @Failure 400 {object} response.ErrorResponse "Invalid role ID"
// @Failure 500 {object} response.ErrorResponse "Failed to restore role"
// @Router /api/role-command/restore/{id} [post]
func (h *roleCommandHandleApi) Restore(w http.ResponseWriter, r *http.Request) error {
	roleID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || roleID <= 0 {
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	req := &pb.FindByIdRoleRequest{
		RoleId: int32(roleID),
	}

	res, err := h.role.RestoreRole(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseRoleDeleteAt(res)

	return httpx.JSON(w, http.StatusOK, so)
}

// DeletePermanent godoc.
// @Summary Permanently delete a role
// @Tags Role Command
// @Security Bearer
// @Description Permanently delete a role by its ID.
// @Accept json
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {object} response.ApiResponseRole "Permanently deleted role data"
// @Failure 400 {object} response.ErrorResponse "Invalid role ID"
// @Failure 500 {object} response.ErrorResponse "Failed to delete role permanently"
// @Router /api/role-command/permanent/{id} [delete]
func (h *roleCommandHandleApi) DeletePermanent(w http.ResponseWriter, r *http.Request) error {
	roleID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || roleID <= 0 {
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	req := &pb.FindByIdRoleRequest{
		RoleId: int32(roleID),
	}

	res, err := h.role.DeleteRolePermanent(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseRoleDelete(res)

	return httpx.JSON(w, http.StatusOK, so)
}

// RestoreAll godoc.
// @Summary Restore all soft-deleted roles
// @Tags Role Command
// @Security Bearer
// @Description Restore all soft-deleted roles.
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseRoleAll "Restored roles data"
// @Failure 500 {object} response.ErrorResponse "Failed to restore all roles"
// @Router /api/role-command/restore/all [post]
func (h *roleCommandHandleApi) RestoreAll(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	res, err := h.role.RestoreAllRole(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseRoleAll(res)

	return httpx.JSON(w, http.StatusOK, so)
}

// DeleteAllPermanent godoc.
// @Summary Permanently delete all roles
// @Tags Role Command
// @Security Bearer
// @Description Permanently delete all roles.
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseRoleAll "Permanently deleted roles data"
// @Failure 500 {object} response.ErrorResponse "Failed to delete all roles permanently"
// @Router /api/role-command/permanent/all [delete]
func (h *roleCommandHandleApi) DeleteAllPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	res, err := h.role.DeleteAllRolePermanent(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseRoleAll(res)

	return httpx.JSON(w, http.StatusOK, so)
}

func (h *roleCommandHandleApi) parseValidationErrors(err error) []errors.ValidationError {
	var validationErrs []errors.ValidationError

	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			validationErrs = append(validationErrs, errors.ValidationError{
				Field:   fe.Field(),
				Message: h.getValidationMessage(fe),
			})
		}
		return validationErrs
	}

	return []errors.ValidationError{
		{
			Field:   "general",
			Message: err.Error(),
		},
	}
}

func (h *roleCommandHandleApi) getValidationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return fmt.Sprintf("Must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("Must be at most %s", fe.Param())
	case "gte":
		return fmt.Sprintf("Must be greater than or equal to %s", fe.Param())
	case "lte":
		return fmt.Sprintf("Must be less than or equal to %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("Must be one of: %s", fe.Param())
	default:
		return fmt.Sprintf("Validation failed on '%s' tag", fe.Tag())
	}
}
