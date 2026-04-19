package pcsService

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"smart-pc-agent/internal/config"
	"smart-pc-agent/internal/domain/models"
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
	cfg config.Service,
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
	if err == nil {
		s.pcID = pcID
		return nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("%s: failed to get saved pc id: %w", op, err)
	}

	pcJson, err := json.Marshal(models.Pc{Name: "New PC"})
	if err != nil {
		return fmt.Errorf("%s: failed to marshal new pc data: %w", op, err)
	}

	req, err := s.apiClient.NewRequest(
		ctx,
		http.MethodPost,
		s.url("/pcs"),
		bytes.NewReader(pcJson),
	)
	if err != nil {
		return fmt.Errorf("%s: failed to create create pc request: %w", op, err)
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := authorization.DoRequest[models.Pc](s.apiClient, req)
	if err != nil {
		return fmt.Errorf("%s: failed to do create pc request: %w", op, err)
	}

	if resp.Status != response.StatusOK {
		return fmt.Errorf("%s: failed to create new pc request, status: %s", op, resp.Status)
	}

	s.pcID = resp.Data.ID

	saveErr := setter.SetPcID(ctx, resp.Data.ID)
	if saveErr == nil {
		return nil
	}

	resp, err = authorization.DoNewRequest[models.Pc](
		ctx,
		s.apiClient,
		http.MethodDelete,
		s.pcURL(""),
		nil,
	)
	if err != nil {
		return fmt.Errorf(
			"%s: failed to save pc id: %w: failed to do delete pc by id request after pc id save failed: %w",
			op,
			saveErr,
			err,
		)
	}

	if resp.Status != response.StatusOK {
		return fmt.Errorf(
			"%s: failed to save pc id: %w: delete pc by id request is failed (status: %s) after pc id save failed: %w",
			op,
			saveErr,
			resp.Status,
			err,
		)
	}

	return fmt.Errorf("%s: failed to save pc id: %w", op, saveErr)
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
