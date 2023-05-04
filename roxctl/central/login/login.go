package login

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

const (
	loginPath     = "/login"
	callbackPath  = "/callback"
	authorizePath = "/authorize-roxctl"
)

// Command provides a command that obtains a token valid for a central instance with an authorization flow.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	loginCmd := loginCommand{env: cliEnvironment}

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to the central instance to obtain a token",
		Long: `Login to the central instance to obtain a token used within roxctl.
This is done by opening a browser, interactively logging in to an auth provider of your choice.

The login token itself will be stored under $HOME/.roxctl/login and used to re-authenticate.`,
		RunE: util.RunENoArgs(func(command *cobra.Command) error {
			if err := loginCmd.construct(command); err != nil {
				return err
			}
			return loginCmd.login()
		}),
	}

	flags.AddTimeoutWithDefault(cmd, 5*time.Minute)

	return cmd
}

type loginCommand struct {
	timeout time.Duration

	env environment.Environment

	// loginSignal is used within the login flow and is signaled when the interactive authorization flow has finished,
	// including any potential errors that occurred during the flow.
	loginSignal concurrency.ErrorSignal

	centralURL *url.URL
}

func (l *loginCommand) construct(cmd *cobra.Command) error {
	l.timeout = flags.Timeout(cmd)
	l.loginSignal = concurrency.NewErrorSignal()
	centralURL, err := flags.CentralURL()
	if err != nil {
		return errors.Wrap(err, "retrieving central URL")
	}
	l.centralURL = centralURL
	return nil
}

func (l *loginCommand) login() error {
	// Use a random port reported as free and usable by the kernel.
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return errors.Wrap(err, "listening on TCP socket")
	}
	localURL := fmt.Sprintf("http://%s", listener.Addr())
	loginURL, err := url.JoinPath(localURL, loginPath)
	if err != nil {
		return errors.Wrap(err, "constructing login URL")
	}
	callbackURL, err := url.JoinPath(localURL, callbackPath)
	if err != nil {
		return errors.Wrap(err, "constructing callback URL")
	}

	mux := http.NewServeMux()
	mux.HandleFunc(loginPath, l.loginHandle(callbackURL))
	mux.HandleFunc(callbackPath, l.callbackHandle)

	server := http.Server{
		Handler: mux,
		Addr:    localURL,
	}
	defer utils.IgnoreError(server.Close)

	serverErrorC := make(chan error, 1)
	go func() {
		serverErrorC <- server.Serve(listener)
	}()

	l.env.Logger().PrintfLn(`Please complete the authorization flow in the browser with an auth provider of your choice.
If no browser window opens, please click on the following URL:
        %s
`, loginURL)

	if err := browser.OpenURL(loginURL); err != nil {
		l.env.Logger().WarnfLn("Failed to open URL in browser: %v", err)
	}

	select {
	case err := <-serverErrorC:
		return errors.Wrap(err, "error from HTTP server")

	case <-time.After(l.timeout):
		l.env.Logger().ErrfLn(`Waited %s for the authorization flow to succeed, but did not finish.
In case you want to increase the timeout, use the --timeout flag.`, l.timeout.String())
		if err := server.Close(); err != nil {
			return err
		}
		return errors.New("ran into timeout during authorization flow")

	case <-l.loginSignal.Done():
		if err := l.loginSignal.Err(); err != nil {
			return errors.Wrap(err, "error within authorization flow")
		}
		return server.Close()
	}
}

// loginHandle provides the http.HandlerFunc for the login path of the authorization flow.
// It will set the callback URL and initiates the authorization flow by redirecting to the central.
func (l *loginCommand) loginHandle(callbackURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		queryParams := make(url.Values)
		queryParams.Set(authproviders.AuthorizeCallbackQueryParameter, callbackURL)
		authorizeURL := l.centralURL
		authorizeURL.Path = authorizePath
		authorizeURL.Fragment = queryParams.Encode()
		w.Header().Set("Location", authorizeURL.String())
		w.WriteHeader(http.StatusSeeOther)
	}
}

// callBackHandle provides the http.HandlerFunc for the callback path of the authorization flow.
// It will parse the response from central, specifically parsing the token, expiresAt, and refreshToken query parameters.
// Afterward, the received login information will be persisted locally under a well-known path ($HOME/.roxctl/login).
func (l *loginCommand) callbackHandle(w http.ResponseWriter, req *http.Request) {
	queryParams := req.URL.Query()

	// In case the query parameter error is set, mark this as failed.
	err := queryParams.Get("error")
	if err != "" {
		errDescription := queryParams.Get("errorDescription")
		err = utils.IfThenElse(errDescription != "", fmt.Sprintf("%s: %s", err, errDescription), err)
		_, _ = fmt.Fprintf(w, "Error: Failed the authorization flow %s\n", err)
		l.loginSignal.SignalWithError(fmt.Errorf("failed the authorization flow: %s", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// The token should be contained within the query parameter as "token"
	token := queryParams.Get(authproviders.TokenQueryParameter)
	if token == "" {
		_, _ = fmt.Fprintln(w, "Error: No token found within response from Central")
		l.loginSignal.SignalWithError(errox.InvalidArgs.New("no token found in authorization response from central"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var expiresAt time.Time
	if expiry := queryParams.Get(authproviders.ExpiresAtQueryParameter); expiry != "" {
		parsedExpiration, err := time.Parse(time.RFC3339, expiry)
		if err != nil {
			_, _ = fmt.Fprintf(w, "Warning: expiresAt could not be parsed from response %q: %v", expiry, err)
		}
		expiresAt = utils.IfThenElse(err != nil, time.Time{}, parsedExpiration)
	}

	// Refresh token is not required, as it may or may not be set depending on the used auth provider.
	refreshToken := queryParams.Get(authproviders.RefreshTokenQueryParameter)
	_, _ = fmt.Fprint(w, `Autentication successful!

You may now close this window.
`)

	l.env.Logger().InfofLn("Received the following after the authorization flow from Central:")
	l.env.Logger().InfofLn("Access token: %s", token)
	if !expiresAt.IsZero() {
		l.env.Logger().InfofLn("Access token expiration: %v", expiresAt)
	}
	if refreshToken != "" {
		l.env.Logger().InfofLn("Refresh token: %s", refreshToken)
	}
	l.env.Logger().InfofLn("Storing these values under $HOME/.roxctl/login ...")
	// TODO: Persisting the login information will be done in a follow-up PR.
	l.loginSignal.Signal()
}
