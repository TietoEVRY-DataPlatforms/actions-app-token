package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/bitnami-labs/pflagenv"
	"github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
)

// https://docs.github.com/en/actions/learn-github-actions/environment-variables
// https://pkg.go.dev/github.com/bitnami-labs/pflagenv

var ghAppID = pflag.Int64("app_id", 0, "Application ID")
var ghAppPrivKey = pflag.BytesBase64("app_pem", []byte{}, "Application private key")
var ghApiURL = pflag.String("api_url", "", "Github URL")

func init() {
	pflagenv.SetFlagsFromEnv("INPUT", pflag.CommandLine)
}

func newGHClient(transport http.RoundTripper) (*github.Client, error) {
	if isGHES() {
		return github.NewEnterpriseClient(*ghApiURL, "", &http.Client{Transport: transport})
	} else {
		return github.NewClient(&http.Client{Transport: transport}), nil
	}
}

func isGHES() bool {
	return *ghApiURL != "https://api.github.com"
}

func addMask(name string) {
	fmt.Printf("::add-mask::%s\n", name)
}

func setOutput(k string, v string) {
	fmt.Printf("::set-output name=%s::%s\n", k, v)
}

func main() {
	logger, _ := zap.NewProduction()
	zap.ReplaceGlobals(logger)

	pflag.Parse()

	ghApiURLFromEnv := os.Getenv("GITHUB_API_URL")
	if *ghApiURL == "" && ghApiURLFromEnv != "" {
		ghApiURL = &ghApiURLFromEnv
		zap.S().Info("Resolved Github API url from env")
	}

	if *ghApiURL == "" {
		zap.S().Fatal("No Github API url found")
	}

	zap.S().Infof("Github API url: %s", *ghApiURL)

	ctx := context.Background()

	if len(*ghAppPrivKey) == 0 {
		zap.S().Fatal("PEM was not decoded")
	}

	transport, err := ghinstallation.NewAppsTransport(http.DefaultTransport, *ghAppID, *ghAppPrivKey)
	if err != nil {
		zap.S().Fatal(err)
	}

	client, err := newGHClient(transport)
	if err != nil {
		zap.S().Fatal(err)
	}

	if isGHES() {
		transport.BaseURL = *ghApiURL
	}

	ghRepoName := os.Getenv("GITHUB_REPOSITORY")
	repoParts := strings.Split(ghRepoName, "/")
	if len(repoParts) != 2 {
		zap.S().Fatalf("Unable to split repo name %s into parts", ghRepoName)
	}

	installation, _, err := client.Apps.FindRepositoryInstallation(ctx, repoParts[0], repoParts[1])
	if err != nil {
		zap.S().Panic(err)
	}

	installTransport := ghinstallation.NewFromAppsTransport(transport, *installation.ID)
	token, err := installTransport.Token(ctx)
	if err != nil {
		zap.S().Panic(err)
	}

	addMask(token)
	setOutput("app_token", token)
}
