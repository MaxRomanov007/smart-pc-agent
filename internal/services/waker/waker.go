package waker

import (
	"context"
	"errors"
	"fmt"
	"go/types"
	"log/slog"
	"net"
	"net/http"
	"smart-pc-agent/internal/config"
	"strings"
	"time"

	"github.com/MaxRomanov007/smart-pc-go-lib/api/response"
	apiclient "github.com/MaxRomanov007/smart-pc-go-lib/authorization/api-client"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/libp2p/go-netroute"
)

type Service struct {
	client       *apiclient.Client
	BaseURL      string
	checkTimeout time.Duration
	mac          string
	done         chan struct{}
	pcID         string
}

const (
	sshPort = 22
)

func New(ctx context.Context, cfg config.Waker, log *slog.Logger) (*Service, error) {
	const op = "services.waker.New"

	router, err := netroute.New()
	if err != nil {
		return nil, fmt.Errorf(
			"%s: failed to initialize netroute on base url is empty: %w",
			op,
			err,
		)
	}

	iface, gw, _, err := router.Route(net.ParseIP("8.8.8.8"))
	if err != nil {
		return nil, fmt.Errorf(
			"%s: failed to get default gateway on base url is empty: %w",
			op,
			err,
		)
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%d", gw, 8506)
	}

	client := apiclient.New(&http.Client{Timeout: cfg.Timeout}, new(TokenProviderDiscard))

	service := &Service{
		client:       client,
		BaseURL:      baseURL,
		checkTimeout: cfg.CheckTimeout,
		mac:          iface.HardwareAddr.String(),
		done:         make(chan struct{}),
	}

	go func() {
		defer close(service.done)
		<-ctx.Done()

		const component = "waker/register-on-exit"
		log := log.With(sl.Component(component))

		isAvailable, err := service.IsAvailable()
		if err != nil {
			log.Error("failed to check if waker is available", sl.Err(err))
			return
		}

		if !isAvailable {
			log.Info("waker is not available")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.RegisterTimeout)
		defer cancel()

		if err := service.Register(ctx); err != nil {
			log.Error("failed to register pc in waker", sl.Err(err))
			return
		}

		log.Info("pc registered on waker")
	}()

	return service, nil
}

func (s *Service) SetPcID(pcID string) {
	s.pcID = pcID
}

func (s *Service) IsAvailable() (bool, error) {
	const op = "services.waker.isAvailable"

	conn, err := net.DialTimeout("tcp", s.getAddress(), s.checkTimeout)
	if err, ok := errors.AsType[net.Error](err); ok && err.Timeout() {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: failed to dial timeout: %w", op, err)
	}
	if err := conn.Close(); err != nil {
		return true, fmt.Errorf("%s: failed to close connection: %w", op, err)
	}

	return true, nil
}

func (s *Service) IsSSHAvailable() (bool, error) {
	const op = "services.waker.isSSHAvailable"

	ip, err := s.IP()
	if err != nil {
		return false, fmt.Errorf("%s: failed to get ip: %w", op, err)
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip.String(), sshPort), s.checkTimeout)
	if err, ok := errors.AsType[net.Error](err); ok && err.Timeout() {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: failed to dial timeout: %w", op, err)
	}
	if err := conn.Close(); err != nil {
		return true, fmt.Errorf("%s: failed to close connection: %w", op, err)
	}

	return true, nil
}

func (s *Service) IP() (net.IP, error) {
	const op = "services.waker.IP"

	ipStr := strings.Split(s.getAddress(), ":")[0]
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf(
			"%s: failed to get ip (calulated ip is not valid) (calculated: %s)",
			op,
			ipStr,
		)
	}

	return ip, nil
}

func (s *Service) getAddress() string {
	schemaSplit := strings.Split(s.BaseURL, "://")
	slashSplit := strings.Split(schemaSplit[len(schemaSplit)-1], "/")
	return slashSplit[0]
}

type Registered struct {
	ID  string `json:"id"`
	MAC string `json:"mac"`
}

func (s *Service) Register(ctx context.Context) error {
	const op = "services.waker.Register"

	resp, err := apiclient.Send[types.Nil](
		ctx,
		s.client,
		http.MethodPost,
		s.url("/registered"),
		Registered{
			ID:  s.pcID,
			MAC: s.mac,
		},
	)
	if err != nil {
		return fmt.Errorf("%s: failed to do request: %w", op, err)
	}

	if resp.Status != response.StatusOK {
		return fmt.Errorf("%s: response status is not ok: %s", op, resp.Status)
	}

	return nil
}

type authStatus struct {
	Authorized bool `json:"authorized"`
}

func (s *Service) IsAuthorized(ctx context.Context) (bool, error) {
	const op = "services.waker.IsAuthorized"

	resp, err := apiclient.Send[authStatus](
		ctx,
		s.client,
		http.MethodGet,
		s.url("/auth/status"),
		nil,
	)
	if err != nil {
		return false, fmt.Errorf("%s: failed to do request: %w", op, err)
	}

	if resp.Status != response.StatusOK {
		return false, fmt.Errorf("%s: response status is not ok: %s", op, resp.Status)
	}

	return resp.Data.Authorized, nil
}

type urlResponse struct {
	URL string `json:"url"`
}

func (s *Service) AuthorizeURL(ctx context.Context) (string, error) {
	const op = "services.waker.IsAuthorized"

	resp, err := apiclient.Send[urlResponse](
		ctx,
		s.client,
		http.MethodGet,
		s.url("/auth/url"),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("%s: failed to do request: %w", op, err)
	}

	if resp.Status != response.StatusOK {
		return "", fmt.Errorf("%s: response status is not ok: %s", op, resp.Status)
	}

	return resp.Data.URL, nil
}

func (s *Service) url(path string) string {
	return fmt.Sprintf("%s%s", s.BaseURL, path)
}

type TokenProviderDiscard struct{}

func (t *TokenProviderDiscard) Token(_ context.Context) (string, error) {
	return "", nil
}

func (s *Service) Done() <-chan struct{} {
	return s.done
}
