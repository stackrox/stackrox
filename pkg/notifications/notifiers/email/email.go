package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/types"
)

var (
	log = logging.New("notifiers/email")
)

// email notifier plugin
type email struct {
	server    string
	sender    string
	recipient string
	username  string
	password  string
	tls       bool

	smtpServer smtpServer

	notifier *v1.Notifier
}

type plainAuthOverTLSConn struct {
	smtp.Auth
}

func tlsPlainAuth(identity, username, password, host string) smtp.Auth {
	return &plainAuthOverTLSConn{smtp.PlainAuth(identity, username, password, host)}
}

func (a *plainAuthOverTLSConn) Start(server *smtp.ServerInfo) (string, []byte, error) {
	server.TLS = true
	return a.Auth.Start(server)
}

type smtpServer struct {
	host string
	port int
}

func (s *smtpServer) endpoint() string {
	return fmt.Sprintf("%v:%v", s.host, s.port)
}

func newEmail(notifier *v1.Notifier) (*email, error) {
	var err error
	server, ok := notifier.Config["server"]
	if !ok {
		return nil, fmt.Errorf("SMTP Server must be defined in the Email Configuration")
	}
	sender, ok := notifier.Config["sender"]
	if !ok {
		return nil, fmt.Errorf("Sender must be defined in the Email Configuration")
	}
	recipient, ok := notifier.Config["recipient"]
	if !ok {
		return nil, fmt.Errorf("Recipient must be defined in the Email Configuration")
	}
	username, ok := notifier.Config["username"]
	if !ok {
		return nil, fmt.Errorf("Username must be defined in the Email Configuration")
	}
	password, ok := notifier.Config["password"]
	if !ok {
		return nil, fmt.Errorf("Password must be defined in the Email Configuration")
	}
	// This parameter is optional
	tlsStr, ok := notifier.Config["tls"]
	tls := true
	if ok {
		tls, err = strconv.ParseBool(tlsStr)
		if err != nil {
			return nil, fmt.Errorf("TLS parameter cannot be '%v' and must be either true/false", tls)
		}
	}

	port := 465 // default TLS SMTP Port
	host := server
	idx := strings.Index(server, ":")
	if idx != -1 && idx != len(server)-1 {
		port, err = strconv.Atoi(server[idx+1:])
		if err != nil || port < 0 || port > 65535 {
			return nil, fmt.Errorf("Port number cannot be '%v' and must be valid port between 0-65535", server[idx+1:])
		}
		host = server[:idx]
	}
	return &email{
		server:    server,
		sender:    sender,
		recipient: recipient,
		username:  username,
		password:  password,
		tls:       tls,
		smtpServer: smtpServer{
			host: host,
			port: port,
		},
		notifier: notifier,
	}, nil
}

type message struct {
	To      string
	Subject string
	Body    string
}

func (m message) Bytes() []byte {
	return []byte(fmt.Sprintf("To: %v\r\nSubject: %v\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%v\r\n", m.To, m.Subject, m.Body))
}

func (e *email) plainTextAlert(alert *v1.Alert) (string, error) {
	funcMap := template.FuncMap{
		"header": func(s string) string {
			return fmt.Sprintf("\r\n%v\r\n", s)
		},
		"subheader": func(s string) string {
			return fmt.Sprintf("\r\n\t%v\r\n", s)
		},
		"line": func(s string) string {
			return fmt.Sprintf("%v\r\n", s)
		},
		"list": func(s string) string {
			return fmt.Sprintf("\t - %v\r\n", s)
		},
		"nestedList": func(s string) string {
			return fmt.Sprintf("\t\t - %v\r\n", s)
		},
	}
	alertLink := notifiers.AlertLink(e.notifier.UiEndpoint)
	return notifiers.FormatPolicy(alert, alertLink, funcMap)
}

func (e *email) plainTextBenchmark(schedule *v1.BenchmarkSchedule) (string, error) {
	benchmarkLink := notifiers.BenchmarkLink(e.notifier.UiEndpoint)
	return notifiers.FormatBenchmark(schedule, benchmarkLink)
}

// AlertNotify takes in an alert and generates the email
func (e *email) AlertNotify(alert *v1.Alert) error {
	subject := fmt.Sprintf("Deployment %v (%v) violates '%v' Policy", alert.GetDeployment().GetName(),
		alert.GetDeployment().GetId(), alert.GetPolicy().GetName())
	body, err := e.plainTextAlert(alert)
	if err != nil {
		return err
	}
	return e.sendEmail(subject, body)
}

// BenchmarkNotify takes in an benchmark and generates the email
func (e *email) BenchmarkNotify(schedule *v1.BenchmarkSchedule) error {
	subject := fmt.Sprintf("New Benchmark Results for %v", schedule.GetName())
	body, err := e.plainTextBenchmark(schedule)
	if err != nil {
		return err
	}
	return e.sendEmail(subject, body)
}

// Test sends a test notification
func (e *email) Test() error {
	subject := "StackRox Test Email"
	body := fmt.Sprintf("%v\r\n", "This is a test email created to test integration with StackRox.")
	err := e.sendEmail(subject, body)
	return err
}

func (e *email) sendEmail(subject, body string) error {
	msg := message{
		To:      e.recipient,
		Subject: subject,
		Body:    body,
	}

	var err error
	var conn net.Conn

	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	var auth smtp.Auth
	if e.tls {
		tlsconfig := &tls.Config{
			ServerName: e.smtpServer.host,
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", e.smtpServer.endpoint(), tlsconfig)
		if err != nil {
			log.Error(err)
			return err
		}
		auth = tlsPlainAuth("", e.sender, e.password, e.smtpServer.host)
	} else {
		conn, err = dialer.Dial("tcp", e.smtpServer.endpoint())
		if err != nil {
			log.Error(err)
			return err
		}
		auth = smtp.PlainAuth("", e.sender, e.password, e.smtpServer.host)
	}
	client, err := smtp.NewClient(conn, e.smtpServer.host)
	if err != nil {
		log.Error(err)
		return err
	}
	defer client.Quit()
	if err = client.Auth(auth); err != nil {
		log.Error(err)
		return err
	}

	if err = client.Mail(e.sender); err != nil {
		log.Error(err)
		return err
	}
	if err = client.Rcpt(e.recipient); err != nil {
		log.Error(err)
		return err
	}

	w, err := client.Data()
	if err != nil {
		log.Error(err)
		return err
	}
	defer w.Close()

	_, err = w.Write(msg.Bytes())
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func (e *email) ProtoNotifier() *v1.Notifier {
	return e.notifier
}

func init() {
	notifiers.Add("email", func(notifier *v1.Notifier) (types.Notifier, error) {
		e, err := newEmail(notifier)
		return e, err
	})
}
