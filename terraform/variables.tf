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

variable "s3" {
  type = object({
    endpoint        = string
    region          = string
    ak              = string
    sk              = string
    internal_bucket = string
    public_bucket   = string
  })
  nullable = true
  default  = null
}

variable "nats_events_subject" {
  type    = string
  default = ""
}
