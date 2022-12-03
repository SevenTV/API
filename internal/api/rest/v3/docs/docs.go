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
        "termsOfService": "TODO",
        "contact": {
            "name": "7TV Developers",
            "url": "https://discord.gg/7tv",
            "email": "dev@7tv.io"
        },
        "license": {
            "name": "Apache 2.0 + Commons Clause",
            "url": "https://github.com/SevenTV/REST/blob/dev/LICENSE.md"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/emote-sets": {
            "get": {
                "description": "Search for Emote Sets",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "emote-sets"
                ],
                "summary": "Search Emote Sets",
                "parameters": [
                    {
                        "type": "string",
                        "description": "search by emote set name / tags",
                        "name": "query",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/model.EmoteSetModel"
                            }
                        }
                    }
                }
            }
        },
        "/emote-sets/{emote-set.id}": {
            "get": {
                "description": "Get an emote set by its ID",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "emote-sets"
                ],
                "summary": "Get Emote Set",
                "parameters": [
                    {
                        "type": "string",
                        "description": "ID of the emote set",
                        "name": "emote-set.id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/model.EmoteSetModel"
                        }
                    }
                }
            }
        },
        "/emotes": {
            "get": {
                "description": "Search for emotes",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "emotes"
                ],
                "summary": "Search Emotes",
                "parameters": [
                    {
                        "type": "string",
                        "description": "search by emote name / tags",
                        "name": "query",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/model.EmoteModel"
                            }
                        }
                    }
                }
            },
            "post": {
                "description": "Upload a new emote",
                "consumes": [
                    "image/webp",
                    "image/gif",
                    "image/png",
                    "image/apng",
                    "image/avif",
                    "image/jpeg",
                    "image/tiff",
                    "image/webm"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "emotes"
                ],
                "summary": "Create Emote",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Initial emote properties",
                        "name": "X-Emote-Data",
                        "in": "header"
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/model.EmoteModel"
                        }
                    }
                }
            }
        },
        "/emotes/{emote.id}": {
            "get": {
                "description": "Get emote by ID",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "emotes"
                ],
                "summary": "Get Emote",
                "parameters": [
                    {
                        "type": "string",
                        "description": "ID of the emote",
                        "name": "emoteID",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/model.EmoteModel"
                        }
                    }
                }
            }
        },
        "/users": {
            "get": {
                "description": "Search for users",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "users"
                ],
                "summary": "Search Users",
                "parameters": [
                    {
                        "type": "string",
                        "description": "search by username, user id, channel name or channel id",
                        "name": "query",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": ""
                    }
                }
            }
        },
        "/users/{connection.platform}/{connection.id}": {
            "get": {
                "description": "Query for a user's connected account and its attached emote set",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "users"
                ],
                "summary": "Get User Connection",
                "parameters": [
                    {
                        "type": "string",
                        "description": "twitch, youtube or discord user ID",
                        "name": "{connection.id}",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/model.UserConnectionModel"
                        }
                    }
                }
            }
        },
        "/users/{user.id}": {
            "get": {
                "description": "Get user by ID",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "users"
                ],
                "summary": "Get User",
                "parameters": [
                    {
                        "type": "string",
                        "description": "ID of the user",
                        "name": "userID",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/model.UserModel"
                        }
                    }
                }
            }
        },
        "/users/{user.id}/profile-picture": {
            "put": {
                "description": "Set a new profile picture",
                "consumes": [
                    "image/avif",
                    "image/webp",
                    "image/gif",
                    "image/apng",
                    "image/png",
                    "image/jpeg"
                ],
                "tags": [
                    "users"
                ],
                "summary": "Upload Profile Picture",
                "responses": {
                    "200": {
                        "description": ""
                    }
                }
            }
        }
    },
    "definitions": {
        "model.ActiveEmoteModel": {
            "type": "object",
            "properties": {
                "actor_id": {
                    "type": "string",
                    "x-nullable": true
                },
                "data": {
                    "x-nullable": true,
                    "$ref": "#/definitions/model.EmotePartialModel"
                },
                "flags": {
                    "type": "integer"
                },
                "id": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "origin_id": {
                    "type": "string",
                    "x-omitempty": true
                },
                "timestamp": {
                    "type": "integer"
                }
            }
        },
        "model.CosmeticPaintDropShadow": {
            "type": "object",
            "properties": {
                "color": {
                    "type": "integer"
                },
                "radius": {
                    "type": "number"
                },
                "x_offset": {
                    "type": "number"
                },
                "y_offset": {
                    "type": "number"
                }
            }
        },
        "model.CosmeticPaintGradientStop": {
            "type": "object",
            "properties": {
                "at": {
                    "type": "number"
                },
                "color": {
                    "type": "integer"
                }
            }
        },
        "model.CosmeticPaintModel": {
            "type": "object",
            "properties": {
                "angle": {
                    "type": "integer"
                },
                "color": {
                    "type": "integer"
                },
                "function": {
                    "type": "string",
                    "enum": [
                        "LINEAR_GRADIENT",
                        "RADIAL_GRADIENT",
                        "URL"
                    ]
                },
                "id": {
                    "type": "string"
                },
                "image_url": {
                    "type": "string"
                },
                "repeat": {
                    "type": "boolean"
                },
                "shadows": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/model.CosmeticPaintDropShadow"
                    }
                },
                "shape": {
                    "type": "string"
                },
                "stops": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/model.CosmeticPaintGradientStop"
                    }
                }
            }
        },
        "model.EmoteModel": {
            "type": "object",
            "properties": {
                "animated": {
                    "type": "boolean"
                },
                "flags": {
                    "type": "integer"
                },
                "host": {
                    "$ref": "#/definitions/model.ImageHost"
                },
                "id": {
                    "type": "string"
                },
                "lifecycle": {
                    "type": "integer"
                },
                "listed": {
                    "type": "boolean"
                },
                "name": {
                    "type": "string"
                },
                "owner": {
                    "x-omitempty": true,
                    "$ref": "#/definitions/model.UserPartialModel"
                },
                "tags": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "versions": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/model.EmoteVersionModel"
                    }
                }
            }
        },
        "model.EmotePartialModel": {
            "type": "object",
            "properties": {
                "animated": {
                    "type": "boolean"
                },
                "flags": {
                    "type": "integer"
                },
                "host": {
                    "$ref": "#/definitions/model.ImageHost"
                },
                "id": {
                    "type": "string"
                },
                "lifecycle": {
                    "type": "integer"
                },
                "listed": {
                    "type": "boolean"
                },
                "name": {
                    "type": "string"
                },
                "owner": {
                    "x-omitempty": true,
                    "$ref": "#/definitions/model.UserPartialModel"
                },
                "tags": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                }
            }
        },
        "model.EmoteSetModel": {
            "type": "object",
            "properties": {
                "capacity": {
                    "type": "integer"
                },
                "emote_count": {
                    "type": "integer",
                    "x-omitempty": true
                },
                "emotes": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/model.ActiveEmoteModel"
                    },
                    "x-omitempty": true
                },
                "id": {
                    "type": "string"
                },
                "immutable": {
                    "type": "boolean"
                },
                "name": {
                    "type": "string"
                },
                "origins": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/model.EmoteSetOrigin"
                    },
                    "x-omitempty": true
                },
                "owner": {
                    "x-nullable": true,
                    "$ref": "#/definitions/model.UserPartialModel"
                },
                "privileged": {
                    "type": "boolean"
                },
                "tags": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                }
            }
        },
        "model.EmoteSetOrigin": {
            "type": "object",
            "properties": {
                "id": {
                    "type": "string"
                },
                "slices": {
                    "type": "array",
                    "items": {
                        "type": "integer"
                    }
                },
                "weight": {
                    "type": "integer"
                }
            }
        },
        "model.EmoteSetPartialModel": {
            "type": "object",
            "properties": {
                "capacity": {
                    "type": "integer"
                },
                "id": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "owner": {
                    "x-nullable": true,
                    "$ref": "#/definitions/model.UserPartialModel"
                },
                "tags": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                }
            }
        },
        "model.EmoteVersionModel": {
            "type": "object",
            "properties": {
                "animated": {
                    "type": "boolean"
                },
                "createdAt": {
                    "type": "integer"
                },
                "description": {
                    "type": "string"
                },
                "host": {
                    "x-omitempty": true,
                    "$ref": "#/definitions/model.ImageHost"
                },
                "id": {
                    "type": "string"
                },
                "lifecycle": {
                    "type": "integer"
                },
                "listed": {
                    "type": "boolean"
                },
                "name": {
                    "type": "string"
                }
            }
        },
        "model.ImageFile": {
            "type": "object",
            "properties": {
                "format": {
                    "type": "string"
                },
                "frame_count": {
                    "type": "integer"
                },
                "height": {
                    "type": "integer"
                },
                "name": {
                    "type": "string"
                },
                "size": {
                    "type": "integer"
                },
                "static_name": {
                    "type": "string"
                },
                "width": {
                    "type": "integer"
                }
            }
        },
        "model.ImageHost": {
            "type": "object",
            "properties": {
                "files": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/model.ImageFile"
                    }
                },
                "url": {
                    "type": "string"
                }
            }
        },
        "model.UserConnectionModel": {
            "type": "object",
            "properties": {
                "display_name": {
                    "type": "string"
                },
                "emote_capacity": {
                    "type": "integer"
                },
                "emote_set": {
                    "x-omitempty": true,
                    "$ref": "#/definitions/model.EmoteSetModel"
                },
                "id": {
                    "type": "string"
                },
                "linked_at": {
                    "type": "integer"
                },
                "platform": {
                    "type": "string",
                    "enum": [
                        "TWITCH",
                        "YOUTUBE",
                        "DISCORD"
                    ]
                },
                "user": {
                    "x-omitempty": true,
                    "$ref": "#/definitions/model.UserModel"
                },
                "username": {
                    "type": "string"
                }
            }
        },
        "model.UserEditorModel": {
            "type": "object",
            "properties": {
                "added_at": {
                    "type": "integer"
                },
                "id": {
                    "type": "string"
                },
                "permissions": {
                    "type": "integer"
                },
                "visible": {
                    "type": "boolean"
                }
            }
        },
        "model.UserModel": {
            "type": "object",
            "properties": {
                "avatar_url": {
                    "type": "string"
                },
                "biography": {
                    "type": "string",
                    "x-omitempty": true
                },
                "connections": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/model.UserConnectionModel"
                    }
                },
                "createdAt": {
                    "type": "integer"
                },
                "display_name": {
                    "type": "string"
                },
                "editors": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/model.UserEditorModel"
                    }
                },
                "emote_sets": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/model.EmoteSetPartialModel"
                    },
                    "x-omitempty": true
                },
                "id": {
                    "type": "string"
                },
                "roles": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "style": {
                    "$ref": "#/definitions/model.UserStyle"
                },
                "type": {
                    "type": "string",
                    "enum": [
                        "",
                        "BOT",
                        "SYSTEM"
                    ]
                },
                "username": {
                    "type": "string"
                }
            }
        },
        "model.UserPartialModel": {
            "type": "object",
            "properties": {
                "avatar_url": {
                    "type": "string"
                },
                "connections": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/model.UserConnectionModel"
                    }
                },
                "display_name": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "roles": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "style": {
                    "$ref": "#/definitions/model.UserStyle"
                },
                "type": {
                    "type": "string",
                    "enum": [
                        "",
                        "BOT",
                        "SYSTEM"
                    ]
                },
                "username": {
                    "type": "string"
                }
            }
        },
        "model.UserStyle": {
            "type": "object",
            "properties": {
                "color": {
                    "type": "integer"
                },
                "paint": {
                    "x-nullable": true,
                    "$ref": "#/definitions/model.CosmeticPaintModel"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "3.0",
	Host:             "7tv.io",
	BasePath:         "/v3",
	Schemes:          []string{"http"},
	Title:            "7TV REST API",
	Description:      "This is the REST API for 7TV",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
