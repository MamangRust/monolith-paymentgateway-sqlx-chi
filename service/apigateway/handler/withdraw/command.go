package withdrawhandler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/withdraw"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/withdraw"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	withdraw_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/withdraw"
)

type withdrawCommandHandleApi struct {
	client pb.WithdrawCommandServiceClient

	logger logger.LoggerInterface

	mapper apimapper.WithdrawCommandResponseMapper

	cache withdraw_cache.WithdrawMencache

	apiHandler apierror.ApiHandler
}

type withdrawCommandHandleDeps struct {
	client pb.WithdrawCommandServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.WithdrawCommandResponseMapper

	cache withdraw_cache.WithdrawMencache

	apiHandler apierror.ApiHandler
}

func NewWithdrawCommandHandleApi(params *withdrawCommandHandleDeps) *withdrawCommandHandleApi {

	withdrawCommandHandleApi := &withdrawCommandHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/withdraw-command", func(routerWithdraw chi.Router) {

		routerWithdraw.Post("/create", params.apiHandler.Handle("create-withdraw", withdrawCommandHandleApi.Create))
		routerWithdraw.Post("/update/{id}", params.apiHandler.Handle("update-withdraw", withdrawCommandHandleApi.Update))

		routerWithdraw.Post("/trashed/{id}", params.apiHandler.Handle("trash-withdraw", withdrawCommandHandleApi.TrashWithdraw))
		routerWithdraw.Post("/restore/{id}", params.apiHandler.Handle("restore-withdraw", withdrawCommandHandleApi.RestoreWithdraw))
		routerWithdraw.Delete("/permanent/{id}", params.apiHandler.Handle("delete-withdraw-permanent", withdrawCommandHandleApi.DeleteWithdrawPermanent))

		routerWithdraw.Post("/restore/all", params.apiHandler.Handle("restore-all-withdraws", withdrawCommandHandleApi.RestoreAllWithdraw))
		routerWithdraw.Post("/permanent/all", params.apiHandler.Handle("delete-all-withdraws-permanent", withdrawCommandHandleApi.DeleteAllWithdrawPermanent))

	})
	return withdrawCommandHandleApi
}

