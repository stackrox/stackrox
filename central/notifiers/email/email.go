package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/mitchellh/go-wordwrap"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers/metadatagetter"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/administration/events/option"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	mitreDS "github.com/stackrox/rox/pkg/mitre/datastore"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule(option.EnableAdministrationEvents())
)

const (
	connectTimeout  = 5 * time.Second
	emailLineLength = 78
)

// email notifier plugin.
type email struct {
	config      *storage.Email
	creds       string
	cryptoKey   string
	cryptoCodec cryptocodec.CryptoCodec
	smtpServer  smtpServer

	metadataGetter notifiers.MetadataGetter
	mitreStore     mitreDS.AttackReadOnlyDataStore

	notifier *storage.Notifier
}

type plainAuthUnencrypted struct {
	identity, username, password string
	host                         string
}

func unencryptedPlainAuth(identity, username, password, host string) smtp.Auth {
	return &plainAuthUnencrypted{
		identity: identity,
		username: username,
		password: password,
		host:     host,
	}
}

func (a *plainAuthUnencrypted) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	// This is modified from smtp.plainAuth.Start()
	// to remove the check that passwords can only be sent unencrypted
	// to localhost.
	// As long as we claim to support unencrypted SMTP we need to
	// override this check, since the user is explicitly telling us
	// to do the bad idea.
	resp := []byte(a.identity + "\x00" + a.username + "\x00" + a.password)
	return "PLAIN", resp, nil
}

func (a *plainAuthUnencrypted) Next(_ []byte, more bool) ([]byte, error) {
	// This is copied from smtp.plainAuth.Next().
	// See Start() for reasons why we have copied this type.
	if more {
		// We've already sent everything.
		return nil, errors.New("unexpected server challenge")
	}
	return nil, nil
}

type loginAuth struct {
	username, password string
}

func loginAuthMethod(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		serverStr := strings.ToLower(string(fromServer))
		switch serverStr {
		case "username:":
			return []byte(a.username), nil
		case "password:":
			return []byte(a.password), nil
		default:
			return nil, fmt.Errorf("unknown value request %q from server", serverStr)
		}
	}
	return nil, nil
}

type smtpServer struct {
	host string
	port int
}

func (s *smtpServer) endpoint() string {
	return fmt.Sprintf("%v:%v", s.host, s.port)
}

func validate(notifier *storage.Notifier) error {
	if env.EncNotifierCreds.BooleanSetting() {
		if notifier.GetNotifierSecret() == "" {
			return errors.New("Notifier secret must be non-empty")
		}
	}
	emailConf := notifier.GetEmail()
	if emailConf == nil {
		return errors.New("Email configuration is required")
	}
	errorList := errorhelpers.NewErrorList("Email validation")
	if emailConf.GetServer() == "" {
		errorList.AddString("SMTP Server must be specified")
	}
	if emailConf.GetSender() == "" {
		errorList.AddString("Sender must be specified")
	}
	// username and password are optional for unauthenticated smtp
	if !emailConf.AllowUnauthenticatedSmtp {
		if emailConf.GetUsername() == "" {
			errorList.AddString("Username must be specified")
		}
		if emailConf.GetPassword() == "" {
			errorList.AddString("Password must be specified")
		}
	}
	if !emailConf.GetDisableTLS() && emailConf.GetStartTLSAuthMethod() != storage.Email_DISABLED {
		errorList.AddString("TLS must be disabled to use a StartTLS Auth Method")
	}
	return errorList.ToError()
}

// NewEmail exported to allow for usage in various components.
func NewEmail(notifier *storage.Notifier, metadataGetter notifiers.MetadataGetter, mitreStore mitreDS.AttackReadOnlyDataStore,
	cryptoCodec cryptocodec.CryptoCodec, cryptoKey string) (*email, error) {
	if err := validate(notifier); err != nil {
		return nil, err
	}

	conf := notifier.GetEmail()
	port := 465 // default TLS SMTP Port
	server := conf.GetServer()
	host := conf.GetServer()
	idx := strings.Index(server, ":")
	if idx != -1 && idx != len(server)-1 {
		parsedPort, err := strconv.Atoi(server[idx+1:])
		if err != nil || parsedPort < 0 || parsedPort > 65535 {
			return nil, fmt.Errorf("Port number cannot be '%v' and must be valid port between 0-65535", server[idx+1:])
		}
		port = parsedPort
		host = server[:idx]
	}

	return &email{
		config:      conf,
		creds:       "",
		cryptoKey:   cryptoKey,
		cryptoCodec: cryptoCodec,
		smtpServer: smtpServer{
			host: host,
			port: port,
		},
		notifier:       notifier,
		metadataGetter: metadataGetter,
		mitreStore:     mitreStore,
	}, nil
}

