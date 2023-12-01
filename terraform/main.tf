terraform {
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "7tv"

    workspaces {
      prefix = "seventv-api-"
    }
  }
}

locals {
  infra_workspace_name = replace(terraform.workspace, "api", "infra")
  infra                = data.terraform_remote_state.infra.outputs
  image_url            = var.image_url != null ? var.image_url : format("ghcr.io/seventv/api:%s-latest", trimprefix(terraform.workspace, "seventv-api-"))
  s3 = var.s3 != null ? var.s3 : {
    region          = local.infra.region
    ak              = local.infra.s3_access_key.id
    sk              = local.infra.s3_access_key.secret
    internal_bucket = local.infra.s3_bucket.internal
    public_bucket   = local.infra.s3_bucket.public
  }
}
