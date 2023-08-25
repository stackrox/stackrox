package profile

import (
	"context"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
)

// Controller defines a chaos proxy controller, which should be used to introduce network conditions.
type Controller interface {
	Run(*toxiproxy.Proxy)
}

// GetConfig returns a chaos proxy controller given their name.
func GetConfig(ctx context.Context, name string) Controller {
	switch name {
	case "periodicdisconnect":
		return newPeriodicDisconnect(ctx, 5*time.Minute, 10*time.Second)
	case "droppedpackets":
		return newDroppedPackets(ctx, 0.01)
	case "none":
		return &none{}
	default:
		log.Warnf("Profile not set. Chaos proxy profile set to 'none'")
		return &none{}
	}

}
