terraform {
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "7tv"

    workspaces {
      prefix = "7tv-api-"
    }
  }
}

module "api" {
  source                      = "./api"
  docker_image                = var.api_docker_image
  infra                       = data.terraform_remote_state.infra.outputs
  discord                     = data.terraform_remote_state.discord.outputs
  website_url                 = var.website_url
  cookie_domain               = var.cookie_domain
  proxy_endpoint_url          = var.proxy_endpoint_url
  twitch_client_id            = var.twitch_client_id
  twitch_client_secret        = var.twitch_client_secret
  twitch_redirect_uri         = var.twitch_redirect_uri
  proxy_endpoint_bypass_token = var.proxy_endpoint_bypass_token
  s3_endpoint                 = var.s3_endpoint
  s3_region                   = var.s3_region
  s3_external_bucket          = var.s3_external_bucket
  s3_internal_bucket          = var.s3_internal_bucket
  seventv_domain              = var.seventv_domain
  io_seventv_domain           = var.io_seventv_domain
  discord_client_id           = var.discord_client_id
  discord_client_secret       = var.discord_client_secret
}
