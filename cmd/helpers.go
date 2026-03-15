package cmd

import (
	"fmt"
	"strings"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// parseExtensionIDs парсит строковые аргументы в слайс ExtensionID
func parseExtensionIDs(args []string) ([]domain.ExtensionID, error) {
	ids := make([]domain.ExtensionID, len(args))
	for i, arg := range args {
		id, err := domain.ParseExtensionID(arg)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}

func parseInstallTargets(args []string) ([]domain.InstallTarget, error) {
	installTargets := make([]domain.InstallTarget, len(args))
	for i, arg := range args {
		split := strings.Split(arg, "@")
		if len(split) > 2 {
			return nil, fmt.Errorf("parse install target: invalid format %q", arg)
		}

		id, err := domain.ParseExtensionID(split[0])
		if err != nil {
			return nil, err
		}
		var version *domain.Version

		if len(split) == 2 {
			parseVer, err := domain.ParseVersion(split[1])
			if err != nil {
				return nil, err
			}
			version = &parseVer
		}

		installTargets[i] = domain.InstallTarget{ID: id, Version: version}
	}
	return installTargets, nil
}