type message struct {
	To          []string
	From        string
	Subject     string
	Body        string
	Attachments map[string][]byte
	EmbedLogo   bool
}

// This function does not support UTF-8 strings.
func applyRfc5322LineLengthLimit(str string) string {
	strLen := len(str)

	startPos := 0
	numOfChunks := strLen / emailLineLength

	var builder strings.Builder
	for numOfChunks > 0 && startPos+emailLineLength < strLen {
		builder.WriteString(str[startPos : startPos+emailLineLength])
		builder.WriteString("\r\n")

		numOfChunks--
		startPos += emailLineLength
	}
	builder.WriteString(str[startPos:strLen])

	return builder.String()
}

func applyRfc5322TextWordWrap(text string) string {
	wrappedText := wordwrap.WrapString(text, emailLineLength)

	// In case when text is formatted with \r\n and additionally wrapped,
	// we have a combination of \n and \r\n. First, we must normalize the text.
	// Otherwise, we will have wrong formatting if we replace \n with \r\n.
	// If not normalized, we can get results with double \r. i.e. \r\r\n
	wrappedText = strings.Replace(wrappedText, "\r\n", "\n", -1)
	wrappedText = strings.Replace(wrappedText, "\n", "\r\n", -1)

	return wrappedText
}

func (m message) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(fmt.Sprintf("From: %s\r\n", m.From))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(m.To, ",")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", m.Subject))

	buf.WriteString("MIME-Version: 1.0\r\n")

	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()

	if m.EmbedLogo {
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		buf.WriteString(fmt.Sprintf("\n--%s\r\n", boundary))

		buf.WriteString("Content-Type: image/png; name=logo.png\r\n")
		buf.WriteString("Content-Transfer-Encoding: base64\r\n")
		buf.WriteString("Content-Disposition: inline; filename=logo.png\r\n")
		buf.WriteString("Content-ID: <logo.png>\r\n")
		buf.WriteString("X-Attachment-Id: logo.png\r\n")
		buf.WriteString(fmt.Sprintf("\r\n%s\r\n", applyRfc5322LineLengthLimit(branding.GetLogoBase64())))
		buf.WriteString(fmt.Sprintf("\n--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n\r\n")
		buf.WriteString("<img src=\"cid:logo.png\" width=\"20%\" height=\"20%\"><br><br><div>\r\n")
		buf.WriteString(fmt.Sprintf("%s\r\n", applyRfc5322TextWordWrap(m.Body)))
		buf.WriteString("</div>\r\n")
	} else {
		buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
		buf.WriteString(fmt.Sprintf("%s\r\n", applyRfc5322TextWordWrap(m.Body)))
	}

	for k, v := range m.Attachments {
		buf.WriteString(fmt.Sprintf("\n--%s\r\n", boundary))
		buf.WriteString("Content-Type: application/zip\r\n")
		buf.WriteString("Content-Transfer-Encoding: base64\r\n")
		buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%s\r\n", k))
		buf.WriteString(fmt.Sprintf("\r\n%s\r\n", applyRfc5322LineLengthLimit(base64.StdEncoding.EncodeToString(v))))
		buf.WriteString(fmt.Sprintf("\n--%s\r\n", boundary))
	}
	return buf.Bytes()
}

func (e *email) plainTextAlert(alert *storage.Alert) (string, error) {
	funcMap := template.FuncMap{
		"header": func(s string) string {
			return fmt.Sprintf("\r\n%s\r\n", s)
		},
		"subheader": func(s string) string {
			return fmt.Sprintf("\r\n\t%s\r\n", s)
		},
		"line": func(s string) string {
			return fmt.Sprintf("%s\r\n", s)
		},
		"list": func(s string) string {
			return fmt.Sprintf("\t - %s\r\n", s)
		},
		"nestedList": func(s string) string {
			return fmt.Sprintf("\t\t - %s\r\n", s)
		},
		"codeBlock": func(s string) string {
			return fmt.Sprintf("\n %s \n", s)
		},
		"section": func(s string) string {
			return fmt.Sprintf("\r\n\t\t%s\r\n", s)
		},
		"group": func(s string) string {
			return fmt.Sprintf("\r\n\t\t\t - %s", s)
		},
	}
	alertLink := notifiers.AlertLink(e.notifier.UiEndpoint, alert)
	return notifiers.FormatAlert(alert, alertLink, funcMap, e.mitreStore)
}

func (*email) Close(context.Context) error {
	return nil
}

// AlertNotify takes in an alert and generates the email.
func (e *email) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	subject := notifiers.SummaryForAlert(alert)
	body, err := e.plainTextAlert(alert)
	if err != nil {
		return err
	}

	recipient := e.metadataGetter.GetAnnotationValue(ctx, alert, e.notifier.GetLabelKey(), e.notifier.GetLabelDefault())
	return e.sendEmail(ctx, recipient, subject, body)
}

