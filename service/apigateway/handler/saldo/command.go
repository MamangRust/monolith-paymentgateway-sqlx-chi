package saldohandler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/apierror"
	pb "github.com/MamangRust/monolith-payment-gateway-pb/saldo"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	apimapper "github.com/MamangRust/monolith-payment-gateway-shared/mapper/saldo"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	saldo_cache "github.com/MamangRust/monolith-payment-gateway-apigateway/redis/api/saldo"
)

type saldoCommandHandleApi struct {
	saldo pb.SaldoCommandServiceClient

	logger logger.LoggerInterface

	mapper apimapper.SaldoCommandResponseMapper

	cache saldo_cache.SaldoMencache

	apiHandler apierror.ApiHandler
}

type saldoCommandHandleDeps struct {
	client pb.SaldoCommandServiceClient

	router chi.Router

	logger logger.LoggerInterface

	mapper apimapper.SaldoCommandResponseMapper

	cache saldo_cache.SaldoMencache

	apiHandler apierror.ApiHandler
}

func NewSaldoCommandHandleApi(params *saldoCommandHandleDeps) *saldoCommandHandleApi {

	saldoHandler := &saldoCommandHandleApi{
		saldo:      params.client,
		logger:     params.logger,
		mapper:     params.mapper,
		cache:      params.cache,
		apiHandler: params.apiHandler,
	}

	params.router.Route("/api/saldo-command", func(routerSaldo chi.Router) {

		routerSaldo.Post("/create", params.apiHandler.Handle("create-saldo", saldoHandler.Create))
		routerSaldo.Post("/update/{id}", params.apiHandler.Handle("update-saldo", saldoHandler.Update))
		routerSaldo.Post("/trashed/{id}", params.apiHandler.Handle("trash-saldo", saldoHandler.TrashSaldo))
		routerSaldo.Post("/restore/{id}", params.apiHandler.Handle("restore-saldo", saldoHandler.RestoreSaldo))
		routerSaldo.Delete("/permanent/{id}", params.apiHandler.Handle("delete-saldo-permanent", saldoHandler.Delete))

		routerSaldo.Post("/restore/all", params.apiHandler.Handle("restore-all-saldos", saldoHandler.RestoreAllSaldo))
		routerSaldo.Post("/permanent/all", params.apiHandler.Handle("delete-all-saldos-permanent", saldoHandler.DeleteAllSaldoPermanent))

	})
	return saldoHandler
}

// @Summary Create a new saldo
// @Tags Saldo Command
// @Security Bearer
// @Description Create a new saldo record with the provided card number and total balance.
// @Accept json
// @Produce json
// @Param CreateSaldoRequest body requests.CreateSaldoRequest true "Create Saldo Request"
// @Success 200 {object} response.ApiResponseSaldo "Successfully created saldo record"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid request body or validation error"
// @Failure 500 {object} response.ErrorResponse "Failed to create saldo"
// @Router /api/saldo-command/create [post]
func (h *saldoCommandHandleApi) Create(w http.ResponseWriter, r *http.Request) error {
	var body requests.CreateSaldoRequest

	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("Invalid request")
	}

	if err := body.Validate(); err != nil {
		validations := h.parseValidationErrors(err)
		return errors.NewValidationError(validations)
	}

	ctx := r.Context()

	res, err := h.saldo.CreateSaldo(ctx, &pb.CreateSaldoRequest{
		CardNumber:   body.CardNumber,
		TotalBalance: int32(body.TotalBalance),
	})

	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseSaldo(res)

	h.cache.SetCachedSaldoById(ctx, so.Data.ID, so)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Update an existing saldo
