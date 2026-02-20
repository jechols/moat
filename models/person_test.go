package models

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestPersonRoundTrip(t *testing.T) {
	// 1. Read the original XML file
	originalData, err := os.ReadFile("testdata/person.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// 2. Unmarshal into the struct (p1)
	var p1 Person
	if err := xml.Unmarshal(originalData, &p1); err != nil {
		t.Fatalf("Failed to unmarshal original XML: %v", err)
	}

	// 3. Marshal p1 back to XML
	generatedData, err := xml.MarshalIndent(p1, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal p1: %v", err)
	}

	// 4. Compare originalData and generatedData using semantic XML comparison
	if err := assertXMLEqual(originalData, generatedData); err != nil {
		t.Errorf("XML Round-trip verification failed: %v", err)
		// Output the generated XML for debugging
		t.Logf("Generated XML:\n%s", string(generatedData))
	}
}

// assertXMLEqual compares two XML byte slices for semantic equality.
// It ignores whitespace, attribute order, and namespace prefixes (by resolving them).
func assertXMLEqual(xml1, xml2 []byte) error {
	d1 := xml.NewDecoder(bytes.NewReader(xml1))
	d2 := xml.NewDecoder(bytes.NewReader(xml2))

	for {
		tok1, err1 := nextSignificantToken(d1)
		tok2, err2 := nextSignificantToken(d2)

		if err1 == io.EOF && err2 == io.EOF {
			return nil
		}
		if err1 == io.EOF || err2 == io.EOF {
			return fmt.Errorf("XML length mismatch: one stream ended before the other")
		}
		if err1 != nil {
			return err1
		}
		if err2 != nil {
			return err2
		}

		if !tokensEqual(tok1, tok2) {
			return fmt.Errorf("token mismatch:\nExpect: %v\nGot:    %v", tok1, tok2)
		}
	}
}

func nextSignificantToken(d *xml.Decoder) (xml.Token, error) {
	for {
		tok, err := d.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.Comment, xml.ProcInst, xml.Directive:
			continue
		case xml.CharData:
			if len(bytes.TrimSpace(t)) == 0 {
				continue
			}
			return t, nil
		default:
			return t, nil
		}
	}
}

func tokensEqual(t1, t2 xml.Token) bool {
	switch v1 := t1.(type) {
	case xml.StartElement:
		v2, ok := t2.(xml.StartElement)
		if !ok {
			return false
		}
		// Compare names (resolved namespaces)
		if v1.Name.Space != v2.Name.Space || v1.Name.Local != v2.Name.Local {
			return false
		}
		return attrsEqual(v1.Attr, v2.Attr)
	case xml.EndElement:
		v2, ok := t2.(xml.EndElement)
		if !ok {
			return false
		}
		return v1.Name.Space == v2.Name.Space && v1.Name.Local == v2.Name.Local
	case xml.CharData:
		v2, ok := t2.(xml.CharData)
		if !ok {
			return false
		}
		return string(bytes.TrimSpace(v1)) == string(bytes.TrimSpace(v2))
	default:
		return false
	}
}

func attrsEqual(attrs1, attrs2 []xml.Attr) bool {
	a1 := normalizeAttrs(attrs1)
	a2 := normalizeAttrs(attrs2)

	if len(a1) != len(a2) {
		return false
	}

	for i := range a1 {
		if a1[i].Name.Space != a2[i].Name.Space ||
			a1[i].Name.Local != a2[i].Name.Local ||
			a1[i].Value != a2[i].Value {
			return false
		}
	}
	return true
}

func normalizeAttrs(attrs []xml.Attr) []xml.Attr {
	var out []xml.Attr
	for _, a := range attrs {
		// Ignore xmlns definitions
		if a.Name.Space == "xmlns" || a.Name.Local == "xmlns" || strings.HasPrefix(a.Name.Local, "xmlns:") {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		key1 := out[i].Name.Space + "|" + out[i].Name.Local
		key2 := out[j].Name.Space + "|" + out[j].Name.Local
		return key1 < key2
	})
	return out
}
