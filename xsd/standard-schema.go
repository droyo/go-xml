package xsd

import "encoding/xml"

var wsdl2003xsd = []byte(`<?xml version="1.0" encoding="UTF-8" ?> 
<!-- 
 
Copyright 2001 - 2005, International Business Machines Corporation and Microsoft Corporation
All Rights Reserved

License for WSDL Schema Files

The Authors grant permission to copy and distribute the WSDL Schema 
Files in any medium without fee or royalty as long as this notice and 
license are distributed with them.  The originals of these files can 
be located at:

http://schemas.xmlsoap.org/wsdl/2003-02-11.xsd

THESE SCHEMA FILES ARE PROVIDED "AS IS," AND THE AUTHORS MAKE NO REPRESENTATIONS 
OR WARRANTIES, EXPRESS OR IMPLIED, REGARDING THESE FILES, INCLUDING, BUT NOT 
LIMITED TO, WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, 
NON-INFRINGEMENT OR TITLE.  THE AUTHORS WILL NOT BE LIABLE FOR ANY DIRECT, 
INDIRECT, SPECIAL, INCIDENTAL OR CONSEQUENTIAL DAMAGES ARISING OUT OF OR 
RELATING TO ANY USE OR DISTRIBUTION OF THESE FILES.

The name and trademarks of the Authors may NOT be used in any manner, 
including advertising or publicity pertaining to these files or any program 
or service that uses these files, written prior permission.  Title to copyright 
in these files will at all times remain with the Authors.

No other rights are granted by implication, estoppel or otherwise.


--> 
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"
           targetNamespace="http://schemas.xmlsoap.org/wsdl/"
           elementFormDefault="qualified" >
   
  <xs:complexType mixed="true" name="tDocumentation" >
    <xs:sequence>
      <xs:any minOccurs="0" maxOccurs="unbounded" processContents="lax" />
    </xs:sequence>
  </xs:complexType>

  <xs:complexType name="tDocumented" >
    <xs:annotation>
      <xs:documentation>
      This type is extended by  component types to allow them to be documented
      </xs:documentation>
    </xs:annotation>
    <xs:sequence>
      <xs:element name="documentation" type="wsdl:tDocumentation" minOccurs="0" />
    </xs:sequence>
  </xs:complexType>
	 
  <xs:complexType name="tExtensibleAttributesDocumented" abstract="true" >
    <xs:complexContent>
      <xs:extension base="wsdl:tDocumented" >
        <xs:annotation>
          <xs:documentation>
          This type is extended by component types to allow attributes from other namespaces to be added.
          </xs:documentation>
        </xs:annotation>
        <xs:anyAttribute namespace="##other" processContents="lax" />    
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>

  <xs:complexType name="tExtensibleDocumented" abstract="true" >
    <xs:complexContent>
      <xs:extension base="wsdl:tDocumented" >
        <xs:annotation>
          <xs:documentation>
          This type is extended by component types to allow elements from other namespaces to be added.
          </xs:documentation>
        </xs:annotation>
        <xs:sequence>
          <xs:any namespace="##other" minOccurs="0" maxOccurs="unbounded" processContents="lax" />
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>

  <xs:element name="definitions" type="wsdl:tDefinitions" >
    <xs:key name="message" >
      <xs:selector xpath="wsdl:message" />
      <xs:field xpath="@name" />
    </xs:key>
    <xs:key name="portType" >
      <xs:selector xpath="wsdl:portType" />
      <xs:field xpath="@name" />
    </xs:key>
    <xs:key name="binding" >
      <xs:selector xpath="wsdl:binding" />
      <xs:field xpath="@name" />
    </xs:key>
    <xs:key name="service" >
      <xs:selector xpath="wsdl:service" />
      <xs:field xpath="@name" />
    </xs:key>
    <xs:key name="import" >
      <xs:selector xpath="wsdl:import" />
      <xs:field xpath="@namespace" />
    </xs:key>
  </xs:element>

  <xs:group name="anyTopLevelOptionalElement" >
    <xs:annotation>
      <xs:documentation>
      Any top level optional element allowed to appear more then once - any child of definitions element except wsdl:types. Any extensibility element is allowed in any place.
      </xs:documentation>
    </xs:annotation>
    <xs:choice>
      <xs:element name="import" type="wsdl:tImport" />
      <xs:element name="types" type="wsdl:tTypes" />                     
      <xs:element name="message"  type="wsdl:tMessage" >
        <xs:unique name="part" >
          <xs:selector xpath="wsdl:part" />
          <xs:field xpath="@name" />
        </xs:unique>
      </xs:element>
      <xs:element name="portType" type="wsdl:tPortType" />
      <xs:element name="binding"  type="wsdl:tBinding" />
      <xs:element name="service"  type="wsdl:tService" >
        <xs:unique name="port" >
          <xs:selector xpath="wsdl:port" />
          <xs:field xpath="@name" />
        </xs:unique>
	  </xs:element>
    </xs:choice>
  </xs:group>

  <xs:complexType name="tDefinitions" >
    <xs:complexContent>
      <xs:extension base="wsdl:tExtensibleDocumented" >
        <xs:sequence>
          <xs:group ref="wsdl:anyTopLevelOptionalElement"  minOccurs="0"   maxOccurs="unbounded" />
        </xs:sequence>
        <xs:attribute name="targetNamespace" type="xs:anyURI" use="optional" />
        <xs:attribute name="name" type="xs:NCName" use="optional" />
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
   
  <xs:complexType name="tImport" >
    <xs:complexContent>
      <xs:extension base="wsdl:tExtensibleAttributesDocumented" >
        <xs:attribute name="namespace" type="xs:anyURI" use="required" />
        <xs:attribute name="location" type="xs:anyURI" use="required" />
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
   
  <xs:complexType name="tTypes" >
    <xs:complexContent>   
      <xs:extension base="wsdl:tExtensibleDocumented" />
    </xs:complexContent>   
  </xs:complexType>
     
  <xs:complexType name="tMessage" >
    <xs:complexContent>   
      <xs:extension base="wsdl:tExtensibleDocumented" >
        <xs:sequence>
          <xs:element name="part" type="wsdl:tPart" minOccurs="0" maxOccurs="unbounded" />
        </xs:sequence>
        <xs:attribute name="name" type="xs:NCName" use="required" />
      </xs:extension>
    </xs:complexContent>   
  </xs:complexType>

  <xs:complexType name="tPart" >
    <xs:complexContent>   
      <xs:extension base="wsdl:tExtensibleAttributesDocumented" >
        <xs:attribute name="name" type="xs:NCName" use="required" />
        <xs:attribute name="element" type="xs:QName" use="optional" />
        <xs:attribute name="type" type="xs:QName" use="optional" />    
      </xs:extension>
    </xs:complexContent>   
  </xs:complexType>

  <xs:complexType name="tPortType" >
    <xs:complexContent>   
      <xs:extension base="wsdl:tExtensibleAttributesDocumented" >
        <xs:sequence>
          <xs:element name="operation" type="wsdl:tOperation" minOccurs="0" maxOccurs="unbounded" />
        </xs:sequence>
        <xs:attribute name="name" type="xs:NCName" use="required" />
      </xs:extension>
    </xs:complexContent>   
  </xs:complexType>
   
  <xs:complexType name="tOperation" >
    <xs:complexContent>   
      <xs:extension base="wsdl:tExtensibleDocumented" >
	    <xs:sequence>
          <xs:choice>
            <xs:group ref="wsdl:request-response-or-one-way-operation" />
            <xs:group ref="wsdl:solicit-response-or-notification-operation" />
          </xs:choice>
        </xs:sequence>
        <xs:attribute name="name" type="xs:NCName" use="required" />
        <xs:attribute name="parameterOrder" type="xs:NMTOKENS" use="optional" />
      </xs:extension>
    </xs:complexContent>   
  </xs:complexType>
    
  <xs:group name="request-response-or-one-way-operation" >
    <xs:sequence>
      <xs:element name="input" type="wsdl:tParam" />
	  <xs:sequence minOccurs='0' >
	    <xs:element name="output" type="wsdl:tParam" />
		<xs:element name="fault" type="wsdl:tFault" minOccurs="0" maxOccurs="unbounded" />
      </xs:sequence>
    </xs:sequence>
  </xs:group>

  <xs:group name="solicit-response-or-notification-operation" >
    <xs:sequence>
      <xs:element name="output" type="wsdl:tParam" />
	  <xs:sequence minOccurs='0' >
	    <xs:element name="input" type="wsdl:tParam" />
		<xs:element name="fault" type="wsdl:tFault" minOccurs="0" maxOccurs="unbounded" />
	  </xs:sequence>
    </xs:sequence>
  </xs:group>
        
  <xs:complexType name="tParam" >
    <xs:complexContent>
      <xs:extension base="wsdl:tExtensibleAttributesDocumented" >
        <xs:attribute name="name" type="xs:NCName" use="optional" />
        <xs:attribute name="message" type="xs:QName" use="required" />
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>

  <xs:complexType name="tFault" >
    <xs:complexContent>
      <xs:extension base="wsdl:tExtensibleAttributesDocumented" >
        <xs:attribute name="name" type="xs:NCName"  use="required" />
        <xs:attribute name="message" type="xs:QName" use="required" />
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
     
  <xs:complexType name="tBinding" >
    <xs:complexContent>
      <xs:extension base="wsdl:tExtensibleDocumented" >
        <xs:sequence>
          <xs:element name="operation" type="wsdl:tBindingOperation" minOccurs="0" maxOccurs="unbounded" />
        </xs:sequence>
        <xs:attribute name="name" type="xs:NCName" use="required" />
        <xs:attribute name="type" type="xs:QName" use="required" />
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
    
  <xs:complexType name="tBindingOperationMessage" >
    <xs:complexContent>
      <xs:extension base="wsdl:tExtensibleDocumented" >
        <xs:attribute name="name" type="xs:NCName" use="optional" />
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  
  <xs:complexType name="tBindingOperationFault" >
    <xs:complexContent>
      <xs:extension base="wsdl:tExtensibleDocumented" >
        <xs:attribute name="name" type="xs:NCName" use="required" />
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>

  <xs:complexType name="tBindingOperation" >
    <xs:complexContent>
      <xs:extension base="wsdl:tExtensibleDocumented" >
        <xs:sequence>
          <xs:element name="input" type="wsdl:tBindingOperationMessage" minOccurs="0" />
          <xs:element name="output" type="wsdl:tBindingOperationMessage" minOccurs="0" />
          <xs:element name="fault" type="wsdl:tBindingOperationFault" minOccurs="0" maxOccurs="unbounded" />
        </xs:sequence>
        <xs:attribute name="name" type="xs:NCName" use="required" />
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
     
  <xs:complexType name="tService" >
    <xs:complexContent>
      <xs:extension base="wsdl:tExtensibleDocumented" >
        <xs:sequence>
          <xs:element name="port" type="wsdl:tPort" minOccurs="0" maxOccurs="unbounded" />
        </xs:sequence>
        <xs:attribute name="name" type="xs:NCName" use="required" />
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
     
  <xs:complexType name="tPort" >
    <xs:complexContent>
      <xs:extension base="wsdl:tExtensibleDocumented" >
        <xs:attribute name="name" type="xs:NCName" use="required" />
        <xs:attribute name="binding" type="xs:QName" use="required" />
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>

  <xs:attribute name="arrayType" type="xs:string" />
  <xs:attribute name="required" type="xs:boolean" />
  <xs:complexType name="tExtensibilityElement" abstract="true" >
    <xs:attribute ref="wsdl:required" use="optional" />
  </xs:complexType>

</xs:schema>`)

