package jsonobj

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
)

// c constructs the string out of in parameters.
func c(in ...interface{}) string {
	s := ""
	for _, v := range in {
		if s != "" {
			s += " "
		}
		s += fmt.Sprintf("%#v", v)
	}
	return s
}

// P prints any number of in in one line.
func p(in ...interface{}) {
	fmt.Println(c(in...))
}

// K panics with all in in one line.
func k(in ...interface{}) {
	panic(c(in...))
}

func Get(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	buff := bytes.NewBuffer(nil)
	if _, err := buff.ReadFrom(resp.Body); err != nil {
		return "", err
	}
	return buff.String(), nil
}

func TestArray(t *testing.T) {

	const json = `[
	{
		"name" : "Mirko",
		"age" : 42
	},
	{
		"name" : "Mirjana",
		"age": 34
	}
]`

	j, err := Unmarshal([]byte(json))
	if err != nil {
		t.Fatal("TestArray failed", err)
	}
	var Put string
	if err := j.Get("[1].name", &Put); err != nil {
		t.Fatal("TestArray.Get failed", err)
	}
	p(Put)
}

func TestSet1(t *testing.T) {

	const json = `{
		"items" : [
			{
				"name": "Mirko",
				"age": 42
			},
			{
				"name": "Mirjana",
				"age": 64
			}
		]
}`
	p(json)

	j, err := Unmarshal([]byte(json))
	if err != nil {
		t.Fatal("TestSet1 failed", err)
	}
	if err := j.Set("items[1].age", "pregzbušt"); err != nil {
		t.Fatal("TestSet1.Set failed", err)
	}

	out, err := j.Export("")
	if err != nil {
		t.Fatal("TestSet1.Export failed", err)
	}
	p(string(out))

	type intyp struct {
		Name    string
		Address string
	}

	// replace array with a different type
	in := intyp{"Votevr", "Adresa"}
	if err = j.Set("items[0]", in); err != nil {
		t.Fatal("TestSet1.Set2 failed", err)
	}

	out, err = j.Export("")
	if err != nil {
		t.Fatal("TestSet1.Export failed", err)
	}
	p(string(out))

	// Add another index
	in = intyp{"Antverp", "Adsresasasa"}
	if err = j.Set("items[1]", in); err != nil {
		t.Fatal("TestSet1.Set3 failed", err)
	}

	out, err = j.Export("")
	if err != nil {
		t.Fatal("TestSet1.Export failed", err)
	}
	p(string(out))

}
