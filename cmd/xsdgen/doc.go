/*
xsdgen is a tool to automatically generate Go type declarations and
associated methods based on one or more XML Schema.

Usage:

	xsdgen [-o file] [-ns xmlns] [-pkg name] [-r rule] file ...

Given a set of XML files containing <xsd:schema> declarations,
xsdgen will create a new self-contained Go source file containing
a type declaration for each type defined in the schema. The generated
source file is self-contained and only depends on the Go standard
library.

If the -ns flag is used, only types defined in schema with the specified
target namespace will be declared in the Go source. The -ns flag may
be used more than once. If -ns is not specified, types for all schema in
all files will be declared.

The default package name and output file are "ws" and "xsdgen_output.go",
and can be overriddent by the -pkg and -o flags, respectively. The xsdgen
command will try to fetch any schema dependencies before parsing.

The -r flag can be used to specify a series of replacement rules. A replacement
rule is a string of the form

	regex -> replacement

For example, the rule

	Array_Of_soapenc_(.*) -> ${1}Array

will transform the identifier Array_Of_soapenc_boolean to booleanArray.
All identifiers are passed through the defined substitution rules.

The xsdgen command may be used with the go generate command. Simply
embed a comment in your go source like so:

	//go:generate xsdgen -ns "http://example.net/ws/" schemafile.xml
*/
package main