var soapenc11xsd = []byte(`<?xml version='1.0' encoding='UTF-8' ?>

<!-- Schema for the SOAP/1.1 encoding

Portions © 2001 DevelopMentor. 
© 2001 W3C (Massachusetts Institute of Technology, Institut National de Recherche en Informatique et en Automatique, Keio University). All Rights Reserved.  
 
This document is governed by the W3C Software License [1] as described in the FAQ [2].
[1] http://www.w3.org/Consortium/Legal/copyright-software-19980720
[2] http://www.w3.org/Consortium/Legal/IPR-FAQ-20000620.html#DTD 
By obtaining, using and/or copying this work, you (the licensee) agree that you have read, understood, and will comply with the following terms and conditions:

Permission to use, copy, modify, and distribute this software and its documentation, with or without modification,  for any purpose and without fee or royalty is hereby granted, provided that you include the following on ALL copies of the software and documentation or portions thereof, including modifications, that you make:

1.  The full text of this NOTICE in a location viewable to users of the redistributed or derivative work. 

2.  Any pre-existing intellectual property disclaimers, notices, or terms and conditions. If none exist, a short notice of the following form (hypertext is preferred, text is permitted) should be used within the body of any redistributed or derivative code: "Copyright © 2001 World Wide Web Consortium, (Massachusetts Institute of Technology, Institut National de Recherche en Informatique et en Automatique, Keio University). All Rights Reserved. http://www.w3.org/Consortium/Legal/" 

3.  Notice of any changes or modifications to the W3C files, including the date changes were made. (We recommend you provide URIs to the location from which the code is derived.)   

Original W3C files; http://www.w3.org/2001/06/soap-encoding
Changes made: 
     - reverted namespace to http://schemas.xmlsoap.org/soap/encoding/
     - reverted root to only allow 0 and 1 as lexical values
	 - removed default value from root attribute declaration

THIS SOFTWARE AND DOCUMENTATION IS PROVIDED "AS IS," AND COPYRIGHT HOLDERS MAKE NO REPRESENTATIONS OR WARRANTIES, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO, WARRANTIES OF MERCHANTABILITY OR FITNESS FOR ANY PARTICULAR PURPOSE OR THAT THE USE OF THE SOFTWARE OR DOCUMENTATION WILL NOT INFRINGE ANY THIRD PARTY PATENTS, COPYRIGHTS, TRADEMARKS OR OTHER RIGHTS.

COPYRIGHT HOLDERS WILL NOT BE LIABLE FOR ANY DIRECT, INDIRECT, SPECIAL OR CONSEQUENTIAL DAMAGES ARISING OUT OF ANY USE OF THE SOFTWARE OR DOCUMENTATION.

The name and trademarks of copyright holders may NOT be used in advertising or publicity pertaining to the software without specific, written prior permission. Title to copyright in this software and any associated documentation will at all times remain with copyright holders.

-->
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:tns="http://schemas.xmlsoap.org/soap/encoding/"
           targetNamespace="http://schemas.xmlsoap.org/soap/encoding/" >
        
 <xs:attribute name="root" >
   <xs:annotation>
     <xs:documentation>
	   'root' can be used to distinguish serialization roots from other
       elements that are present in a serialization but are not roots of
       a serialized value graph 
	 </xs:documentation>
   </xs:annotation>
   <xs:simpleType>
     <xs:restriction base='xs:boolean'>
	   <xs:pattern value='0|1' />
	 </xs:restriction>
   </xs:simpleType>
 </xs:attribute>

  <xs:attributeGroup name="commonAttributes" >
    <xs:annotation>
	  <xs:documentation>
	    Attributes common to all elements that function as accessors or 
        represent independent (multi-ref) values.  The href attribute is
        intended to be used in a manner like CONREF.  That is, the element
        content should be empty iff the href attribute appears
	  </xs:documentation>
	</xs:annotation>
    <xs:attribute name="id" type="xs:ID" />
    <xs:attribute name="href" type="xs:anyURI" />
    <xs:anyAttribute namespace="##other" processContents="lax" />
  </xs:attributeGroup>

  <!-- Global Attributes.  The following attributes are intended to be usable via qualified attribute names on any complex type referencing them. -->
       
  <!-- Array attributes. Needed to give the type and dimensions of an array's contents, and the offset for partially-transmitted arrays. -->
   
  <xs:simpleType name="arrayCoordinate" >
    <xs:restriction base="xs:string" />
  </xs:simpleType>
          
  <xs:attribute name="arrayType" type="xs:string" />
  <xs:attribute name="offset" type="tns:arrayCoordinate" />
  
  <xs:attributeGroup name="arrayAttributes" >
    <xs:attribute ref="tns:arrayType" />
    <xs:attribute ref="tns:offset" />
  </xs:attributeGroup>    
  
  <xs:attribute name="position" type="tns:arrayCoordinate" /> 
  
  <xs:attributeGroup name="arrayMemberAttributes" >
    <xs:attribute ref="tns:position" />
  </xs:attributeGroup>    

  <xs:group name="Array" >
    <xs:sequence>
      <xs:any namespace="##any" minOccurs="0" maxOccurs="unbounded" processContents="lax" />
	</xs:sequence>
  </xs:group>

  <xs:element name="Array" type="tns:Array" />
  <xs:complexType name="Array" >
    <xs:annotation>
	  <xs:documentation>
	   'Array' is a complex type for accessors identified by position 
	  </xs:documentation>
	</xs:annotation>
    <xs:group ref="tns:Array" minOccurs="0" />
    <xs:attributeGroup ref="tns:arrayAttributes" />
    <xs:attributeGroup ref="tns:commonAttributes" />
  </xs:complexType> 

  <!-- 'Struct' is a complex type for accessors identified by name. 
       Constraint: No element may be have the same name as any other,
       nor may any element have a maxOccurs > 1. -->
   
  <xs:element name="Struct" type="tns:Struct" />

  <xs:group name="Struct" >
    <xs:sequence>
      <xs:any namespace="##any" minOccurs="0" maxOccurs="unbounded" processContents="lax" />
	</xs:sequence>
  </xs:group>

  <xs:complexType name="Struct" >
    <xs:group ref="tns:Struct" minOccurs="0" />
    <xs:attributeGroup ref="tns:commonAttributes"/>
  </xs:complexType> 

  <!-- 'Base64' can be used to serialize binary data using base64 encoding
       as defined in RFC2045 but without the MIME line length limitation. -->

  <xs:simpleType name="base64" >
    <xs:restriction base="xs:base64Binary" />
  </xs:simpleType>

 <!-- Element declarations corresponding to each of the simple types in the 
      XML Schemas Specification. -->

  <xs:element name="duration" type="tns:duration" />
  <xs:complexType name="duration" >
    <xs:simpleContent>
      <xs:extension base="xs:duration" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="dateTime" type="tns:dateTime" />
  <xs:complexType name="dateTime" >
    <xs:simpleContent>
      <xs:extension base="xs:dateTime" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>



  <xs:element name="NOTATION" type="tns:NOTATION" />
  <xs:complexType name="NOTATION" >
    <xs:simpleContent>
      <xs:extension base="xs:QName" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  

  <xs:element name="time" type="tns:time" />
  <xs:complexType name="time" >
    <xs:simpleContent>
      <xs:extension base="xs:time" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="date" type="tns:date" />
  <xs:complexType name="date" >
    <xs:simpleContent>
      <xs:extension base="xs:date" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="gYearMonth" type="tns:gYearMonth" />
  <xs:complexType name="gYearMonth" >
    <xs:simpleContent>
      <xs:extension base="xs:gYearMonth" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="gYear" type="tns:gYear" />
  <xs:complexType name="gYear" >
    <xs:simpleContent>
      <xs:extension base="xs:gYear" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="gMonthDay" type="tns:gMonthDay" />
  <xs:complexType name="gMonthDay" >
    <xs:simpleContent>
      <xs:extension base="xs:gMonthDay" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="gDay" type="tns:gDay" />
  <xs:complexType name="gDay" >
    <xs:simpleContent>
      <xs:extension base="xs:gDay" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="gMonth" type="tns:gMonth" />
  <xs:complexType name="gMonth" >
    <xs:simpleContent>
      <xs:extension base="xs:gMonth" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  
  <xs:element name="boolean" type="tns:boolean" />
  <xs:complexType name="boolean" >
    <xs:simpleContent>
      <xs:extension base="xs:boolean" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="base64Binary" type="tns:base64Binary" />
  <xs:complexType name="base64Binary" >
    <xs:simpleContent>
      <xs:extension base="xs:base64Binary" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="hexBinary" type="tns:hexBinary" />
  <xs:complexType name="hexBinary" >
    <xs:simpleContent>
     <xs:extension base="xs:hexBinary" >
       <xs:attributeGroup ref="tns:commonAttributes" />
     </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="float" type="tns:float" />
  <xs:complexType name="float" >
    <xs:simpleContent>
      <xs:extension base="xs:float" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="double" type="tns:double" />
  <xs:complexType name="double" >
    <xs:simpleContent>
      <xs:extension base="xs:double" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="anyURI" type="tns:anyURI" />
  <xs:complexType name="anyURI" >
    <xs:simpleContent>
      <xs:extension base="xs:anyURI" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="QName" type="tns:QName" />
  <xs:complexType name="QName" >
    <xs:simpleContent>
      <xs:extension base="xs:QName" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  
  <xs:element name="string" type="tns:string" />
  <xs:complexType name="string" >
    <xs:simpleContent>
      <xs:extension base="xs:string" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="normalizedString" type="tns:normalizedString" />
  <xs:complexType name="normalizedString" >
    <xs:simpleContent>
      <xs:extension base="xs:normalizedString" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="token" type="tns:token" />
  <xs:complexType name="token" >
    <xs:simpleContent>
      <xs:extension base="xs:token" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="language" type="tns:language" />
  <xs:complexType name="language" >
    <xs:simpleContent>
      <xs:extension base="xs:language" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="Name" type="tns:Name" />
  <xs:complexType name="Name" >
    <xs:simpleContent>
      <xs:extension base="xs:Name" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="NMTOKEN" type="tns:NMTOKEN" />
  <xs:complexType name="NMTOKEN" >
    <xs:simpleContent>
      <xs:extension base="xs:NMTOKEN" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="NCName" type="tns:NCName" />
  <xs:complexType name="NCName" >
    <xs:simpleContent>
      <xs:extension base="xs:NCName" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="NMTOKENS" type="tns:NMTOKENS" />
  <xs:complexType name="NMTOKENS" >
    <xs:simpleContent>
      <xs:extension base="xs:NMTOKENS" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="ID" type="tns:ID" />
  <xs:complexType name="ID" >
    <xs:simpleContent>
      <xs:extension base="xs:ID" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="IDREF" type="tns:IDREF" />
  <xs:complexType name="IDREF" >
    <xs:simpleContent>
      <xs:extension base="xs:IDREF" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="ENTITY" type="tns:ENTITY" />
  <xs:complexType name="ENTITY" >
    <xs:simpleContent>
      <xs:extension base="xs:ENTITY" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="IDREFS" type="tns:IDREFS" />
  <xs:complexType name="IDREFS" >
    <xs:simpleContent>
      <xs:extension base="xs:IDREFS" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="ENTITIES" type="tns:ENTITIES" />
  <xs:complexType name="ENTITIES" >
    <xs:simpleContent>
      <xs:extension base="xs:ENTITIES" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="decimal" type="tns:decimal" />
  <xs:complexType name="decimal" >
    <xs:simpleContent>
      <xs:extension base="xs:decimal" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="integer" type="tns:integer" />
  <xs:complexType name="integer" >
    <xs:simpleContent>
      <xs:extension base="xs:integer" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="nonPositiveInteger" type="tns:nonPositiveInteger" />
  <xs:complexType name="nonPositiveInteger" >
    <xs:simpleContent>
      <xs:extension base="xs:nonPositiveInteger" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="negativeInteger" type="tns:negativeInteger" />
  <xs:complexType name="negativeInteger" >
    <xs:simpleContent>
      <xs:extension base="xs:negativeInteger" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="long" type="tns:long" />
  <xs:complexType name="long" >
    <xs:simpleContent>
      <xs:extension base="xs:long" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="int" type="tns:int" />
  <xs:complexType name="int" >
    <xs:simpleContent>
      <xs:extension base="xs:int" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="short" type="tns:short" />
  <xs:complexType name="short" >
    <xs:simpleContent>
      <xs:extension base="xs:short" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="byte" type="tns:byte" />
  <xs:complexType name="byte" >
    <xs:simpleContent>
      <xs:extension base="xs:byte" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="nonNegativeInteger" type="tns:nonNegativeInteger" />
  <xs:complexType name="nonNegativeInteger" >
    <xs:simpleContent>
      <xs:extension base="xs:nonNegativeInteger" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="unsignedLong" type="tns:unsignedLong" />
  <xs:complexType name="unsignedLong" >
    <xs:simpleContent>
      <xs:extension base="xs:unsignedLong" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="unsignedInt" type="tns:unsignedInt" />
  <xs:complexType name="unsignedInt" >
    <xs:simpleContent>
      <xs:extension base="xs:unsignedInt" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="unsignedShort" type="tns:unsignedShort" />
  <xs:complexType name="unsignedShort" >
    <xs:simpleContent>
      <xs:extension base="xs:unsignedShort" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="unsignedByte" type="tns:unsignedByte" />
  <xs:complexType name="unsignedByte" >
    <xs:simpleContent>
      <xs:extension base="xs:unsignedByte" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="positiveInteger" type="tns:positiveInteger" />
  <xs:complexType name="positiveInteger" >
    <xs:simpleContent>
      <xs:extension base="xs:positiveInteger" >
        <xs:attributeGroup ref="tns:commonAttributes" />
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:element name="anyType" />
</xs:schema>

`)

