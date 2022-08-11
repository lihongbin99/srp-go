package config

import "io/ioutil"

var (
	SecurityPrivateKey []byte
	SecurityPublicKey  []byte
)

func flushSecurity(_ *Config, newConfig *Config) {
	if len(newConfig.Common.Security.PrivateKey) > 0 {
		if SecurityPrivateKeyTemp, err := ioutil.ReadFile(newConfig.Common.Security.PrivateKey); err != nil {
			log.Error("read private key pem error", err)
		} else {
			SecurityPrivateKey = SecurityPrivateKeyTemp
		}
	} else {
		SecurityPrivateKey = nil
	}

	if len(newConfig.Common.Security.PublicKey) > 0 {
		if SecurityPublicKeyTemp, err := ioutil.ReadFile(newConfig.Common.Security.PublicKey); err != nil {
			log.Error("read public key pem error", err)
		} else {
			SecurityPublicKey = SecurityPublicKeyTemp
		}
	} else {
		SecurityPublicKey = nil
	}
	log.Info("flushConfig")
}
