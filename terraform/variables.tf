data "terraform_remote_state" "infra" {
  backend = "remote"

  config = {
    organization = "7tv"
    workspaces = {
      name = local.infra_workspace_name
    }
  }
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-2"
}

variable "namespace" {
  type    = string
  default = "app"
}

variable "production" {
  description = "Whether or not to scale resources to a production state"
  type        = bool
  default     = false
}

variable "image_url" {
  type     = string
  nullable = true
  default  = null
}

variable "image_pull_policy" {
  type    = string
  default = "Always"
}

variable "cdn_url" {
  type = string
}

variable "http_addr" {
  type    = string
  default = "0.0.0.0"
}

variable "http_port_gql" {
  type    = number
  default = 3000
}

variable "http_port_rest" {
  type    = number
  default = 3100
}

variable "twitch_client_id" {
  type    = string
  default = ""
}

variable "twitch_client_secret" {
  type    = string
  default = ""
}

variable "twitch_redirect_uri" {
  type    = string
  default = ""
}

variable "discord_client_id" {
  type    = string
  default = ""
}

variable "discord_client_secret" {
  type    = string
  default = ""
}

variable "discord_redirect_uri" {
  type    = string
  default = ""
}
