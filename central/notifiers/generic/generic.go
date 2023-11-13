package generic

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	alertMessageKey         = "alert"
	auditMessageKey         = "audit"
	networkPolicyMessageKey = "networkpolicy"

	// serviceOperatorCAPath points to the secret of the service account, which within an OpenShift environment
	// also has the service-ca.crt, which includes the CA to verify certificates issued by the service-ca operator.
	// This could be i.e. the default ingress controller certificate.
	serviceOperatorCAPath = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
)

var (
	timeout = env.WebhookTimeout.DurationSetting()
)

// generic notifier plugin.
type generic struct {
	*storage.Notifier

	client      *http.Client
	creds       string
	cryptoKey   string
	cryptoCodec cryptocodec.CryptoCodec

	fullyQualifiedEndpoint string
	extraFieldsJSONPrefix  string
}

func (*generic) Close(_ context.Context) error {
	return nil
}

// AlertNotify takes in an alert and generates the Slack message.
func (g *generic) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	return g.postMessageWithRetry(ctx, alert, alertMessageKey)
}

// NetworkPolicyYAMLNotify takes in a yaml file and generates the Slack message.
func (g *generic) NetworkPolicyYAMLNotify(ctx context.Context, yaml string, clusterName string) error {
	msg := &v1.NetworkPolicyNotification{
		Cluster: clusterName,
		Yaml:    yaml,
	}
	return g.postMessageWithRetry(ctx, msg, networkPolicyMessageKey)
}

// Validate Generic notifier
func Validate(generic *storage.Generic, validateSecret bool) error {
	errList := errorhelpers.NewErrorList("Generic webhook validation")
	if generic.GetEndpoint() == "" {
		errList.AddString("endpoint is required")
	}
	if validateSecret && generic.GetUsername() != generic.GetPassword() && stringutils.AtLeastOneEmpty(generic.GetUsername(), generic.GetPassword()) {
		errList.AddString("both username and password must be defined together")
	}
	for _, f := range generic.GetHeaders() {
		if f.GetKey() == "" || f.GetValue() == "" {
			errList.AddString("all headers must have both a key and a value")
		}
	}
	for _, f := range generic.GetExtraFields() {
		if f.GetKey() == "" || f.GetValue() == "" {
			errList.AddString("all extra fields must have both a key and a value")
		}
	}
	return errList.ToError()
}

func getExtraFieldJSON(fields []*storage.KeyValuePair) (string, error) {
	fieldMap := make(map[string]string)
	for _, f := range fields {
		fieldMap[f.Key] = f.Value
	}
	data, err := json.Marshal(fieldMap)
	if err != nil {
		return "", err
	}

	// Cut off trailing '}'
	data = data[:len(data)-1]
	return string(data), nil
}

func newGeneric(notifier *storage.Notifier, cryptoCodec cryptocodec.CryptoCodec, cryptoKey string) (*generic, error) {
	conf := notifier.GetGeneric()
	if err := Validate(conf, !env.EncNotifierCreds.BooleanSetting()); err != nil {
		return nil, err
	}
	fullyQualifiedEndpoint := urlfmt.FormatURL(conf.GetEndpoint(), urlfmt.HTTPS, urlfmt.HonorInputSlash)

	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = x509.NewCertPool()
	}
	if conf.GetCaCert() != "" {
		if ok := rootCAs.AppendCertsFromPEM([]byte(conf.GetCaCert())); !ok {
			return nil, errors.New("could not add CA Cert passed in configuration")
		}
	}
	// Trust local cluster services.
	if serviceCA, err := os.ReadFile(serviceOperatorCAPath); err == nil {
		rootCAs.AppendCertsFromPEM(serviceCA)
	}
	extraFieldsJSON, err := getExtraFieldJSON(conf.ExtraFields)
	if err != nil {
		return nil, err
	}

	return &generic{
		Notifier: notifier,

		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: conf.GetSkipTLSVerify(),
					RootCAs:            rootCAs,
				},
				Proxy: proxy.FromConfig(),
			},
		},
		creds:                  "",
		cryptoKey:              cryptoKey,
		cryptoCodec:            cryptoCodec,
		fullyQualifiedEndpoint: fullyQualifiedEndpoint,
		extraFieldsJSONPrefix:  extraFieldsJSON,
	}, nil
}

func (g *generic) ProtoNotifier() *storage.Notifier {
	return g.Notifier
}

func (g *generic) Test(ctx context.Context) error {
	alert := &storage.Alert{
		Id: "testalert",
		Policy: &storage.Policy{
			Name: "This is a test message created to test integration with StackRox.",
		},
	}
	return g.AlertNotify(ctx, alert)
}

func (g *generic) constructJSON(message proto.Message, msgKey string) (io.Reader, error) {
	msgStr, err := new(jsonpb.Marshaler).MarshalToString(message)
	if err != nil {
		return nil, err
	}

	var strJSON string
	// No extra fields append so that the payload is something like {"alert": {...}}
	if len(g.Notifier.GetGeneric().GetExtraFields()) == 0 {
		strJSON = fmt.Sprintf(`{"%s": %s}`, msgKey, msgStr)
	} else {
		strJSON = fmt.Sprintf(`%s,"%s": %s}`, g.extraFieldsJSONPrefix, msgKey, msgStr)
	}
	return bytes.NewBufferString(strJSON), nil
}

func (g *generic) postMessage(ctx context.Context, message proto.Message, msgKey string) error {
	password, err := g.getPassword()
	if err != nil {
		return err
	}
	body, err := g.constructJSON(message, msgKey)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.fullyQualifiedEndpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for _, h := range g.GetGeneric().GetHeaders() {
		req.Header.Add(h.GetKey(), h.GetValue())
	}

	if g.GetGeneric().GetUsername() != "" {
		req.SetBasicAuth(g.GetGeneric().GetUsername(), password)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}

	return notifiers.CreateError(g.GetName(), resp, codes.WebhookGeneric)
}

func (g *generic) postMessageWithRetry(ctx context.Context, message proto.Message, msgKey string) error {
	return retry.WithRetry(
		func() error {
			return g.postMessage(ctx, message, msgKey)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

func (g *generic) SendAuditMessage(ctx context.Context, msg *v1.Audit_Message) error {
	if !g.AuditLoggingEnabled() {
		return nil
	}
	return g.postMessageWithRetry(ctx, msg, auditMessageKey)
}

func (g *generic) AuditLoggingEnabled() bool {
	return g.GetGeneric().GetAuditLoggingEnabled()
}

func (g *generic) getPassword() (string, error) {
	if g.GetGeneric().GetUsername() == "" {
		// Both username and password can be empty for unauthenticated generic notifier integration
		return "", nil
	}

	if g.creds != "" {
		return g.creds, nil
	}

	if !env.EncNotifierCreds.BooleanSetting() {
		g.creds = g.GetGeneric().GetPassword()
		return g.creds, nil
	}

	if g.GetNotifierSecret() == "" {
		return "", errors.Errorf("encrypted notifier credentials for notifier '%s' empty", g.GetName())
	}

	decCreds, err := g.cryptoCodec.Decrypt(g.cryptoKey, g.GetNotifierSecret())
	if err != nil {
		return "", errors.Errorf("Error decrypting notifier secret for notifier '%s'", g.GetName())
	}
	g.creds = decCreds
	return g.creds, nil
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
	notifiers.Add(notifiers.GenericType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		g, err := newGeneric(notifier, cryptocodec.Singleton(), cryptoKey)
		return g, err
	})
}
