package main

import (
	"fmt"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

func main() {
	extension := domain.Extension{
		ID:      domain.ExtensionID{},
		Version: "1.31.3",
	}
	fmt.Println(extension)
}
