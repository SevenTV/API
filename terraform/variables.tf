variable "api_docker_image" {
  type    = string
  default = "ghcr.io/seventv/api:latest"
}

data "terraform_remote_state" "infra" {
  backend = "remote"

  config = {
    organization = "7tv"
    workspaces = {
      name = "7tv-infra-${trimprefix(terraform.workspace, "7tv-api-")}"
    }
  }
}

data "terraform_remote_state" "discord" {
  backend = "remote"

  config = {
    organization = "7tv"
    workspaces = {
      name = "7tv-discord-${trimprefix(terraform.workspace, "7tv-api-")}"
    }
  }
}

variable "linode_api_token" {
  type      = string
  sensitive = true
}

variable "website_url" {
  type = string
}

variable "cookie_domain" {
  type = string
}

variable "proxy_endpoint_bypass_token" {
  type      = string
  sensitive = true
}

variable "proxy_endpoint_url" {
  type      = string
  sensitive = true
}

variable "twitch_client_id" {
  type      = string
  sensitive = true
}

variable "twitch_client_secret" {
  type      = string
  sensitive = true
}

variable "twitch_redirect_uri" {
  type = string
}

variable "s3_region" {
  type      = string
  sensitive = true
}

variable "s3_external_bucket" {
  type      = string
  sensitive = true
}

variable "s3_internal_bucket" {
  type      = string
  sensitive = true
}

variable "s3_endpoint" {
  type      = string
  sensitive = true
}

variable "seventv_domain" {
  type = string
}

variable "io_seventv_domain" {
  type = string
}

variable "discord_client_id" {
  type = string
  sensitive   = true
}

variable "discord_client_secret" {
  type = string
  sensitive   = true
}
