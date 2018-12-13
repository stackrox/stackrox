package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/notifiers/slack"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const (
	slackNotifierName  = `slack test`
	slackTestChannelID = `C740BUV25`

	slackBotID = `B24K7CCN9`

	slackAuthTokenEnvVar = `SLACK_AUTH_TOKEN`
)

var (
	slackAuthToken string
)

func init() {
	slackAuthToken = os.Getenv(slackAuthTokenEnvVar)
}

func TestNotification(t *testing.T) {
	t.Skip("Skipping due to broken auth token. Tracked in AP-515")

	require.NotEmpty(t, slackAuthToken)

	defer teardownTestNotification(t)
	setupTestNotification(t)

	raisedAlert := &storage.Alert{}

	subtests := []struct {
		name string
		test func(t *testing.T, alert *storage.Alert)
	}{
		{
			name: "alerts",
			test: verifyAlertsForLatestTag,
		},
		{
			name: "slack",
			test: verifySlack,
		},
	}

	for _, sub := range subtests {
		t.Run(sub.name, func(t *testing.T) {
			sub.test(t, raisedAlert)
		})
	}
}

func setupTestNotification(t *testing.T) {
	conn, err := grpcConnection()
	require.NoError(t, err)

	setupNotifier(t, conn)
	addNotifierToPolicy(t, conn)
	setupNginxLatestTagDeployment(t)
}

func teardownTestNotification(t *testing.T) {
	conn, err := grpcConnection()
	require.NoError(t, err)

	teardownNginxLatestTagDeployment(t)
	teardownNotifier(t, conn)
	verifyPolicyHasNoNotifier(t, conn)
}

func setupNotifier(t *testing.T, conn *grpc.ClientConn) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	service := v1.NewNotifierServiceClient(conn)

	postResp, err := service.PostNotifier(ctx, notifierConfig)
	require.NoError(t, err)

	notifierConfig.Id = postResp.GetId()
}

func addNotifierToPolicy(t *testing.T, conn *grpc.ClientConn) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	service := v1.NewPolicyServiceClient(conn)
	qb := search.NewQueryBuilder().AddStrings(search.PolicyName, expectedLatestTagPolicy)
	resp, err := service.ListPolicies(ctx, &v1.RawQuery{
		Query: qb.Query(),
	})
	require.NoError(t, err)
	require.Len(t, resp.GetPolicies(), 1)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	p, err := service.GetPolicy(ctx, &v1.ResourceByID{
		Id: resp.GetPolicies()[0].GetId(),
	})
	cancel()

	p.Notifiers = append(p.Notifiers, notifierConfig.GetId())

	_, err = service.PutPolicy(ctx, p)
	require.NoError(t, err)
}

func setupNginxLatestTagDeployment(t *testing.T) {
	cmd := exec.Command(`kubectl`, `run`, nginxDeploymentName, `--image=nginx`)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	waitForDeployment(t, nginxDeploymentName)
}

func teardownNotifier(t *testing.T, conn *grpc.ClientConn) {
	service := v1.NewNotifierServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	_, err := service.DeleteNotifier(ctx, &v1.DeleteNotifierRequest{Id: notifierConfig.GetId(), Force: true})
	cancel()
	require.NoError(t, err)
	notifierConfig.Id = ""
}

func teardownNginxLatestTagDeployment(t *testing.T) {
	cmd := exec.Command(`kubectl`, `delete`, `deployment`, nginxDeploymentName, `--ignore-not-found=true`)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	waitForTermination(t, nginxDeploymentName)
}

func verifyPolicyHasNoNotifier(t *testing.T, conn *grpc.ClientConn) {
	service := v1.NewPolicyServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := service.ListPolicies(ctx, &v1.RawQuery{
		Query: search.NewQueryBuilder().AddStrings(search.PolicyName, expectedLatestTagPolicy).Query(),
	})
	cancel()
	require.NoError(t, err)
	require.Len(t, resp.GetPolicies(), 1)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	p, err := service.GetPolicy(ctx, &v1.ResourceByID{
		Id: resp.GetPolicies()[0].GetId(),
	})
	cancel()

	assert.Empty(t, p.GetNotifiers())
}

