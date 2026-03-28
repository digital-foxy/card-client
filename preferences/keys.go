package preferences

import (
	"strings"

	"github.com/digital-foxy/card-client/store/templater"
	"github.com/digital-foxy/toolkit/filex"
)

type ValueType byte

const (
	IntegerValue ValueType = iota
	StringValue
	BoolValue
)

type KeyType byte

const (
	StandardKey KeyType = iota
	DirectoryPathKey
	FilePathKey
)

var templateTokenDescription = func() string {
	var sb strings.Builder
	for _, token := range templater.Tokens {
		sb.WriteString("\n" + token.GetKey() + " - " + token.GetDescription())
	}
	return sb.String()
}()

type Key struct {
	ID           string
	Name         string
	KeyType      KeyType
	DefaultValue any
	ValueType    ValueType
	Description  string
}

var ExportPathKey = Key{
	ID:           "export_path",
	Name:         "Export Path",
	KeyType:      DirectoryPathKey,
	ValueType:    StringValue,
	DefaultValue: filex.GetCWD(),
	Description:  "The path where exported cards will be saved",
}
var MaxExportSizeKey = Key{
	ID:           "max_export_size",
	Name:         "Max Export Size",
	KeyType:      StandardKey,
	ValueType:    IntegerValue,
	DefaultValue: 3072,
	Description:  "The maximum size of the exported card PNG (if either width or height is greater than this value, the card will be scaled down, preserving aspect ratio, with the biggest dimension being this value)",
}
var ExportTemplateKey = Key{
	ID:           "export_template",
	Name:         "Export Template",
	KeyType:      StandardKey,
	ValueType:    StringValue,
	DefaultValue: "{{SOURCE}}_{{PLATFORM_ID}}.png",
	Description:  "The template used to generate the filename of exported cards. Available tokens: " + templateTokenDescription,
}
var LastLoadedVaultKey = Key{
	ID:           "last_loaded_vault",
	Name:         "Last Vault",
	KeyType:      StandardKey,
	ValueType:    StringValue,
	DefaultValue: "",
	Description:  "The name of the last vault loaded into the application.",
}
var RenderMdHTML = Key{
	ID:           "render_md_html",
	Name:         "Render Markdown/HTML",
	KeyType:      StandardKey,
	ValueType:    BoolValue,
	DefaultValue: false,
	Description:  "Allow rendering of Markdown and HTML in card sheet",
}
var ChromePath = Key{
	ID:           "chrome_path",
	Name:         "Chrome Path",
	KeyType:      StandardKey,
	ValueType:    StringValue,
	DefaultValue: "",
	Description:  "Custom path to Chrome executable. Leave blank to use default.",
}

var Keys = []Key{
	ExportPathKey,
	MaxExportSizeKey,
	ExportTemplateKey,
	ChromePath,
}
