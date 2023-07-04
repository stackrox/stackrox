package handler

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

func formatEventToSSE(event *storage.Event) string {
	protoBytes, err := proto.Marshal(event)
	utils.Should(err)
	sb := strings.Builder{}
	sb.WriteString("event: events\n")
	sb.WriteString(fmt.Sprintf("data: %s\n", string(protoBytes)))
	return sb.String()
}
