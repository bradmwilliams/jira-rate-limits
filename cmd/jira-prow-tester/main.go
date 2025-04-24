package main

import (
	"flag"
	"fmt"
	"github.com/bradmwilliams/jira-playpen/pkg/jira"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"os"
	"sigs.k8s.io/prow/pkg/flagutil"
	prowJira "sigs.k8s.io/prow/pkg/jira"
)

type options struct {
	jira flagutil.JiraOptions
}

func (o *options) Run() error {
	logrus.Info("Starting jira prow tester...")

	rateLimitInfo := &jira.RateLimitInfo{Data: map[string]int{}}
	o.jira.CustomBackoff(rateLimitInfo.JiraBackoff)

	client, err := o.jira.Client()
	if err != nil {
		return fmt.Errorf("failed to create jira client: %v", err)
	}

	issueID := "OCPBUGS-36344"

	extBugs, err := client.GetRemoteLinks(issueID)
	if err != nil {
		return fmt.Errorf("failed to get remote links: %v", err)
	}
	logrus.Infof("Found %d remote links", len(extBugs))

	issue, err := client.GetIssue(issueID)
	if prowJira.JiraErrorStatusCode(err) == 403 {
		logrus.Warningf("Permissions error getting issue %s; ignoring", issueID)
		return nil
	}
	if prowJira.JiraErrorStatusCode(err) == 404 {
		logrus.Warningf("Invalid jira issue %s; ignoring", issueID)
		return nil
	}
	if err != nil {
		logrus.Errorf("Unexpected error occured: %v", err)
		return err
	}
	logrus.WithFields(logrus.Fields{"id": issue.ID}).Info("Successfully retrieved issue")
	return nil
}

func init() {
	// Log as JSON instead of the default ASCII formatter.
	logrus.SetFormatter(&logrus.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logrus.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	logrus.SetLevel(logrus.DebugLevel)
}

func main() {
	opt := &options{}

	cmd := &cobra.Command{
		Run: func(cmd *cobra.Command, arguments []string) {
			if err := opt.Run(); err != nil {
				klog.Exitf("error: %v", err)
			}
		},
	}
	flagSet := cmd.Flags()

	goFlagSet := flag.NewFlagSet("prowflags", flag.ContinueOnError)
	opt.jira.AddFlags(goFlagSet)
	flagSet.AddGoFlagSet(goFlagSet)

	if err := cmd.Execute(); err != nil {
		klog.Exitf("error: %v", err)
	}
}
