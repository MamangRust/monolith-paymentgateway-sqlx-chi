package transferhandler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/transfer"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/transfer"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	transfer_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/transfer"
)

type transferCommandHandleApi struct {
	client pb.TransferCommandServiceClient

	logger logger.LoggerInterface

	mapper apimapper.TransferCommandResponseMapper

	cache transfer_cache.TransferMencache

	apiHandler apierror.ApiHandler
}

type transferCommandHandleDeps struct {
	client pb.TransferCommandServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.TransferCommandResponseMapper

	cache transfer_cache.TransferMencache

	apiHandler apierror.ApiHandler
}

func NewTransferCommandHandleApi(params *transferCommandHandleDeps) *transferCommandHandleApi {

	transferCommandHandleApi := &transferCommandHandleApi{
		client:     params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/transfer-command", func(routerTransfer chi.Router) {

		routerTransfer.Post("/create", params.apiHandler.Handle("create-transfer", transferCommandHandleApi.CreateTransfer))
		routerTransfer.Post("/update/{id}", params.apiHandler.Handle("update-transfer", transferCommandHandleApi.UpdateTransfer))
		routerTransfer.Post("/trashed/{id}", params.apiHandler.Handle("trash-transfer", transferCommandHandleApi.TrashTransfer))
		routerTransfer.Post("/restore/{id}", params.apiHandler.Handle("restore-transfer", transferCommandHandleApi.RestoreTransfer))
		routerTransfer.Delete("/permanent/{id}", params.apiHandler.Handle("delete-transfer-permanent", transferCommandHandleApi.DeleteTransferPermanent))

		routerTransfer.Post("/restore/all", params.apiHandler.Handle("restore-all-transfers", transferCommandHandleApi.RestoreAllTransfer))
		routerTransfer.Post("/permanent/all", params.apiHandler.Handle("delete-all-transfers-permanent", transferCommandHandleApi.DeleteAllTransferPermanent))

	})
	return transferCommandHandleApi
}

