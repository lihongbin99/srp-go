package nat

import "srp-go/common/config"

type clientConfig struct {
	*config.NatConfig
	status string
}