func verifyAlertsForLatestTag(t *testing.T, alert *storage.Alert) {
	conn, err := grpcConnection()
	require.NoError(t, err)

	service := v1.NewAlertServiceClient(conn)

	qb := search.NewQueryBuilder().AddStrings(search.DeploymentName, nginxDeploymentName).AddStrings(search.PolicyName, expectedLatestTagPolicy).AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	alerts, err := service.ListAlerts(ctx, &v1.ListAlertsRequest{
		Query: qb.Query(),
	})
	cancel()
	require.NoError(t, err)
	require.Len(t, alerts.GetAlerts(), 1)

	newAlert, err := getAlert(service, alerts.GetAlerts()[0].GetId())
	require.NoError(t, err)

	*alert = *newAlert
}

func verifySlack(t *testing.T, alert *storage.Alert) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()

	oldestMessageTime := alert.GetTime().GetSeconds() - 5

	for {
		select {
		case <-ticker.C:
			messages := getSlackMessages(t, oldestMessageTime)
			if verifySlackMessagesHasAlert(t, messages, alert) {
				return
			}
			if len(messages) > 0 {
				ts, err := strconv.ParseFloat(messages[0].TS, 64)
				if err != nil {
					t.Logf("unable to parse message timestamp: %s", err)
					oldestMessageTime = time.Now().Unix()
				} else {
					oldestMessageTime = int64(ts)
				}
			} else {
				oldestMessageTime = time.Now().Unix()
			}

		case <-timer.C:
			t.Fatalf("unable to retrieve notification from slack for alert id: %s", alert.GetId())
		}
	}
}

func getSlackMessages(t *testing.T, oldest int64) []slackMessageResponse {
	url := fmt.Sprintf(`https://slack.com/api/channels.history?oldest=%d&channel=%s`, oldest, slackTestChannelID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", slackAuthToken))

	c := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := c.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var slackResp slackResponse
	err = json.NewDecoder(resp.Body).Decode(&slackResp)
	require.NoError(t, err)
	require.True(t, slackResp.OK)

	return slackResp.Messages
}

func verifySlackMessagesHasAlert(t *testing.T, messages []slackMessageResponse, alert *storage.Alert) (hasAlert bool) {
	for _, m := range messages {
		if m.Type != `message` || m.Subtype != `bot_message` || m.BotID != slackBotID {
			continue
		}

		for _, a := range m.Attachments {
			if strings.Contains(a.Text, alert.GetId()) {
				verifySlackMessageMatchesAlert(t, a, alert)
				return true
			}
		}
	}

	return
}

func verifySlackMessageMatchesAlert(t *testing.T, attachment slackAttachmentResponse, alert *storage.Alert) {
	assert.Contains(t, attachment.Text, notifierConfig.GetUiEndpoint())
	assert.Contains(t, attachment.Text, alert.GetId())
	assert.Contains(t, attachment.Text, notifiers.SeverityString(alert.GetPolicy().GetSeverity()))
	assert.Contains(t, attachment.Text, alert.GetPolicy().GetDescription())

	assert.Contains(t, attachment.Fallback, notifierConfig.GetUiEndpoint())
	assert.Contains(t, attachment.Fallback, alert.GetId())
	assert.Contains(t, attachment.Fallback, notifiers.SeverityString(alert.GetPolicy().GetSeverity()))
	assert.Contains(t, attachment.Fallback, alert.GetPolicy().GetDescription())

	assert.Contains(t, attachment.Pretext, alert.GetPolicy().GetName())

	assert.Equal(t, strings.TrimPrefix(slack.GetAttachmentColor(alert.GetPolicy().GetSeverity()), "#"), attachment.Color)
}

// slackResponse corresponds to the responses as detailed on https://api.slack.com/methods/channels.history.
// Some fields are omitted.
type slackResponse struct {
	OK       bool `json:"ok"`
	Messages []slackMessageResponse
}

type slackMessageResponse struct {
	Subtype     string
	Type        string
	TS          string `json:"ts"`
	Text        string
	Username    string
	BotID       string `json:"bot_id"`
	Attachments []slackAttachmentResponse
}

type slackAttachmentResponse struct {
	Text     string
	Fallback string
	Pretext  string
	Color    string
}
