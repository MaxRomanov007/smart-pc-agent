package pcsService

import (
	"context"
	"fmt"
	"net/http"
	"smart-pc-agent/internal/config"
	"smart-pc-agent/internal/domain/models"

	"github.com/MaxRomanov007/smart-pc-go-lib/api/response"
	"github.com/MaxRomanov007/smart-pc-go-lib/authorization"
)

type Service struct {
	apiClient *authorization.ApiClient
	baseURL   string
	pcID      string
}

func New(
	ctx context.Context,
	auth *authorization.Auth,
	cfg config.Service,
	pcID string,
) (*Service, error) {
	const op = "pcs-service.New"

	client, err := auth.NewApiClient(ctx, &http.Client{
		Timeout: cfg.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create api client: %w", op, err)
	}

	return &Service{
		apiClient: client,
		baseURL:   cfg.BaseURL,
		pcID:      pcID,
	}, nil
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