// @Summary Create a transfer
// @Tags Transfer Command
// @Security Bearer
// @Description Create a new transfer record
// @Accept json
// @Produce json
// @Param body body requests.CreateTransferRequest true "Transfer request"
// @Success 200 {object} response.ApiResponseTransfer "Transfer data"
// @Failure 400 {object} response.ErrorResponse "Validation Error"
// @Failure 500 {object} response.ErrorResponse "Failed to create transfer"
// @Router /api/transfer-command/create [post]
func (h *transferCommandHandleApi) CreateTransfer(w http.ResponseWriter, r *http.Request) error {
	var body requests.CreateTransferRequest

	if err := httpx.Bind(r, &body); err != nil {
		h.logger.Debug("Invalid request body: ", zap.Error(err))
		return errors.NewBadRequestError("Invalid request format").WithInternal(err)
	}

	if err := body.Validate(); err != nil {
		h.logger.Debug("Validation Error: ", zap.Error(err))
		validations := h.parseValidationErrors(err)
		return errors.NewValidationError(validations)
	}

	ctx := r.Context()

	res, err := h.client.CreateTransfer(ctx, &pb.CreateTransferRequest{
		TransferFrom:   body.TransferFrom,
		TransferTo:     body.TransferTo,
		TransferAmount: int32(body.TransferAmount),
		IdempotencyKey: body.IdempotencyKey,
	})

	if err != nil {
		h.logger.Debug("Failed to create transfer: ", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseTransfer(res)

	h.cache.SetCachedTransferCache(ctx, so)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Update a transfer
// @Tags Transfer Command
// @Security Bearer
// @Description Update an existing transfer record
// @Accept json
// @Produce json
// @Param id path int true "Transfer ID"
// @Param body body requests.UpdateTransferRequest true "Transfer request"
// @Success 200 {object} response.ApiResponseTransfer "Transfer data"
// @Failure 400 {object} response.ErrorResponse "Validation Error"
// @Failure 500 {object} response.ErrorResponse "Failed to update transfer"
// @Router /api/transfer-command/update/{id} [post]
func (h *transferCommandHandleApi) UpdateTransfer(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")

	idInt, err := strconv.Atoi(id)

	if err != nil {
		h.logger.Debug("Bad Request: Invalid ID", zap.Error(err))
		return errors.NewBadRequestError("id is required")
	}

	var body requests.UpdateTransferRequest

	if err := httpx.Bind(r, &body); err != nil {
		h.logger.Debug("Invalid request body: ", zap.Error(err))
		return errors.NewBadRequestError("Invalid request format").WithInternal(err)
	}

	body.TransferID = &idInt

	if err := body.Validate(); err != nil {
		h.logger.Debug("Validation Error: ", zap.Error(err))
		validations := h.parseValidationErrors(err)
		return errors.NewValidationError(validations)
	}

	ctx := r.Context()

	res, err := h.client.UpdateTransfer(ctx, &pb.UpdateTransferRequest{
		TransferId:     int32(idInt),
		TransferFrom:   body.TransferFrom,
		TransferTo:     body.TransferTo,
		TransferAmount: int32(body.TransferAmount),
	})

	if err != nil {
		h.logger.Debug("Failed to update transfer: ", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseTransfer(res)

	h.cache.DeleteTransferCache(ctx, idInt)
	h.cache.SetCachedTransferCache(ctx, so)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Soft delete a transfer
// @Tags Transfer Command
// @Security Bearer
// @Description Soft delete a transfer record by its ID.
// @Accept json
// @Produce json
// @Param id path int true "Transfer ID"
// @Success 200 {object} response.ApiResponseTransfer "Successfully trashed transfer record"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to trashed transfer"
// @Router /api/transfer-command/trash/{id} [post]
func (h *transferCommandHandleApi) TrashTransfer(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")

	idInt, err := strconv.Atoi(id)

	if err != nil {
		h.logger.Debug("Bad Request: Invalid ID", zap.Error(err))
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	res, err := h.client.TrashedTransfer(ctx, &pb.FindByIdTransferRequest{
		TransferId: int32(idInt),
	})

	if err != nil {
		h.logger.Debug("Failed to trash transfer: ", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseTransferDeleteAt(res)

	h.cache.DeleteTransferCache(ctx, idInt)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Restore a trashed transfer
// @Tags Transfer Command
// @Security Bearer
// @Description Restore a trashed transfer record by its ID.
// @Accept json
// @Produce json
// @Param id path int true "Transfer ID"
// @Success 200 {object} response.ApiResponseTransfer "Successfully restored transfer record"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to restore transfer:"
// @Router /api/transfer-command/restore/{id} [post]
func (h *transferCommandHandleApi) RestoreTransfer(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")

	idInt, err := strconv.Atoi(id)

	if err != nil {
		h.logger.Debug("Bad Request: Invalid ID", zap.Error(err))
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	res, err := h.client.RestoreTransfer(ctx, &pb.FindByIdTransferRequest{
		TransferId: int32(idInt),
	})

	if err != nil {
		h.logger.Debug("Failed to restore transfer: ", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseTransferDeleteAt(res)

	h.cache.DeleteTransferCache(ctx, idInt)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Permanently delete a transfer
// @Tags Transfer Command
// @Security Bearer
// @Description Permanently delete a transfer record by its ID.
// @Accept json
// @Produce json
// @Param id path int true "Transfer ID"
// @Success 200 {object} response.ApiResponseTransferDelete "Successfully deleted transfer record permanently"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to delete transfer:"
// @Router /api/transfer-command/permanent/{id} [delete]
func (h *transferCommandHandleApi) DeleteTransferPermanent(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")

	idInt, err := strconv.Atoi(id)

	if err != nil {
		h.logger.Debug("Bad Request: Invalid ID", zap.Error(err))
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	res, err := h.client.DeleteTransferPermanent(ctx, &pb.FindByIdTransferRequest{
		TransferId: int32(idInt),
	})

	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseTransferDelete(res)

	h.cache.DeleteTransferCache(ctx, idInt)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Restore a trashed transfer
// @Tags Transfer Command
// @Security Bearer
// @Description Restore a trashed transfer all
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseTransferAll "Successfully restored transfer record"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to restore transfer:"
// @Router /api/transfer-command/restore/all [post]
func (h *transferCommandHandleApi) RestoreAllTransfer(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	res, err := h.client.RestoreAllTransfer(ctx, &emptypb.Empty{})

	if err != nil {
		h.logger.Error("Failed to restore all transfer", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	h.logger.Debug("Successfully restored all transfer")

	so := h.mapper.ToApiResponseTransferAll(res)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Permanently delete a transfer
// @Tags Transfer Command
// @Security Bearer
// @Description Permanently delete a transfer record all.
// @Accept json
// @Produce json
// @Param id path int true "Transfer ID"
// @Success 200 {object} response.ApiResponseTransferAll "Successfully deleted transfer all"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to delete transfer:"
// @Router /api/transfer-command/permanent/all [post]
func (h *transferCommandHandleApi) DeleteAllTransferPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	res, err := h.client.DeleteAllTransferPermanent(ctx, &emptypb.Empty{})

	if err != nil {
		h.logger.Error("Failed to permanently delete all transfer", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	h.logger.Debug("Successfully deleted all transfer permanently")

	so := h.mapper.ToApiResponseTransferAll(res)

	return httpx.JSON(w, http.StatusOK, so)
}

func (h *transferCommandHandleApi) parseValidationErrors(err error) []errors.ValidationError {
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

func (h *transferCommandHandleApi) getValidationMessage(fe validator.FieldError) string {
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
