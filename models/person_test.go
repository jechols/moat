package models

import (
	"encoding/xml"
	"os"
	"reflect"
	"testing"
)

func TestPersonRoundTrip(t *testing.T) {
	// 1. Read the original XML file
	originalData, err := os.ReadFile("testdata/person.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// 2. Unmarshal into the first struct (p1)
	var p1 Person
	if err := xml.Unmarshal(originalData, &p1); err != nil {
		t.Fatalf("Failed to unmarshal original XML: %v", err)
	}

	// 3. Marshal p1 back to XML
	// We use MarshalIndent to make it human-readable if we need to inspect it,
	// though for the machine comparison it doesn't matter much.
	generatedData, err := xml.MarshalIndent(p1, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal p1: %v", err)
	}

	// 4. Unmarshal the generated XML into a second struct (p2)
	var p2 Person
	if err := xml.Unmarshal(generatedData, &p2); err != nil {
		t.Fatalf("Failed to unmarshal generated XML: %v", err)
	}

	// 5. Compare p1 and p2
	// If the data round-tripped correctly, the structs should be identical.
	if !reflect.DeepEqual(p1, p2) {
		t.Errorf("Round-trip verification failed. p1 (original) != p2 (rehydrated)")
		
		// To help debugging, we can print what field might be different?
		// Since we can't easily traverse, we'll print the XML representations which might show differences.
		t.Logf("Original Struct (Marshaled):\n%s", generatedData)
		
		rehydratedData, _ := xml.MarshalIndent(p2, "", "  ")
		t.Logf("Rehydrated Struct (Marshaled):\n%s", rehydratedData)
	}
}
