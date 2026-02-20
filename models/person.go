package models

import "encoding/xml"

type Person struct {
	XMLName            xml.Name            `xml:"http://www.orcid.org/ns/person person"`
	XmlnsCommon        string              `xml:"xmlns:common,attr,omitempty"`
	XmlnsPerson        string              `xml:"xmlns:person,attr,omitempty"`
	XmlnsPersonal      string              `xml:"xmlns:personal-details,attr,omitempty"`
	XmlnsOtherName     string              `xml:"xmlns:other-name,attr,omitempty"`
	XmlnsResearcherUrl string              `xml:"xmlns:researcher-url,attr,omitempty"`
	XmlnsEmail         string              `xml:"xmlns:email,attr,omitempty"`
	XmlnsAddress       string              `xml:"xmlns:address,attr,omitempty"`
	XmlnsKeyword       string              `xml:"xmlns:keyword,attr,omitempty"`
	XmlnsExternalId    string              `xml:"xmlns:external-identifier,attr,omitempty"`
	XmlnsXsi           string              `xml:"xmlns:xsi,attr,omitempty"`
	XsiSchemaLocation  string              `xml:"http://www.w3.org/2001/XMLSchema-instance schemaLocation,attr,omitempty"`

	LastModifiedDate   *string             `xml:"http://www.orcid.org/ns/common last-modified-date"`
	Name               *PersonName         `xml:"http://www.orcid.org/ns/person name"`
	OtherNames         *OtherNames         `xml:"http://www.orcid.org/ns/other-name other-names"`
	Biography          *Biography          `xml:"http://www.orcid.org/ns/person biography"`
	ResearcherUrls     *ResearcherUrls     `xml:"http://www.orcid.org/ns/researcher-url researcher-urls"`
	Emails             *Emails             `xml:"http://www.orcid.org/ns/email emails"`
	Addresses          *Addresses          `xml:"http://www.orcid.org/ns/address addresses"`
	Keywords           *Keywords           `xml:"http://www.orcid.org/ns/keyword keywords"`
	ExternalIdentifiers *ExternalIdentifiers `xml:"http://www.orcid.org/ns/external-identifier external-identifiers"`
	Path               string              `xml:"path,attr,omitempty"`
}

type PersonName struct {
	Visibility       string  `xml:"visibility,attr,omitempty"`
	CreatedDate      *string `xml:"http://www.orcid.org/ns/common created-date"`
	LastModifiedDate *string `xml:"http://www.orcid.org/ns/common last-modified-date"`
	GivenNames       string  `xml:"http://www.orcid.org/ns/personal-details given-names,omitempty"`
	FamilyName       string  `xml:"http://www.orcid.org/ns/personal-details family-name,omitempty"`
	CreditName       string  `xml:"http://www.orcid.org/ns/personal-details credit-name,omitempty"`
}

type OtherNames struct {
	LastModifiedDate *string      `xml:"http://www.orcid.org/ns/common last-modified-date"`
	OtherNames       []*OtherName `xml:"http://www.orcid.org/ns/other-name other-name"`
}

type OtherName struct {
	Visibility       string  `xml:"visibility,attr,omitempty"`
	PutCode          string  `xml:"put-code,attr,omitempty"`
	CreatedDate      *string `xml:"http://www.orcid.org/ns/common created-date"`
	LastModifiedDate *string `xml:"http://www.orcid.org/ns/common last-modified-date"`
	Source           *Source `xml:"http://www.orcid.org/ns/common source"`
	Content          string  `xml:"http://www.orcid.org/ns/other-name content"`
}

type Biography struct {
	Visibility       string  `xml:"visibility,attr,omitempty"`
	CreatedDate      *string `xml:"http://www.orcid.org/ns/common created-date"`
	LastModifiedDate *string `xml:"http://www.orcid.org/ns/common last-modified-date"`
	Content          string  `xml:"http://www.orcid.org/ns/personal-details content"`
}

