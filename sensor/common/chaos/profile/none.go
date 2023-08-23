package profile

import toxiproxy "github.com/Shopify/toxiproxy/v2/client"

type none struct{}

// Run starts the chaos proxy controller
func (*none) Run(_ *toxiproxy.Proxy) {}
