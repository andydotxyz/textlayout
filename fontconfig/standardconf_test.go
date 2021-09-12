package fontconfig

import (
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func dumpStandard(t *testing.T) {
	c := NewConfig()
	if err := c.LoadFromDir("confs"); err != nil {
		t.Fatal(err)
	}

	content := fmt.Sprintf(`package fontconfig

	// Code generated by standardconf_test.go. DO NOT EDIT

	// Standard exposes the parsed configuration 
	// described in the 'confs' folder.
	var Standard = &%s
	`, c.asGoSource())

	file := []byte(content)
	file, err := format.Source(file)
	if err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile("standardconf.go", file, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	fmt.Println("Standard conf code written in standardconf.go")
}

func TestDoDumpStandard(t *testing.T) {
	// Uncomment and launch the test if the standard conf needs adjustements
	// dumpStandard(t)
}

func TestDumpStandardConfig(t *testing.T) {
	c := NewConfig()
	if err := c.LoadFromDir("confs"); err != nil {
		t.Fatal(err)
	}

	if len(c.subst) != len(Standard.subst) {
		t.Fatal("invalid rules dump length")
	}
	for i := range c.subst {
		exp, got := c.subst[i], Standard.subst[i]
		if !reflect.DeepEqual(got, exp) {
			for j := range exp.subst {
				ds1, ds2 := exp.subst[j], got.subst[j]
				if len(ds1) != len(ds2) {
					continue
				}
				for k := range ds1 {
					d1, d2 := ds1[k], ds2[k]
					if !reflect.DeepEqual(got, exp) {
						fmt.Printf("expected %#v\n, got %#v\n\n", d1, d2)
					}
				}
			}
			t.Fatalf("rule %d: expected\n%#v\n, got\n%#v", i, exp, got)
		}
	}

	if !reflect.DeepEqual(c, Standard) {
		t.Fatal("invalid config dump")
	}
}

func TestCustomObjects(t *testing.T) {
	c := NewConfig()
	if err := c.LoadFromDir("confs"); err != nil {
		t.Fatal(err)
	}

	fmt.Println(c.customObjects)
}
