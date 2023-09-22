module github.com/g41797/syslogsidecar

go 1.19

require (
	github.com/RackSec/srslog v0.0.0-20180709174129-a4725f04ec91
	github.com/g41797/kissngoqueue v0.1.5
	github.com/g41797/sputnik v0.0.12
	gopkg.in/mcuadros/go-syslog.v2 v2.3.0
)

require (
	github.com/g41797/gonfig v1.0.1 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

//replace github.com/g41797/sputnik => ../../g41797/sputnik/
//replace github.com/g41797/sputnik/sidecar => ../../g41797/sputnik/sidecar
