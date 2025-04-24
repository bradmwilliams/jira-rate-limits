package main

import (
	"fmt"
	jiraclient "github.com/andygrunwald/go-jira"
	jira "github.com/bradmwilliams/jira-playpen/pkg/jira"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
)

type retryableHTTPLogrusWrapper struct {
	log *logrus.Entry
}

// fieldsForContext translates a list of context fields to a
// logrus format; any items that don't conform to our expectations
// are omitted
func (l *retryableHTTPLogrusWrapper) fieldsForContext(context ...interface{}) logrus.Fields {
	fields := logrus.Fields{}
	for i := 0; i < len(context)-1; i += 2 {
		key, ok := context[i].(string)
		if !ok {
			continue
		}
		fields[key] = context[i+1]
	}
	return fields
}

func (l *retryableHTTPLogrusWrapper) Error(msg string, context ...interface{}) {
	l.log.WithFields(l.fieldsForContext(context...)).Error(msg)
}

func (l *retryableHTTPLogrusWrapper) Info(msg string, context ...interface{}) {
	l.log.WithFields(l.fieldsForContext(context...)).Info(msg)
}

func (l *retryableHTTPLogrusWrapper) Debug(msg string, context ...interface{}) {
	l.log.WithFields(l.fieldsForContext(context...)).Debug(msg)
}

func (l *retryableHTTPLogrusWrapper) Warn(msg string, context ...interface{}) {
	l.log.WithFields(l.fieldsForContext(context...)).Warn(msg)
}

type BearerAuthGenerator func() (token string)

type bearerAuthRoundtripper struct {
	generator BearerAuthGenerator
	upstream  http.RoundTripper
}

func (bart *bearerAuthRoundtripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := new(http.Request)
	*req2 = *req
	req2.URL = new(url.URL)
	*req2.URL = *req.URL
	token := bart.generator()
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	logrus.WithField("curl", toCurl(req2)).Trace("Executing http request")
	return bart.upstream.RoundTrip(req2)
}

func toCurl(r *http.Request) string {
	headers := ""
	for key, values := range r.Header {
		for _, value := range values {
			headers += fmt.Sprintf(` -H %q`, fmt.Sprintf("%s: %s", key, maskAuthorizationHeader(key, value)))
		}
	}

	return fmt.Sprintf("curl -k -v -X%s %s '%s'", r.Method, headers, r.URL.String())
}

var knownAuthTypes = sets.New[string]("bearer", "basic", "negotiate")

func maskAuthorizationHeader(key string, value string) string {
	if !strings.EqualFold(key, "Authorization") {
		return value
	}
	if len(value) == 0 {
		return ""
	}
	var authType string
	if i := strings.Index(value, " "); i > 0 {
		authType = value[0:i]
	} else {
		authType = value
	}
	if !knownAuthTypes.Has(strings.ToLower(authType)) {
		return "<masked>"
	}
	if len(value) > len(authType)+1 {
		value = authType + " <masked>"
	} else {
		value = authType
	}
	return value
}

func getIssue(client *jiraclient.Client, id string) (*jiraclient.Issue, error) {
	issue, response, err := client.Issue.Get(id, &jiraclient.GetQueryOptions{})
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("jira issue not found: %v", err)
		}
		return nil, fmt.Errorf("error getting jira issue: %v", err)
	}
	return issue, nil
}

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}

func main() {
	endpoint := os.Getenv("JIRA_ENDPOINT")
	if endpoint == "" {
		fmt.Println("Environment variable 'JIRA_ENDPOINT' is not set.")
		os.Exit(1)
	}

	apiKey := os.Getenv("JIRA_API_KEY")
	if apiKey == "" {
		fmt.Println("Environment variable 'JIRA_API_KEY' is not set.")
		os.Exit(1)
	}

	rateLimitInfo := &jira.RateLimitInfo{Data: map[string]int{}}

	log := logrus.WithField("client", "jira")

	retryingClient := retryablehttp.NewClient()
	retryingClient.Logger = &retryableHTTPLogrusWrapper{log: log}
	retryingClient.HTTPClient.Transport = &bearerAuthRoundtripper{
		generator: func() (token string) {
			return apiKey
		},
		upstream: retryingClient.HTTPClient.Transport,
	}
	// These are the default values...
	retryingClient.RetryMax = 4
	retryingClient.RetryWaitMin = 1 * time.Second
	retryingClient.RetryWaitMax = 30 * time.Second
	retryingClient.Backoff = rateLimitInfo.JiraBackoff

	jiraClient, err := jiraclient.NewClient(retryingClient.StandardClient(), endpoint)
	if err != nil {
		return
	}

	issue, err := getIssue(jiraClient, "OCPBUGS-36344")
	if err != nil {
		log.WithError(err).Error("Error getting issue")
		return
	}

	log.WithField("issue", issue.ID).Info("Found jira issue")
}
