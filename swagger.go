package main

import (
	"bytes"
	"io/ioutil"
	"os"

	"github.com/alecthomas/template"
	"github.com/swaggo/swag"
)

type swaggerInfo struct {
	Version     string
	Host        string
	BasePath    string
	Title       string
	Description string
}

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo swaggerInfo
//SwaggerDocPath : swagger path
const (
	SwaggerDocPath = "./crypto-wallet-adapter.yaml"
)

type s struct{}

func (s *s) ReadDoc() string {
	file, _ := os.Open(SwaggerDocPath)
	defer file.Close()

	result, _ := ioutil.ReadAll(file)

	doc := string(result)

	t, err := template.New("swagger_info").ParseFiles(doc)
	if err != nil {
		return doc
	}

	var tpl bytes.Buffer
	if err := t.Execute(&tpl, SwaggerInfo); err != nil {
		return doc
	}

	return tpl.String()
}

func init() {
	swag.Register(swag.Name, &s{})
}