type ResearcherUrls struct {
	LastModifiedDate *string          `xml:"http://www.orcid.org/ns/common last-modified-date"`
	ResearcherUrls   []*ResearcherUrl `xml:"http://www.orcid.org/ns/researcher-url researcher-url"`
}

type ResearcherUrl struct {
	Visibility       string  `xml:"visibility,attr,omitempty"`
	PutCode          string  `xml:"put-code,attr,omitempty"`
	CreatedDate      *string `xml:"http://www.orcid.org/ns/common created-date"`
	LastModifiedDate *string `xml:"http://www.orcid.org/ns/common last-modified-date"`
	Source           *Source `xml:"http://www.orcid.org/ns/common source"`
	UrlName          string  `xml:"http://www.orcid.org/ns/researcher-url url-name"`
	Url              string  `xml:"http://www.orcid.org/ns/researcher-url url"`
}

type Emails struct {
	Emails []*Email `xml:"http://www.orcid.org/ns/email email"`
}

type Email struct {
	Visibility       string  `xml:"visibility,attr,omitempty"`
	CreatedDate      *string `xml:"http://www.orcid.org/ns/common created-date"`
	LastModifiedDate *string `xml:"http://www.orcid.org/ns/common last-modified-date"`
	Source           *Source `xml:"http://www.orcid.org/ns/common source"`
	Email            string  `xml:"http://www.orcid.org/ns/email email"`
}

type Addresses struct {
	Addresses []*Address `xml:"http://www.orcid.org/ns/address address"`
}

type Address struct {
	Visibility       string  `xml:"visibility,attr,omitempty"`
	PutCode          string  `xml:"put-code,attr,omitempty"`
	CreatedDate      *string `xml:"http://www.orcid.org/ns/common created-date"`
	LastModifiedDate *string `xml:"http://www.orcid.org/ns/common last-modified-date"`
	Source           *Source `xml:"http://www.orcid.org/ns/common source"`
	Country          string  `xml:"http://www.orcid.org/ns/address country"`
}

type Keywords struct {
	Keywords []*Keyword `xml:"http://www.orcid.org/ns/keyword keyword"`
}

type Keyword struct {
	Visibility       string  `xml:"visibility,attr,omitempty"`
	PutCode          string  `xml:"put-code,attr,omitempty"`
	CreatedDate      *string `xml:"http://www.orcid.org/ns/common created-date"`
	LastModifiedDate *string `xml:"http://www.orcid.org/ns/common last-modified-date"`
	Source           *Source `xml:"http://www.orcid.org/ns/common source"`
	Content          string  `xml:"http://www.orcid.org/ns/keyword content"`
}

type ExternalIdentifiers struct {
	ExternalIdentifiers []*ExternalIdentifier `xml:"http://www.orcid.org/ns/external-identifier external-identifier"`
}

type ExternalIdentifier struct {
	Visibility       string  `xml:"visibility,attr,omitempty"`
	PutCode          string  `xml:"put-code,attr,omitempty"`
	CreatedDate      *string `xml:"http://www.orcid.org/ns/common created-date"`
	LastModifiedDate *string `xml:"http://www.orcid.org/ns/common last-modified-date"`
	Source           *Source `xml:"http://www.orcid.org/ns/common source"`
	ExternalIdType   string  `xml:"http://www.orcid.org/ns/common external-id-type"`
	ExternalIdValue  string  `xml:"http://www.orcid.org/ns/common external-id-value"`
	ExternalIdUrl    string  `xml:"http://www.orcid.org/ns/common external-id-url"`
}

type Source struct {
	SourceOrcid *SourceOrcid `xml:"http://www.orcid.org/ns/common source-orcid"`
	SourceName  *SourceName  `xml:"http://www.orcid.org/ns/common source-name"`
}

type SourceOrcid struct {
	Uri  string `xml:"http://www.orcid.org/ns/common uri,omitempty"`
	Path string `xml:"http://www.orcid.org/ns/common path,omitempty"`
	Host string `xml:"http://www.orcid.org/ns/common host,omitempty"`
}

type SourceName struct {
	Value string `xml:",chardata"`
}