// @Summary Create a new withdraw
// @Tags Withdraw Command
// @Security Bearer
// @Description Create a new withdraw record with the provided details.
// @Accept json
// @Produce json
// @Param CreateWithdrawRequest body requests.CreateWithdrawRequest true "Create Withdraw Request"
// @Success 200 {object} response.ApiResponseWithdraw "Successfully created withdraw record"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid request body or validation error"
// @Failure 500 {object} response.ErrorResponse "Failed to create withdraw"
// @Router /api/withdraw-command/create [post]
func (h *withdrawCommandHandleApi) Create(w http.ResponseWriter, r *http.Request) error {
	var body requests.CreateWithdrawRequest

	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("Invalid request")
	}

	if err := body.Validate(); err != nil {
		validations := h.parseValidationErrors(err)
		return errors.NewValidationError(validations)
	}

	ctx := r.Context()

	res, err := h.client.CreateWithdraw(ctx, &pb.CreateWithdrawRequest{
		CardNumber:     body.CardNumber,
		WithdrawAmount: int32(body.WithdrawAmount),
		WithdrawTime:   timestamppb.New(body.WithdrawTime),
		IdempotencyKey: body.IdempotencyKey,
	})

	if err != nil {
		h.logger.Debug("Failed to create withdraw", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseWithdraw(res)
	h.cache.SetCachedWithdrawCache(ctx, so)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Update an existing withdraw
// @Tags Withdraw Command
// @Security Bearer
// @Description Update an existing withdraw record with the provided details.
// @Accept json
// @Produce json
// @Param id path int true "Withdraw ID"
// @Param UpdateWithdrawRequest body requests.UpdateWithdrawRequest true "Update Withdraw Request"
// @Success 200 {object} response.ApiResponseWithdraw "Successfully updated withdraw record"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid request body or validation error"
// @Failure 500 {object} response.ErrorResponse "Failed to update withdraw"
// @Router /api/withdraw-command/update/{id} [post]
func (h *withdrawCommandHandleApi) Update(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))

	if err != nil {
		return errors.NewBadRequestError("id is required")
	}

	var body requests.UpdateWithdrawRequest

	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("Invalid request")
	}

	if err := body.Validate(); err != nil {
		validations := h.parseValidationErrors(err)
		return errors.NewValidationError(validations)
	}

	ctx := r.Context()

	res, err := h.client.UpdateWithdraw(ctx, &pb.UpdateWithdrawRequest{
		WithdrawId:     int32(id),
		CardNumber:     body.CardNumber,
		WithdrawAmount: int32(body.WithdrawAmount),
		WithdrawTime:   timestamppb.New(body.WithdrawTime),
	})

	if err != nil {
		h.logger.Debug("Failed to update withdraw", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseWithdraw(res)

	h.cache.DeleteCachedWithdrawCache(ctx, id)
	h.cache.SetCachedWithdrawCache(ctx, so)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Trash a withdraw by ID
// @Tags Withdraw Command
// @Security Bearer
// @Description Trash a withdraw using its ID
// @Accept json
// @Produce json
// @Param id path int true "Withdraw ID"
// @Success 200 {object} response.ApiResponseWithdraw "Withdaw data"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to trash withdraw"
// @Router /api/withdraw-command/trashed/{id} [post]
func (h *withdrawCommandHandleApi) TrashWithdraw(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))

	if err != nil {
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	res, err := h.client.TrashedWithdraw(ctx, &pb.FindByIdWithdrawRequest{
		WithdrawId: int32(id),
	})

	if err != nil {
		h.logger.Debug("Failed to trash withdraw", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseWithdrawDeleteAt(res)

	h.cache.DeleteCachedWithdrawCache(ctx, id)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Restore a withdraw by ID
// @Tags Withdraw Command
// @Security Bearer
// @Description Restore a withdraw by its ID
// @Accept json
// @Produce json
// @Param id path int true "Withdraw ID"
// @Success 200 {object} response.ApiResponseWithdraw "Withdraw data"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to restore withdraw"
// @Router /api/withdraw-command/restore/{id} [post]
func (h *withdrawCommandHandleApi) RestoreWithdraw(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))

	if err != nil {
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	res, err := h.client.RestoreWithdraw(ctx, &pb.FindByIdWithdrawRequest{
		WithdrawId: int32(id),
	})

	if err != nil {
		h.logger.Debug("Failed to restore withdraw", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseWithdrawDeleteAt(res)

	h.cache.DeleteCachedWithdrawCache(ctx, id)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Permanently delete a withdraw by ID
// @Tags Withdraw Command
// @Security Bearer
// @Description Permanently delete a withdraw by its ID
// @Accept json
// @Produce json
// @Param id path int true "Withdraw ID"
// @Success 200 {object} response.ApiResponseWithdrawDelete "Successfully deleted withdraw permanently"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to delete withdraw permanently:"
// @Router /api/withdraw-command/permanent/{id} [delete]
func (h *withdrawCommandHandleApi) DeleteWithdrawPermanent(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))

	if err != nil {
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	res, err := h.client.DeleteWithdrawPermanent(ctx, &pb.FindByIdWithdrawRequest{
		WithdrawId: int32(id),
	})

	if err != nil {
		h.logger.Debug("Failed to delete withdraw permanent", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseWithdrawDelete(res)

	h.cache.DeleteCachedWithdrawCache(ctx, id)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Restore a withdraw all
// @Tags Withdraw Command
// @Security Bearer
// @Description Restore a withdraw all
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseWithdrawAll "Withdraw data"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to restore withdraw"
// @Router /api/withdraw-command/restore/all [post]
func (h *withdrawCommandHandleApi) RestoreAllWithdraw(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	res, err := h.client.RestoreAllWithdraw(ctx, &emptypb.Empty{})

	if err != nil {
		h.logger.Debug("Failed to restore all withdraw", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	h.logger.Debug("Successfully restored all withdraw")

	so := h.mapper.ToApiResponseWithdrawAll(res)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Permanently delete a withdraw by ID
// @Tags Withdraw Command
// @Security Bearer
// @Description Permanently delete a withdraw by its ID
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseWithdrawAll "Successfully deleted withdraw permanently"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to delete withdraw permanently:"
// @Router /api/withdraw-command/permanent/all [post]
func (h *withdrawCommandHandleApi) DeleteAllWithdrawPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	res, err := h.client.DeleteAllWithdrawPermanent(ctx, &emptypb.Empty{})

	if err != nil {
		h.logger.Debug("Failed to delete all withdraw permanent", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	h.logger.Debug("Successfully deleted all withdraw permanently")

	so := h.mapper.ToApiResponseWithdrawAll(res)

	return httpx.JSON(w, http.StatusOK, so)
}

func (h *withdrawCommandHandleApi) parseValidationErrors(err error) []errors.ValidationError {
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

func (h *withdrawCommandHandleApi) getValidationMessage(fe validator.FieldError) string {
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
