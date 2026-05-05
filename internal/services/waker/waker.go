package waker

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"smart-pc-agent/internal/config"
	"strings"
	"time"

	"github.com/libp2p/go-netroute"
)

type Service struct {
	client       http.Client
	BaseURL      string
	checkTimeout time.Duration
}

func New(cfg config.Waker) (*Service, error) {
	const op = "services.waker.New"

	baseURL := cfg.BaseURL
	if baseURL == "" {
		router, err := netroute.New()
		if err != nil {
			return nil, fmt.Errorf(
				"%s: failed to initialize netroute on base url is empty: %w",
				op,
				err,
			)
		}

		_, gw, _, err := router.Route(net.ParseIP("8.8.8.8"))
		if err != nil {
			return nil, fmt.Errorf(
				"%s: failed to get default gateway on base url is empty: %w",
				op,
				err,
			)
		}

		baseURL = fmt.Sprintf("http://%s:%d", gw, 8506)
	}

	return &Service{
		client: http.Client{
			Timeout: cfg.Timeout,
		},
		BaseURL:      baseURL,
		checkTimeout: cfg.CheckTimeout,
	}, nil
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
