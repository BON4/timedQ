// Package docs GENERATED BY SWAG; DO NOT EDIT
// This file was generated by swaggo/swag
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/": {
            "post": {
                "description": "sets key-value, where user is providing value, and gets key",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "general"
                ],
                "summary": "Set redirect",
                "parameters": [
                    {
                        "description": "encoded short url",
                        "name": "input",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/http.serviceSetRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/http.serviceSetResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {}
                    }
                }
            }
        },
        "/{key}": {
            "get": {
                "description": "by known key, user can get an url",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "general"
                ],
                "summary": "Get redirect",
                "parameters": [
                    {
                        "type": "string",
                        "description": "decoded full url",
                        "name": "key",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/http.serviceGetResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "http.serviceGetResponse": {
            "type": "object",
            "properties": {
                "decode_url": {
                    "type": "string"
                }
            }
        },
        "http.serviceSetRequest": {
            "type": "object",
            "required": [
                "redirect"
            ],
            "properties": {
                "redirect": {
                    "type": "string"
                }
            }
        },
        "http.serviceSetResponse": {
            "type": "object",
            "properties": {
                "encode_url": {
                    "type": "string"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "/v1",
	Schemes:          []string{},
	Title:            "TimedQ API",
	Description:      "This service will store values provided via API up to certain time. If the value has been accessed, expiration time updates. Key-Value stores in binary file with ttlStore package.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
