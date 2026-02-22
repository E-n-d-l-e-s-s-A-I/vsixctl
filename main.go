package main

import (
	"net/http"
	"os"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/cmd"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/registry/marketplace"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/storage/vscode"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/ui/cli"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/usecases"
)

func main() {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxConnsPerHost:     10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 5 * time.Second,
		},
		Timeout: 10 * time.Second,
	}
	registry := marketplace.NewRegistry("https://marketplace.visualstudio.com/_apis/public/gallery", client)
	storage := vscode.NewVSCodeStorage("")
	userCase := usecases.NewUserCaseService(registry, storage)
	app := &cmd.App{
		UseCase:   userCase,
		Presenter: cli.NewPresenter(os.Stdout),
	}

	if err := cmd.NewRootCmd(app).Execute(); err != nil {
		os.Exit(1)
	}
}
