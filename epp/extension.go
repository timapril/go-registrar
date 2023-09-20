package epp

import (
	"encoding/xml"
)

// Extension is used to construct and receive <extension> messages.
type Extension struct {
	XMLName                         xml.Name                   `xml:"extension" json:"-"`
	NameStoreExtensionObject        *NameStoreExtension        `xml:"namestoreExt:namestoreExt" json:"namestoreExt.namestoreExt"`
	GenericNameStoreExtensionObject *GenericNameStoreExtension `xml:"namestoreExt" json:"namestore"`
	SyncUpdateObject                *SyncUpdateExtension       `xml:"sync:update" json:"sync.update"`
	RestoreRequest                  *RestoreExtension          `xml:"rgp:update" json:"rgp.update"`
	SecDNSUpdate                    *SecDNSUpdate              `xml:"secDNS:update" json:"secDNS.update"`
}

// NameStoreExtension is used to construct and receive
// <namestoreExt:namestoreExt> messages.
type NameStoreExtension struct {
	XMLName              xml.Name `xml:"namestoreExt:namestoreExt" json:"-"`
	XMLNSNamestoreExt    string   `xml:"xmlns:namestoreExt,attr" json:"xmlns.namestoreExt"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr"  json:"xmlns.schemaLocation"`
	SubProducts          []string `xml:"namestoreExt:subProduct" json:"namestoreExt.subProduct"`
}

// GenericNameStoreExtension is used to receive
// <namestoreExt:namestoreExt> messages.
type GenericNameStoreExtension struct {
	XMLName              xml.Name `xml:"namestoreExt" json:"-"`
	XMLNSNamestoreExt    string   `xml:"namestoreExt,attr" json:"xmlns.namestoreExt"`
	XMLNSxsi             string   `xml:"xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`
	SubProducts          []string `xml:"subProduct" json:"subProduct"`
}

// NameStoreExtensionProduct is used to indicate a namestore product
// type.
type NameStoreExtensionProduct string

// NameStoreProductCOM holds the name of the verisign product that is
// used for domains in .COM.
const NameStoreProductCOM NameStoreExtensionProduct = "dotCOM"

// NameStoreProductNET holds the name of the verisign product that is
// used for domains in .NET.
const NameStoreProductNET NameStoreExtensionProduct = "dotNET"

// GetCOMNamestoreExtension will return an extension object for the .COM
// verisign product.
func GetCOMNamestoreExtension() Extension {
	ext := Extension{}
	ext.NameStoreExtensionObject = GetNameStoreExtension(NameStoreProductCOM)

	return ext
}

// GetNETNamestoreExtension will return an extension object for the .NET
// verisign product.
func GetNETNamestoreExtension() Extension {
	ext := Extension{}
	ext.NameStoreExtensionObject = GetNameStoreExtension(NameStoreProductNET)

	return ext
}

// GetDefaultNameStoreExtension creates a default NameStoreExtension
// object with the standard namespaces set.
func GetDefaultNameStoreExtension() *NameStoreExtension {
	nse := NameStoreExtension{}
	nse.XMLNSxsi = W3XMLNSxsi
	nse.XMLNSNamestoreExt = "http://www.verisign-grs.com/epp/namestoreExt-1.1"
	nse.XMLxsiSchemaLocation = "http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd"

	return &nse
}

// GetNameStoreExtension will create a NameStoreExtension object with
// the provided product name.
func GetNameStoreExtension(productName NameStoreExtensionProduct) *NameStoreExtension {
	nse := GetDefaultNameStoreExtension()
	nse.SubProducts = append(nse.SubProducts, string(productName))

	return nse
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (e GenericNameStoreExtension) TypedMessage() NameStoreExtension {
	out := NameStoreExtension{}

	out.XMLNSNamestoreExt = e.XMLNSNamestoreExt
	out.XMLNSxsi = e.XMLNSxsi
	out.XMLxsiSchemaLocation = e.XMLxsiSchemaLocation
	out.SubProducts = e.SubProducts

	return out
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (e Extension) TypedMessage() Extension {
	out := Extension{}

	// TODO
	if e.GenericNameStoreExtensionObject != nil {
		namestore := e.GenericNameStoreExtensionObject.TypedMessage()
		out.NameStoreExtensionObject = &namestore
	}
	// TODO
	// if e.SyncUpdateObject != nil {
	// 	sync := e.SyncUpdateObject.TypedMessage()
	// 	out.SyncUpdateObject = &sync
	// }
	// TODO
	// if e.RestoreRequest != nil {
	// 	restore := e.RestoreRequest.TypedMessage()
	// 	out.RestoreRequest = &restore
	// }

	return out
}
