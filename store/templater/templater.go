package templater

import (
	"strconv"

	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/toolkit/templater"
	"github.com/digital-foxy/toolkit/timestamp"
)

// Templater is a template engine for resource records
type Templater = templater.Templater[*resource.Record]

// CompiledTemplate is a pre-compiled template for resource records
type CompiledTemplate = templater.CompiledTemplate[*resource.Record]

// Token is a template token that extracts values from resource records
type Token = templater.Token[*resource.Record]

// BasicToken is a simple token with a key and extractor function
type BasicToken = templater.BasicToken[*resource.Record]

// RichToken is a token with additional metadata like description
type RichToken = templater.RichToken[*resource.Record]

// Tokens is the list of available template tokens for resource records
var Tokens = []Token{
	&RichToken{
		BasicToken: BasicToken{
			Key: "{{SOURCE}}",
			Extractor: func(record *resource.Record) string {
				return string(record.InfoData.Source)
			},
		},
		Description: "The source platform of the card",
	},
	&RichToken{
		BasicToken: BasicToken{
			Key: "{{PLATFORM_ID}}",
			Extractor: func(record *resource.Record) string {
				return record.InfoData.PlatformID
			},
		},
		Description: "The unique platform identifier for the card",
	},
	&RichToken{
		BasicToken: BasicToken{
			Key: "{{CHARACTER_ID}}",
			Extractor: func(record *resource.Record) string {
				return record.CharacterID
			},
		},
		Description: "The unique character identifier",
	},
	&RichToken{
		BasicToken: BasicToken{
			Key: "{{NAME}}",
			Extractor: func(record *resource.Record) string {
				return record.Name
			},
		},
		Description: "The name of the character",
	},
	&RichToken{
		BasicToken: BasicToken{
			Key: "{{TITLE}}",
			Extractor: func(record *resource.Record) string {
				return record.Title
			},
		},
		Description: "The title of the card",
	},
	&RichToken{
		BasicToken: BasicToken{
			Key: "{{CREATE_TIME}}",
			Extractor: func(record *resource.Record) string {
				return strconv.Itoa(int(timestamp.ConvertToSeconds(record.CreateTime)))
			},
		},
		Description: "The creation timestamp in seconds",
	},
	&RichToken{
		BasicToken: BasicToken{
			Key: "{{CREATE_DATE}}",
			Extractor: func(record *resource.Record) string {
				return strconv.Itoa(int(timestamp.ConvertToSeconds(record.CreateTime)))
			},
		},
		Description: "The creation date in UTC format",
	},
	&RichToken{
		BasicToken: BasicToken{
			Key: "{{UPDATE_TIME}}",
			Extractor: func(record *resource.Record) string {
				return strconv.Itoa(int(timestamp.ConvertToSeconds(record.UpdateTime)))
			},
		},
		Description: "The last update timestamp in seconds",
	},
	&RichToken{
		BasicToken: BasicToken{
			Key: "{{UPDATE_DATE}}",
			Extractor: func(record *resource.Record) string {
				return strconv.Itoa(int(timestamp.ConvertToSeconds(record.UpdateTime)))
			},
		},
		Description: "The last update date in UTC format",
	},
	&RichToken{
		BasicToken: BasicToken{
			Key: "{{C_NICKNAME}}",
			Extractor: func(record *resource.Record) string {
				return record.Nickname
			},
		},
		Description: "The nickname of the card creator",
	},
	&RichToken{
		BasicToken: BasicToken{
			Key: "{{C_USERNAME}}",
			Extractor: func(record *resource.Record) string {
				return record.Username
			},
		},
		Description: "The username of the card creator",
	},
}

// New creates a new Templater with all available tokens
func New() *Templater {
	return templater.New(Tokens...)
}
