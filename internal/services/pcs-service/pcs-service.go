package pcsService

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"smart-pc-agent/internal/config"
	"smart-pc-agent/internal/domain/models"
	"smart-pc-agent/internal/services"
	"smart-pc-agent/internal/storage"

	"github.com/MaxRomanov007/smart-pc-go-lib/api/response"
	"github.com/MaxRomanov007/smart-pc-go-lib/authorization"
)

type Service struct {
	apiClient *authorization.ApiClient
	baseURL   string
	pcID      string
}

type PcIDGetter interface {
	GetPcID(ctx context.Context) (string, error)
}

type PcIDSetter interface {
	SetPcID(ctx context.Context, id string) error
}

func New(
	ctx context.Context,
	auth *authorization.Auth,
	cfg config.PcsService,
	getter PcIDGetter,
	setter PcIDSetter,
) (*Service, error) {
	const op = "pcs-service.New"

	client, err := auth.NewApiClient(ctx, &http.Client{
		Timeout: cfg.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create api client: %w", op, err)
	}

	service := &Service{
		apiClient: client,
		baseURL:   cfg.BaseURL,
	}

	if err := service.ensurePcIDCreated(ctx, getter, setter); err != nil {
		return nil, fmt.Errorf("%s: failed to get pc id: %w", op, err)
	}

	return service, nil
}

func (s *Service) ensurePcIDCreated(
	ctx context.Context,
	getter PcIDGetter,
	setter PcIDSetter,
) error {
	const op = "pcs-service.ensurePcIDCreated"

	pcID, err := getter.GetPcID(ctx)
	if errors.Is(err, storage.ErrNotFound) {
		if err := s.createAndSaveNewPc(ctx, setter); err != nil {
			return fmt.Errorf("%s: failed to create and save new pc: %w", op, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: failed to get saved pc id: %w", op, err)
	}

	_, err = s.GetPc(ctx, pcID)
	if errors.Is(err, services.ErrNotFound) {
		if err := s.createAndSaveNewPc(ctx, setter); err != nil {
			return fmt.Errorf("%s: failed to renew and save pc: %w", op, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: failed to get saved pc from server: %w", op, err)
	}

	s.pcID = pcID
	return nil
}

func (s *Service) createAndSaveNewPc(ctx context.Context, setter PcIDSetter) error {
	const op = "pcs-service.createAndSaveNewPc"

	newPc, err := s.CreatePc(ctx, models.Pc{Name: "New PC"})
	if err != nil {
		return fmt.Errorf("%s: failed to create new pc: %w", op, err)
	}

	saveErr := setter.SetPcID(ctx, newPc.ID)
	if saveErr != nil {
		if _, err := s.DeletePc(ctx, newPc.ID); err != nil {
			return fmt.Errorf(
				"%s: failed to delete saved pc (error: %w) after save pc id failed (error: %w)",
				op,
				err,
				saveErr,
			)
		}

		return fmt.Errorf("%s: failed to save pc id: %w", op, saveErr)
	}

	s.pcID = newPc.ID
	return nil
}

func (s *Service) url(endpoint string) string {
	return fmt.Sprintf("%s/u/%s%s", s.baseURL, s.apiClient.UID, endpoint)
}

func (s *Service) pcURL(endpoint string) string {
	return s.url(fmt.Sprintf("/pcs/%s%s", s.pcID, endpoint))
}

func (s *Service) pcCommandUrl(commandID, endpoint string) string {
	return s.pcURL(fmt.Sprintf("/commands/%s%s", commandID, endpoint))
}

func (s *Service) CreatePc(ctx context.Context, pc models.Pc) (models.Pc, error) {
	const op = "pcs-service.CreatePc"

	resp, err := authorization.DoNewRequest[models.Pc](
		ctx,
		s.apiClient,
		http.MethodPost,
		s.url("/pcs"),
		pc,
	)
	if err != nil {
		return models.Pc{}, fmt.Errorf("%s: failed to do request: %w", op, err)
	}

	if resp.Status != response.StatusOK {
		return models.Pc{}, fmt.Errorf("%s: response status is not ok: %s", op, resp.Status)
	}

	return *resp.Data, nil
}

func (s *Service) DeletePc(ctx context.Context, id string) (models.Pc, error) {
	const op = "pcs-service.DeletePc"

	resp, err := authorization.DoNewRequest[models.Pc](
		ctx,
		s.apiClient,
		http.MethodDelete,
		s.url(fmt.Sprintf("/pcs/%s", id)),
		nil,
	)
	if err != nil {
		return models.Pc{}, fmt.Errorf(
			"%s: failed to do delete pc by id: %w",
			op,
			err,
		)
	}

	if resp.Status != response.StatusOK {
		return models.Pc{}, fmt.Errorf(
			"%s: response status is not ok: %s",
			op,
			resp.Status,
		)
	}

	return *resp.Data, nil
}

func (s *Service) GetPc(ctx context.Context, id string) (models.Pc, error) {
	const op = "pcs-service.GetPc"

	resp, err := authorization.DoNewRequest[models.Pc](
		ctx,
		s.apiClient,
		http.MethodGet,
		s.url(fmt.Sprintf("/pcs/%s", id)),
		nil,
	)
	if err != nil {
		return models.Pc{}, fmt.Errorf(
			"%s: failed to do get pc by id: %w",
			op,
			err,
		)
	}

	if resp.Status == response.StatusNotFound {
		return models.Pc{}, services.ErrNotFound
	}
	if resp.Status != response.StatusOK {
		return models.Pc{}, fmt.Errorf(
			"%s: response status is not ok: %s",
			op,
			resp.Status,
		)
	}

	return *resp.Data, nil
}

func (s *Service) GetCommands(ctx context.Context) ([]models.Command, error) {
	const op = "pcs-service.GetCommands"

	resp, err := authorization.DoNewRequest[[]models.Command](
		ctx,
		s.apiClient,
		http.MethodGet,
		s.pcURL("/commands"),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to do request: %w", op, err)
	}

	if resp.Status != response.StatusOK {
		return nil, fmt.Errorf("%s: status is not OK: %s", op, resp.Status)
	}

	return *resp.Data, nil
}

func (s *Service) GetCommandParameters(
	ctx context.Context,
	id string,
) ([]models.CommandParameter, error) {
	const op = "pcs-service.GetCommandParameters"

	resp, err := authorization.DoNewRequest[[]models.CommandParameter](
		ctx,
		s.apiClient,
		http.MethodGet,
		s.pcCommandUrl(id, "/parameters"),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to do request: %w", op, err)
	}

	if resp.Status != response.StatusOK {
		return nil, fmt.Errorf("%s: status is not OK: %s", op, resp.Status)
	}

	return *resp.Data, nil
}

func (s *Service) CreatePcCommand(
	ctx context.Context,
	command models.Command,
) (models.Command, error) {
	const op = "pcs-service.CreatePcCommand"

	resp, err := authorization.DoNewRequest[models.Command](
		ctx,
		s.apiClient,
		http.MethodPost,
		s.pcURL("/commands"),
		command,
	)
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to do request: %w", op, err)
	}

	if resp.Status != response.StatusOK {
		return models.Command{}, fmt.Errorf("%s: response status is not ok: %s", op, resp.Status)
	}

	return *resp.Data, nil
}

func (s *Service) DeletePcCommand(ctx context.Context, id string) (models.Command, error) {
	const op = "pcs-service.DeletePcCommand"

	resp, err := authorization.DoNewRequest[models.Command](
		ctx,
		s.apiClient,
		http.MethodDelete,
		s.pcCommandUrl(id, ""),
		nil,
	)
	if err != nil {
		return models.Command{}, fmt.Errorf(
			"%s: failed to do delete pc command by id: %w",
			op,
			err,
		)
	}

	if resp.Status == response.StatusNotFound {
		return models.Command{}, services.ErrNotFound
	}
	if resp.Status != response.StatusOK {
		return models.Command{}, fmt.Errorf(
			"%s: response status is not ok: %s",
			op,
			resp.Status,
		)
	}

	return *resp.Data, nil
}

func (s *Service) UpdatePcCommand(
	ctx context.Context,
	command models.Command,
) (models.Command, error) {
	const op = "pcs-service.UpdatePcCommand"

	resp, err := authorization.DoNewRequest[models.Command](
		ctx,
		s.apiClient,
		http.MethodPatch,
		s.pcCommandUrl(command.ID, ""),
		command,
	)
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to do request: %w", op, err)
	}

	if resp.Status != response.StatusOK {
		return models.Command{}, fmt.Errorf("%s: response status is not ok: %s", op, resp.Status)
	}

	return *resp.Data, nil
}