// ReportNotify takes in reporting data, a list of intended recipients, email subject and an email message to send out a report.
// Set subject to empty string for v1 report configs.
func (e *email) ReportNotify(ctx context.Context, zippedReportData *bytes.Buffer, recipients []string, subject, messageText string) error {
	var from string
	if e.config.GetFrom() != "" {
		from = fmt.Sprintf("%s <%s>", e.config.GetFrom(), e.config.GetSender())
	} else {
		from = e.config.GetSender()
	}
	if subject == "" {
		subject = fmt.Sprintf("%s Image Vulnerability Report for %s", branding.GetProductNameShort(), time.Now().Format("02-January-2006"))
	}
	msg := message{
		To:        recipients,
		From:      from,
		Subject:   subject,
		Body:      messageText,
		EmbedLogo: true,
	}

	if zippedReportData != nil {
		msg.Attachments = map[string][]byte{
			fmt.Sprintf("%s_Vulnerability_Report_%s.zip", branding.GetProductNameShort(), time.Now().Format("02_January_2006")): zippedReportData.Bytes(),
		}
	}
	return e.send(ctx, &msg)
}

// NetworkPolicyYAMLNotify takes in a yaml file and generates the email message.
func (e *email) NetworkPolicyYAMLNotify(ctx context.Context, yaml string, clusterName string) error {
	subject := fmt.Sprintf("New network policy YAML for cluster '%s' needs to be applied", clusterName)

	body, err := notifiers.FormatNetworkPolicyYAML(yaml, clusterName, template.FuncMap{
		"codeBlock": func(s string) string {
			return s
		},
	})
	if err != nil {
		return err
	}
	return e.sendEmail(ctx, e.notifier.GetLabelDefault(), subject, body)
}

// Test sends a test notification.
func (e *email) Test(ctx context.Context) error {
	subject := "StackRox Test Email"
	body := fmt.Sprintf("%v\r\n", "This is a test email created to test integration with StackRox.")
	err := e.sendEmail(ctx, e.notifier.GetLabelDefault(), subject, body)
	return err
}

func (e *email) sendEmail(ctx context.Context, recipient, subject, body string) error {
	var from string
	if e.config.GetFrom() != "" {
		from = fmt.Sprintf("%s <%s>", e.config.GetFrom(), e.config.GetSender())
	} else {
		from = e.config.GetSender()
	}

	msg := message{
		To:        []string{recipient},
		From:      from,
		Subject:   subject,
		Body:      body,
		EmbedLogo: false,
	}
	return e.send(ctx, &msg)
}

func (e *email) send(ctx context.Context, m *message) error {
	conn, auth, err := e.connection(ctx)
	if err != nil {
		return createError("Connection failed", err, e.notifier.GetName())
	}

	client, err := e.createClient(conn)
	if err != nil {
		return createError("SMTP client creation failed", err, e.notifier.GetName())
	}
	defer func() {
		if err := client.Quit(); err != nil {
			log.Error("Failed to quit client cleanly", logging.Err(err))
		}
	}()

	if e.config.GetStartTLSAuthMethod() != storage.Email_DISABLED {
		if err = client.StartTLS(e.tlsConfig()); err != nil {
			return createError("SMTP STARTTLS failed", err, e.notifier.GetName())
		}
	}

	if !e.notifier.GetEmail().GetAllowUnauthenticatedSmtp() {
		if err = client.Auth(auth); err != nil {
			return createError("SMTP authentication failed", err, e.notifier.GetName())
		}
	}

	if err = client.Mail(e.config.GetSender()); err != nil {
		return createError("SMTP MAIL command failed", err, e.notifier.GetName())
	}
	for _, toAddr := range m.To {
		if err = client.Rcpt(toAddr); err != nil {
			return createError("SMTP RCPT command failed", err, e.notifier.GetName())
		}
	}

	w, err := client.Data()
	if err != nil {
		return createError("SMTP DATA command failed", err, e.notifier.GetName())
	}
	defer utils.IgnoreError(w.Close)

	_, err = w.Write(m.Bytes())
	if err != nil {
		return createError("SMTP message writing failed", err, e.notifier.GetName())
	}

	return nil
}

