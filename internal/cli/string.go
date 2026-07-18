package cli

import (
	"github.com/spf13/cobra"
)

// stringCollection is the NSID of tangled string records.
const stringCollection = "sh.tangled.string"

// stringRecord is the value of a sh.tangled.string record. Description may
// be empty.
type stringRecord struct {
	Type        string `json:"$type"`
	Filename    string `json:"filename"`
	Description string `json:"description"`
	Contents    string `json:"contents"`
	CreatedAt   string `json:"createdAt"`
}

var stringCmd = &cobra.Command{
	Use:   "string",
	Short: "Manage strings on Tangled",
}
