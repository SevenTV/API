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
}