// createClient creates an SMTP client but bails out in cases where
// smtp.NewClient would otherwise hang.
// The known case (ROX-366) is when dialing a TLS server with a non-TLS dialer;
// in this case the first dial will succeed, but then the net/textproto reader
// will hang for a few minutes.
func (e *email) createClient(conn net.Conn) (c *smtp.Client, err error) {
	var timedOut concurrency.Flag
	// If the timer expires before we return and thereby stop it,
	// we'll close the connection and thereby cause the Client creation
	// to abort immediately instead of waiting for minutes for an EOF.
	//
	// There's a possibility that we have _just_ succeeded returning from
	// NewClient when this timer fires; in this case the subsequent use of
	// the client will fail with an error about using a closed connection.
	// This particular failure mode seems sufficiently unlikely.
	// Importantly, a net.Conn can have multiple clients safely call methods
	// on it at the same time, including Close().
	t := time.AfterFunc(connectTimeout, func() {
		timedOut.Toggle()
		defer utils.IgnoreError(conn.Close)
	})
	defer func() {
		t.Stop()
		if timedOut.Get() {
			err = errors.New("timeout: possibly speaking unencrypted to a server running TLS")
		}
	}()

	return smtp.NewClient(conn, e.smtpServer.host)
}

func (e *email) connection(ctx context.Context) (conn net.Conn, auth smtp.Auth, err error) {
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if e.config.GetDisableTLS() {
		if e.config.GetStartTLSAuthMethod() != storage.Email_DISABLED {
			return e.startTLSConn(ctx)
		}
		return e.unencryptedConn(ctx)
	}
	return e.tlsConn(ctx)
}

func (e *email) tlsConn(dialCtx context.Context) (conn net.Conn, auth smtp.Auth, err error) {
	password, err := e.getPassword()
	if err != nil {
		return nil, nil, err
	}
	// With a connection that starts with TLS, we can simply use the standard
	// library to authenticate.
	auth = smtp.PlainAuth("", e.config.GetUsername(), password, e.smtpServer.host)
	conn, err = proxy.AwareDialContextTLS(dialCtx, e.smtpServer.endpoint(), e.tlsConfig())
	return
}

func (e *email) unencryptedConn(dialCtx context.Context) (conn net.Conn, auth smtp.Auth, err error) {
	password, err := e.getPassword()
	if err != nil {
		return nil, nil, err
	}
	// With a completely unencrypted connection, we must override the
	// standard library's SMTP authenticator, since it blocks attempts
	// to send credentials over any non-TLS connection that isn't localhost.
	auth = unencryptedPlainAuth("", e.config.GetUsername(), password, e.smtpServer.host)
	conn, err = proxy.AwareDialContext(dialCtx, e.smtpServer.endpoint())
	return
}

func (e *email) startTLSConn(dialCtx context.Context) (conn net.Conn, auth smtp.Auth, err error) {
	password, err := e.getPassword()
	if err != nil {
		return nil, nil, err
	}
	// With STARTTLS, we will first connect unencrypted and later
	// "upgrade" the connection to use TLS by the time we authenticate.
	// Hence, we can use the stdlib authenticator, which treats
	// STARTTLS as TLS for purposes of deciding whether it's safe to
	// transmit a password.
	switch e.notifier.GetEmail().GetStartTLSAuthMethod() {
	case storage.Email_PLAIN:
		auth = smtp.PlainAuth("", e.config.GetUsername(), password, e.smtpServer.host)
	case storage.Email_LOGIN:
		auth = loginAuthMethod(e.config.GetUsername(), password)
	}
	conn, err = proxy.AwareDialContext(dialCtx, e.smtpServer.endpoint())
	return
}

func (e *email) tlsConfig() *tls.Config {
	return &tls.Config{
		ServerName: e.smtpServer.host,
	}
}

func (e *email) getPassword() (string, error) {
	if e.creds != "" {
		return e.creds, nil
	}

	if !env.EncNotifierCreds.BooleanSetting() {
		e.creds = e.config.GetPassword()
		return e.creds, nil
	}

	decCreds, err := e.cryptoCodec.Decrypt(e.cryptoKey, e.notifier.GetNotifierSecret())
	if err != nil {
		return "", errors.Errorf("Error decrypting notifier secret for notifier '%s'", e.notifier.GetName())
	}
	e.creds = decCreds
	return e.creds, nil
}

func (e *email) ProtoNotifier() *storage.Notifier {
	return e.notifier
}

func createError(msg string, err error, notifierName string) error {
	if e, _ := err.(*textproto.Error); e != nil {
		msg = fmt.Sprintf("%s (code: %d)", msg, e.Code)
	}
	log.Errorw(msg, logging.Err(err), logging.ErrCode(codes.EmailGeneric),
		logging.NotifierName(notifierName))
	return errors.New(msg)
}

func init() {
	cryptoKey := ""
	var err error
	if env.EncNotifierCreds.BooleanSetting() {
		cryptoKey, err = notifierUtils.GetNotifierSecretEncryptionKey()
		if err != nil {
			utils.CrashOnError(err)
		}
	}
	notifiers.Add(notifiers.EmailType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		e, err := NewEmail(notifier, metadatagetter.Singleton(), mitreDS.Singleton(), cryptocodec.Singleton(), cryptoKey)
		return e, err
	})
}
