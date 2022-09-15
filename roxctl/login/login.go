package login

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/auth"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"github.com/stackrox/rox/roxctl/config"
)

type loginCommand struct {
	// Properties that are injected or constructed.
	env     environment.Environment
	timeout time.Duration
}

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cbr := &cobra.Command{
		Use: "login",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return makeLoginCommand(cliEnvironment, c).login()
		}),
	}

	flags.AddTimeout(cbr)
	return cbr
}

func makeLoginCommand(cliEnvironment environment.Environment, cbr *cobra.Command) *loginCommand {
	return &loginCommand{
		env:     cliEnvironment,
		timeout: flags.Timeout(cbr),
	}
}

func (cmd *loginCommand) login() error {
	conn, err := cmd.env.GRPCConnection(auth.Anonymous())
	if err != nil {
		return fmt.Errorf("could not get gRPC connection: %w", err)
	}
	defer utils.IgnoreError(conn.Close)

	ctx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()

	_, err = v1.NewMetadataServiceClient(conn).GetMetadata(ctx, &v1.Empty{})
	if err != nil {
		return fmt.Errorf("could not query server metadata: %w", err)
	}

	handler := newLoginHandler(cmd.env)
	if err := handler.run(); err != nil {
		return err
	}

	return nil
}

type loginHandler struct {
	env environment.Environment

	ret     concurrency.ErrorSignal
	baseURL string
}

func newLoginHandler(env environment.Environment) *loginHandler {
	return &loginHandler{
		env: env,
	}
}

func (h *loginHandler) run() error {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("could not listen on TCP socket: %w", err)
	}
	h.baseURL = fmt.Sprintf("http://%s", lis.Addr())
	mux := http.NewServeMux()
	mux.HandleFunc("/login", h.serveLogin)
	mux.HandleFunc("/callback", h.serveCallback)

	httpSrv := &http.Server{
		Handler: mux,
	}
	defer utils.IgnoreError(httpSrv.Close)

	srvErrC := make(chan error, 1)
	go func() {
		srvErrC <- httpSrv.Serve(lis)
	}()

	h.ret.Reset()
	loginURL := h.baseURL + "/login"

	if err := browser.OpenURL(loginURL); err != nil {
		h.env.Logger().WarnfLn("Failed to open URL in browser: %v", err)
	}

	h.env.Logger().PrintfLn("Please complete the authorization flow in the browser.")
	h.env.Logger().PrintfLn("If no browser window opens, please click on the following URL:")
	h.env.Logger().PrintfLn("")
	h.env.Logger().PrintfLn("    %s", loginURL)
	h.env.Logger().PrintfLn("")

	select {
	case err := <-srvErrC:
		return fmt.Errorf("http server error: %w", err)
	case <-h.ret.Done():
		if err := h.ret.Err(); err != nil {
			return fmt.Errorf("flow error: %w", err)
		}
		return nil
	}
}

func (h *loginHandler) serveLogin(w http.ResponseWriter, req *http.Request) {
	q := make(url.Values)
	q.Set("authorizeCallback", h.baseURL+"/callback")

	authorizeURL := h.env.BaseURL()
	authorizeURL.Path = "/authorize-cli"
	authorizeURL.Fragment = q.Encode()
	w.Header().Set("Location", authorizeURL.String())
	w.WriteHeader(http.StatusSeeOther)
}

func (h *loginHandler) serveCallback(w http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	token := q.Get("token")
	if token == "" {
		_, _ = fmt.Fprintln(w, "Error: no token in response")
		h.ret.SignalWithError(errors.New("no token in response"))
		return
	}
	var expiresAt time.Time
	if expiryStr := q.Get("expiresAt"); expiryStr != "" {
		var err error
		expiresAt, err = time.Parse(time.RFC3339, expiryStr)
		if err != nil {
			_, _ = fmt.Fprintf(w, "Warning: could not parse expiresAt response %q: %v\n", expiryStr, err)
			expiresAt = time.Time{}
		}
	}

	refreshToken := q.Get("refreshToken")

	_, _ = fmt.Fprintln(w, "Authorization successful!")
	_, _ = fmt.Fprintln(w, "You can now safely close this window")

	h.env.Logger().InfofLn("Token: %s", token)
	if !expiresAt.IsZero() {
		h.env.Logger().InfofLn("Expires: %v", expiresAt)
	}
	if refreshToken != "" {
		h.env.Logger().InfofLn("Refresh token: %s", refreshToken)
	}

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	if cfg.Hosts == nil {
		cfg.Hosts = make(map[string]*config.HostConfig)
	}
	hc := cfg.Hosts[h.env.BaseURL().String()]
	if hc == nil {
		hc = &config.HostConfig{}
		cfg.Hosts[h.env.BaseURL().String()] = hc
	}
	ha := hc.Access
	if ha == nil {
		hc.Access = &config.HostAccessConfig{}
		ha = hc.Access
	}
	ha.Token = token
	ha.ExpiresAt = expiresAt
	ha.RefreshToken = refreshToken

	if err := config.Store(cfg); err != nil {
		panic(err)
	}

	h.ret.Signal()
}
