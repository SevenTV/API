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
                        "description": "OK"
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
        "/users/{user.id}/presences": {
            "post": {
                "description": "Update user presence",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "users"
                ],
                "summary": "Update User Presence",
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
                            "$ref": "#/definitions/model.PresenceModel"
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
                        "description": "OK"
                    }
                }
            }
        }
    },
    "definitions": {
        "model.ActiveEmoteFlagModel": {
            "type": "integer",
            "enum": [
                1,
                65536,
                131072,
                262144
            ],
            "x-enum-comments": {
                "ActiveEmoteFlagModelOverrideBetterTTV": "262144 - Overrides BetterTTV emotes with the same name",
                "ActiveEmoteFlagModelOverrideTwitchGlobal": "65536 - Overrides Twitch Global emotes with the same name",
                "ActiveEmoteFlagModelOverrideTwitchSubscriber": "131072 - Overrides Twitch Subscriber emotes with the same name",
                "ActiveEmoteFlagModelZeroWidth": "1 - Emote is zero-width"
            },
            "x-enum-varnames": [
                "ActiveEmoteFlagModelZeroWidth",
                "ActiveEmoteFlagModelOverrideTwitchGlobal",
                "ActiveEmoteFlagModelOverrideTwitchSubscriber",
                "ActiveEmoteFlagModelOverrideBetterTTV"
            ]
        },
        "model.ActiveEmoteModel": {
            "type": "object",
            "properties": {
                "actor_id": {
                    "type": "string",
                    "x-nullable": true
                },
                "data": {
                    "allOf": [
                        {
                            "$ref": "#/definitions/model.EmotePartialModel"
                        }
                    ],
                    "x-nullable": true
                },
                "flags": {
                    "$ref": "#/definitions/model.ActiveEmoteFlagModel"
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
        "model.CosmeticBadgeModel": {
            "type": "object",
            "properties": {
                "host": {
                    "$ref": "#/definitions/model.ImageHost"
                },
                "id": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "tag": {
                    "type": "string"
                },
                "tooltip": {
                    "type": "string"
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
        "model.CosmeticPaintFunction": {
            "type": "string",
            "enum": [
                "LINEAR_GRADIENT",
                "RADIAL_GRADIENT",
                "URL"
            ],
            "x-enum-varnames": [
                "CosmeticPaintFunctionLinearGradient",
                "CosmeticPaintFunctionRadialGradient",
                "CosmeticPaintFunctionImageURL"
            ]
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
                    "enum": [
                        "LINEAR_GRADIENT",
                        "RADIAL_GRADIENT",
                        "URL"
                    ],
                    "allOf": [
                        {
                            "$ref": "#/definitions/model.CosmeticPaintFunction"
                        }
                    ]
                },
                "id": {
                    "type": "string"
                },
                "image_url": {
                    "type": "string"
                },
                "name": {
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
        "model.EmoteFlagsModel": {
            "type": "integer",
            "enum": [
                1,
                2,
                256,
                65536,
                131072,
                262144,
                16777216
            ],
            "x-enum-comments": {
                "EmoteFlagsAuthentic": "The emote was verified to be an original creation by the uploader",
                "EmoteFlagsContentEdgy": "Edgy or distasteful, may be offensive to some users",
                "EmoteFlagsContentEpilepsy": "Rapid flashing",
                "EmoteFlagsContentSexual": "Sexually Suggesive",
                "EmoteFlagsContentTwitchDisallowed": "Not allowed specifically on the Twitch platform",
                "EmoteFlagsPrivate": "The emote is private and can only be accessed by its owner, editors and moderators",
                "EmoteFlagsZeroWidth": "The emote is recommended to be enabled as Zero-Width"
            },
            "x-enum-varnames": [
                "EmoteFlagsPrivate",
                "EmoteFlagsAuthentic",
                "EmoteFlagsZeroWidth",
                "EmoteFlagsContentSexual",
                "EmoteFlagsContentEpilepsy",
                "EmoteFlagsContentEdgy",
                "EmoteFlagsContentTwitchDisallowed"
            ]
        },
        "model.EmoteLifecycleModel": {
            "type": "integer",
            "enum": [
                -1,
                0,
                1,
                2,
                3,
                -2
            ],
            "x-enum-varnames": [
                "EmoteLifecycleDeleted",
                "EmoteLifecyclePending",
                "EmoteLifecycleProcessing",
                "EmoteLifecycleDisabled",
                "EmoteLifecycleLive",
                "EmoteLifecycleFailed"
            ]
        },
        "model.EmoteModel": {
            "type": "object",
            "properties": {
                "animated": {
                    "type": "boolean"
                },
                "flags": {
                    "$ref": "#/definitions/model.EmoteFlagsModel"
                },
                "host": {
                    "$ref": "#/definitions/model.ImageHost"
                },
                "id": {
                    "type": "string"
                },
                "lifecycle": {
                    "$ref": "#/definitions/model.EmoteLifecycleModel"
                },
                "listed": {
                    "type": "boolean"
                },
                "name": {
                    "type": "string"
                },
                "owner": {
                    "allOf": [
                        {
                            "$ref": "#/definitions/model.UserPartialModel"
                        }
                    ],
                    "x-omitempty": true
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
                    "$ref": "#/definitions/model.EmoteFlagsModel"
                },
                "host": {
                    "$ref": "#/definitions/model.ImageHost"
                },
                "id": {
                    "type": "string"
                },
                "lifecycle": {
                    "$ref": "#/definitions/model.EmoteLifecycleModel"
                },
                "listed": {
                    "type": "boolean"
                },
                "name": {
                    "type": "string"
                },
                "owner": {
                    "allOf": [
                        {
                            "$ref": "#/definitions/model.UserPartialModel"
                        }
                    ],
                    "x-omitempty": true
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
                    "allOf": [
                        {
                            "$ref": "#/definitions/model.UserPartialModel"
                        }
                    ],
                    "x-nullable": true
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
                    "allOf": [
                        {
                            "$ref": "#/definitions/model.UserPartialModel"
                        }
                    ],
                    "x-nullable": true
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
                    "allOf": [
                        {
                            "$ref": "#/definitions/model.ImageHost"
                        }
                    ],
                    "x-omitempty": true
                },
                "id": {
                    "type": "string"
                },
                "lifecycle": {
                    "$ref": "#/definitions/model.EmoteLifecycleModel"
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
                    "$ref": "#/definitions/model.ImageFormat"
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
        "model.ImageFormat": {
            "type": "string",
            "enum": [
                "AVIF",
                "WEBP"
            ],
            "x-enum-varnames": [
                "ImageFormatAVIF",
                "ImageFormatWEBP"
            ]
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
        "model.PresenceKind": {
            "type": "integer",
            "enum": [
                0,
                1,
                2
            ],
            "x-enum-varnames": [
                "UserPresenceKindUnknown",
                "UserPresenceKindChannel",
                "UserPresenceKindWebPage"
            ]
        },
        "model.PresenceModel": {
            "type": "object",
            "properties": {
                "id": {
                    "type": "string"
                },
                "kind": {
                    "$ref": "#/definitions/model.PresenceKind"
                },
                "timestamp": {
                    "type": "integer"
                },
                "ttl": {
                    "type": "integer"
                },
                "user_id": {
                    "type": "string"
                }
            }
        },
        "model.UserConnectionModel": {
            "type": "object",
            "properties": {
                "display_name": {
                    "description": "The display name of the user on the platform.",
                    "type": "string"
                },
                "emote_capacity": {
                    "description": "The maximum size of emote sets that may be bound to this connection.",
                    "type": "integer"
                },
                "emote_set": {
                    "description": "The emote set that is linked to this connection",
                    "allOf": [
                        {
                            "$ref": "#/definitions/model.EmoteSetModel"
                        }
                    ],
                    "x-nullable": true
                },
                "emote_set_id": {
                    "description": "The ID of the emote set bound to this connection.",
                    "type": "string",
                    "x-nullable": true
                },
                "id": {
                    "type": "string"
                },
                "linked_at": {
                    "description": "The time when the user linked this connection",
                    "type": "integer"
                },
                "platform": {
                    "description": "The service of the connection.",
                    "type": "string",
                    "enum": [
                        "TWITCH",
                        "YOUTUBE",
                        "DISCORD"
                    ]
                },
                "presences": {
                    "description": "A list of users active in the channel",
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/model.UserPartialModel"
                    },
                    "x-omitempty": true
                },
                "user": {
                    "description": "App data for the user",
                    "allOf": [
                        {
                            "$ref": "#/definitions/model.UserModel"
                        }
                    ],
                    "x-omitempty": true
                },
                "username": {
                    "description": "The username of the user on the platform.",
                    "type": "string"
                }
            }
        },
        "model.UserConnectionPartialModel": {
            "type": "object",
            "properties": {
                "display_name": {
                    "description": "The display name of the user on the platform.",
                    "type": "string"
                },
                "emote_capacity": {
                    "description": "The maximum size of emote sets that may be bound to this connection.",
                    "type": "integer"
                },
                "emote_set_id": {
                    "description": "The emote set that is linked to this connection",
                    "type": "string",
                    "x-nullable": true
                },
                "id": {
                    "type": "string"
                },
                "linked_at": {
                    "description": "The time when the user linked this connection",
                    "type": "integer"
                },
                "platform": {
                    "description": "The service of the connection.",
                    "type": "string",
                    "enum": [
                        "TWITCH",
                        "YOUTUBE",
                        "DISCORD"
                    ]
                },
                "username": {
                    "description": "The username of the user on the platform.",
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
                    },
                    "x-omitempty": true
                },
                "created_at": {
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
                        "$ref": "#/definitions/model.UserConnectionPartialModel"
                    },
                    "x-omitempty": true
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
                "badge": {
                    "allOf": [
                        {
                            "$ref": "#/definitions/model.CosmeticBadgeModel"
                        }
                    ],
                    "x-omitempty": true
                },
                "badge_id": {
                    "type": "string",
                    "x-omitempty": true
                },
                "color": {
                    "type": "integer",
                    "x-omitempty": true
                },
                "paint": {
                    "allOf": [
                        {
                            "$ref": "#/definitions/model.CosmeticPaintModel"
                        }
                    ],
                    "x-omitempty": true
                },
                "paint_id": {
                    "type": "string",
                    "x-omitempty": true
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
