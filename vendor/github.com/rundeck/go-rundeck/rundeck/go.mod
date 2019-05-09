module github.com/rundeck/go-rundeck/rundeck

go 1.12

require (
	contrib.go.opencensus.io/exporter/ocagent v0.5.0 // indirect
	github.com/Azure/go-autorest v11.2.1+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
)

replace github.com/rundeck/go-rundeck/rundeck/auth => ./auth
