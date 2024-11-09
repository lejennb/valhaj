package writer

import "strings"

// BuildResponse(): Assembles the individual sub-responses into a singular response for the writer.
func BuildResponse(pieces []string) []uint8 {
	delimiter := ""
	response := strings.Join(pieces, delimiter) // Join() uses a strings.Builder under the hood
	return []uint8(response)
}