// @Tags Saldo Command
// @Security Bearer
// @Description Update an existing saldo record with the provided card number and total balance.
// @Accept json
// @Produce json
// @Param id path int true "Saldo ID"
// @Param UpdateSaldoRequest body requests.UpdateSaldoRequest true "Update Saldo Request"
// @Success 200 {object} response.ApiResponseSaldo "Successfully updated saldo record"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid request body or validation error"
// @Failure 500 {object} response.ErrorResponse "Failed to update saldo"
// @Router /api/saldo-command/update/{id} [post]
func (h *saldoCommandHandleApi) Update(w http.ResponseWriter, r *http.Request) error {
	idint, err := strconv.Atoi(chi.URLParam(r, "id"))

	if err != nil {
		return errors.NewBadRequestError("id is required")
	}

	var body requests.UpdateSaldoRequest

	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("Invalid request")
	}

	if err := body.Validate(); err != nil {
		validations := h.parseValidationErrors(err)
		return errors.NewValidationError(validations)
	}

	ctx := r.Context()

	res, err := h.saldo.UpdateSaldo(ctx, &pb.UpdateSaldoRequest{
		SaldoId:      int32(idint),
		CardNumber:   body.CardNumber,
		TotalBalance: int32(body.TotalBalance),
	})

	if err != nil {
		h.logger.Debug("Failed to update saldo", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseSaldo(res)

	h.cache.DeleteSaldoCache(ctx, idint)

	h.cache.SetCachedSaldoById(ctx, so.Data.ID, so)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Soft delete a saldo
// @Tags Saldo Command
// @Security Bearer
// @Description Soft delete an existing saldo record by its ID.
// @Accept json
// @Produce json
// @Param id path int true "Saldo ID"
// @Success 200 {object} response.ApiResponseSaldo "Successfully trashed saldo record"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to trashed saldo"
// @Router /api/saldo-command/trashed/{id} [post]
func (h *saldoCommandHandleApi) TrashSaldo(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")

	idInt, err := strconv.Atoi(id)

	if err != nil {
		h.logger.Debug("Bad Request: Invalid ID", zap.Error(err))
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	res, err := h.saldo.TrashedSaldo(ctx, &pb.FindByIdSaldoRequest{
		SaldoId: int32(idInt),
	})

	if err != nil {
		h.logger.Debug("Failed to trashed saldo", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseSaldoDeleteAt(res)

	h.cache.DeleteSaldoCache(ctx, idInt)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Restore a trashed saldo
// @Tags Saldo Command
// @Security Bearer
// @Description Restore an existing saldo record from the trash by its ID.
// @Accept json
// @Produce json
// @Param id path int true "Saldo ID"
// @Success 200 {object} response.ApiResponseSaldo "Successfully restored saldo record"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to restore saldo"
// @Router /api/saldo-command/restore/{id} [post]
func (h *saldoCommandHandleApi) RestoreSaldo(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")

	idInt, err := strconv.Atoi(id)

	if err != nil {
		h.logger.Debug("Bad Request: Invalid ID", zap.Error(err))
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	res, err := h.saldo.RestoreSaldo(ctx, &pb.FindByIdSaldoRequest{
		SaldoId: int32(idInt),
	})

	if err != nil {
		h.logger.Debug("Failed to restore saldo", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseSaldoDeleteAt(res)

	h.cache.DeleteSaldoCache(ctx, idInt)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Permanently delete a saldo
// @Tags Saldo Command
// @Security Bearer
// @Description Permanently delete an existing saldo record by its ID.
// @Accept json
// @Produce json
// @Param id path int true "Saldo ID"
// @Success 200 {object} response.ApiResponseSaldoDelete "Successfully deleted saldo record"
// @Failure 400 {object} response.ErrorResponse "Bad Request: Invalid ID"
// @Failure 500 {object} response.ErrorResponse "Failed to delete saldo"
// @Router /api/saldo-command/permanent/{id} [delete]
func (h *saldoCommandHandleApi) Delete(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")

	idInt, err := strconv.Atoi(id)

	if err != nil {
		h.logger.Debug("Bad Request: Invalid ID", zap.Error(err))
		return errors.NewBadRequestError("id is required")
	}

	ctx := r.Context()

	res, err := h.saldo.DeleteSaldoPermanent(ctx, &pb.FindByIdSaldoRequest{
		SaldoId: int32(idInt),
	})

	if err != nil {
		h.logger.Debug("Failed to delete saldo", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	so := h.mapper.ToApiResponseSaldoDelete(res)

	h.cache.DeleteSaldoCache(ctx, idInt)

	return httpx.JSON(w, http.StatusOK, so)
}

// RestoreAllSaldo restores all saldo records.
// @Summary Restore all saldo records
// @Tags Saldo Command
// @Security Bearer
// @Description Restore all saldo records that were previously deleted.
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseSaldoAll "Successfully restored all saldo records"
// @Failure 500 {object} response.ErrorResponse "Failed to restore all saldo records"
// @Router /api/saldo-command/restore/all [post]
func (h *saldoCommandHandleApi) RestoreAllSaldo(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	res, err := h.saldo.RestoreAllSaldo(ctx, &emptypb.Empty{})

	if err != nil {
		h.logger.Error("Failed to restore all saldo", zap.Error(err))
		return errors.ParseGrpcError(err)
	}

	h.logger.Debug("Successfully restored all saldo")

	so := h.mapper.ToApiResponseSaldoAll(res)

	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Permanently delete all saldo records
// @Tags Saldo Command
// @Security Bearer
// @Description Permanently delete all saldo records from the database.
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseSaldoAll "Successfully deleted all saldo records permanently"
// @Failure 500 {object} response.ErrorResponse "Failed to permanently delete all saldo records"
// @Router /api/saldo-command/permanent/all [post]
func (h *saldoCommandHandleApi) DeleteAllSaldoPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	res, err := h.saldo.DeleteAllSaldoPermanent(ctx, &emptypb.Empty{})

	if err != nil {
		h.logger.Error("Failed to permanently delete all saldo", zap.Error(err))

		return errors.ParseGrpcError(err)
	}

	h.logger.Debug("Successfully deleted all saldo permanently")

	so := h.mapper.ToApiResponseSaldoAll(res)

	return httpx.JSON(w, http.StatusOK, so)
}

func (h *saldoCommandHandleApi) parseValidationErrors(err error) []errors.ValidationError {
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

func (h *saldoCommandHandleApi) getValidationMessage(fe validator.FieldError) string {
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