var xmlnsxsd = []byte(`<?xml version='1.0'?>
<?xml-stylesheet href="../../2008/09/xsd.xsl" type="text/xsl"?>
<xs:schema targetNamespace="http://www.w3.org/XML/1998/namespace" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns   ="http://www.w3.org/1999/xhtml"
  xml:lang="en">

 <xs:annotation>
  <xs:documentation>
   <div>
    <h1>About the XML namespace</h1>

    <div class="bodytext">
     <p>
      This schema document describes the XML namespace, in a form
      suitable for import by other schema documents.
     </p>
     <p>
      See <a href="http://www.w3.org/XML/1998/namespace.html">
      http://www.w3.org/XML/1998/namespace.html</a> and
      <a href="http://www.w3.org/TR/REC-xml">
      http://www.w3.org/TR/REC-xml</a> for information 
      about this namespace.
     </p>
     <p>
      Note that local names in this namespace are intended to be
      defined only by the World Wide Web Consortium or its subgroups.
      The names currently defined in this namespace are listed below.
      They should not be used with conflicting semantics by any Working
      Group, specification, or document instance.
     </p>
     <p>   
      See further below in this document for more information about <a
      href="#usage">how to refer to this schema document from your own
      XSD schema documents</a> and about <a href="#nsversioning">the
      namespace-versioning policy governing this schema document</a>.
     </p>
    </div>
   </div>
  </xs:documentation>
 </xs:annotation>

 <xs:attribute name="lang">
  <xs:annotation>
   <xs:documentation>
    <div>
     
      <h3>lang (as an attribute name)</h3>
      <p>
       denotes an attribute whose value
       is a language code for the natural language of the content of
       any element; its value is inherited.  This name is reserved
       by virtue of its definition in the XML specification.</p>
     
    </div>
    <div>
     <h4>Notes</h4>
     <p>
      Attempting to install the relevant ISO 2- and 3-letter
      codes as the enumerated possible values is probably never
      going to be a realistic possibility.  
     </p>
     <p>
      See BCP 47 at <a href="http://www.rfc-editor.org/rfc/bcp/bcp47.txt">
       http://www.rfc-editor.org/rfc/bcp/bcp47.txt</a>
      and the IANA language subtag registry at
      <a href="http://www.iana.org/assignments/language-subtag-registry">
       http://www.iana.org/assignments/language-subtag-registry</a>
      for further information.
     </p>
     <p>
      The union allows for the 'un-declaration' of xml:lang with
      the empty string.
     </p>
    </div>
   </xs:documentation>
  </xs:annotation>
  <xs:simpleType>
   <xs:union memberTypes="xs:language">
    <xs:simpleType>    
     <xs:restriction base="xs:string">
      <xs:enumeration value=""/>
     </xs:restriction>
    </xs:simpleType>
   </xs:union>
  </xs:simpleType>
 </xs:attribute>

 <xs:attribute name="space">
  <xs:annotation>
   <xs:documentation>
    <div>
     
      <h3>space (as an attribute name)</h3>
      <p>
       denotes an attribute whose
       value is a keyword indicating what whitespace processing
       discipline is intended for the content of the element; its
       value is inherited.  This name is reserved by virtue of its
       definition in the XML specification.</p>
     
    </div>
   </xs:documentation>
  </xs:annotation>
  <xs:simpleType>
   <xs:restriction base="xs:NCName">
    <xs:enumeration value="default"/>
    <xs:enumeration value="preserve"/>
   </xs:restriction>
  </xs:simpleType>
 </xs:attribute>
 
 <xs:attribute name="base" type="xs:anyURI"> <xs:annotation>
   <xs:documentation>
    <div>
     
      <h3>base (as an attribute name)</h3>
      <p>
       denotes an attribute whose value
       provides a URI to be used as the base for interpreting any
       relative URIs in the scope of the element on which it
       appears; its value is inherited.  This name is reserved
       by virtue of its definition in the XML Base specification.</p>
     
     <p>
      See <a
      href="http://www.w3.org/TR/xmlbase/">http://www.w3.org/TR/xmlbase/</a>
      for information about this attribute.
     </p>
    </div>
   </xs:documentation>
  </xs:annotation>
 </xs:attribute>
 
 <xs:attribute name="id" type="xs:ID">
  <xs:annotation>
   <xs:documentation>
    <div>
     
      <h3>id (as an attribute name)</h3> 
      <p>
       denotes an attribute whose value
       should be interpreted as if declared to be of type ID.
       This name is reserved by virtue of its definition in the
       xml:id specification.</p>
     
     <p>
      See <a
      href="http://www.w3.org/TR/xml-id/">http://www.w3.org/TR/xml-id/</a>
      for information about this attribute.
     </p>
    </div>
   </xs:documentation>
  </xs:annotation>
 </xs:attribute>

 <xs:attributeGroup name="specialAttrs">
  <xs:attribute ref="xml:base"/>
  <xs:attribute ref="xml:lang"/>
  <xs:attribute ref="xml:space"/>
  <xs:attribute ref="xml:id"/>
 </xs:attributeGroup>

 <xs:annotation>
  <xs:documentation>
   <div>
   
    <h3>Father (in any context at all)</h3> 

    <div class="bodytext">
     <p>
      denotes Jon Bosak, the chair of 
      the original XML Working Group.  This name is reserved by 
      the following decision of the W3C XML Plenary and 
      XML Coordination groups:
     </p>
     <blockquote>
       <p>
	In appreciation for his vision, leadership and
	dedication the W3C XML Plenary on this 10th day of
	February, 2000, reserves for Jon Bosak in perpetuity
	the XML name "xml:Father".
       </p>
     </blockquote>
    </div>
   </div>
  </xs:documentation>
 </xs:annotation>

 <xs:annotation>
  <xs:documentation>
   <div xml:id="usage" id="usage">
    <h2><a name="usage">About this schema document</a></h2>

    <div class="bodytext">
     <p>
      This schema defines attributes and an attribute group suitable
      for use by schemas wishing to allow <code>xml:base</code>,
      <code>xml:lang</code>, <code>xml:space</code> or
      <code>xml:id</code> attributes on elements they define.
     </p>
     <p>
      To enable this, such a schema must import this schema for
      the XML namespace, e.g. as follows:
     </p>
     <pre>
          &lt;schema . . .>
           . . .
           &lt;import namespace="http://www.w3.org/XML/1998/namespace"
                      schemaLocation="http://www.w3.org/2001/xml.xsd"/>
     </pre>
     <p>
      or
     </p>
     <pre>
           &lt;import namespace="http://www.w3.org/XML/1998/namespace"
                      schemaLocation="http://www.w3.org/2009/01/xml.xsd"/>
     </pre>
     <p>
      Subsequently, qualified reference to any of the attributes or the
      group defined below will have the desired effect, e.g.
     </p>
     <pre>
          &lt;type . . .>
           . . .
           &lt;attributeGroup ref="xml:specialAttrs"/>
     </pre>
     <p>
      will define a type which will schema-validate an instance element
      with any of those attributes.
     </p>
    </div>
   </div>
  </xs:documentation>
 </xs:annotation>

 <xs:annotation>
  <xs:documentation>
   <div id="nsversioning" xml:id="nsversioning">
    <h2><a name="nsversioning">Versioning policy for this schema document</a></h2>
    <div class="bodytext">
     <p>
      In keeping with the XML Schema WG's standard versioning
      policy, this schema document will persist at
      <a href="http://www.w3.org/2009/01/xml.xsd">
       http://www.w3.org/2009/01/xml.xsd</a>.
     </p>
     <p>
      At the date of issue it can also be found at
      <a href="http://www.w3.org/2001/xml.xsd">
       http://www.w3.org/2001/xml.xsd</a>.
     </p>
     <p>
      The schema document at that URI may however change in the future,
      in order to remain compatible with the latest version of XML
      Schema itself, or with the XML namespace itself.  In other words,
      if the XML Schema or XML namespaces change, the version of this
      document at <a href="http://www.w3.org/2001/xml.xsd">
       http://www.w3.org/2001/xml.xsd 
      </a> 
      will change accordingly; the version at 
      <a href="http://www.w3.org/2009/01/xml.xsd">
       http://www.w3.org/2009/01/xml.xsd 
      </a> 
      will not change.
     </p>
     <p>
      Previous dated (and unchanging) versions of this schema 
      document are at:
     </p>
     <ul>
      <li><a href="http://www.w3.org/2009/01/xml.xsd">
	http://www.w3.org/2009/01/xml.xsd</a></li>
      <li><a href="http://www.w3.org/2007/08/xml.xsd">
	http://www.w3.org/2007/08/xml.xsd</a></li>
      <li><a href="http://www.w3.org/2004/10/xml.xsd">
	http://www.w3.org/2004/10/xml.xsd</a></li>
      <li><a href="http://www.w3.org/2001/03/xml.xsd">
	http://www.w3.org/2001/03/xml.xsd</a></li>
     </ul>
    </div>
   </div>
  </xs:documentation>
 </xs:annotation>

</xs:schema>
`)

var standardSchema = Schema{
	TargetNS: schemaNS,
}

func init() {
	standardSchema.Types = make(map[xml.Name]Type, int(UnsignedShort))
	for i := AnyType; i < UnsignedShort; i++ {
		standardSchema.Types[i.Name()] = i
	}
}
