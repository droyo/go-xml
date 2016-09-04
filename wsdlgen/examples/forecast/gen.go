package forecast

//go:generate -command wsdlgen go run _gen/wsdlgen.go
//go:generate wsdlgen -pkg forecast -c "Package forecast access the National Digital Forecast Database." -r "Type$DOLLAR -> " ndfd.wsdl
