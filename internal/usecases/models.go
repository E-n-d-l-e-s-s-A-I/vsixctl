package usecases

import "github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"

// InstallResult — результат установки расширения
type InstallResult struct {
	ID  domain.ExtensionID
	Err error
}
