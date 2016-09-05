package chemspell

//go:generate -command wsdlgen go run _gen/wsdlgen.go
//go:generate wsdlgen -v -pkg chemspell -c "Package chemspell accesses the NLM ChemSpell web service." chemspell.wsdl
