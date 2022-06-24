package generate

import "fmt"

func (cmd *netpolGenerateCommand) generateNetpol() error {
	fmt.Printf("Got path %s", cmd.folderPath)
	return nil
}
