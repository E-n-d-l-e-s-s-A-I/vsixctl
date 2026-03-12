package cmd

import "github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"

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
