terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.7.0"
    }

    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "2.18.1"
    }

    random = {
      source = "hashicorp/random"
    }
  }
}

provider "aws" {
  region = var.region
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.cluster.endpoint
  token                  = data.aws_eks_cluster_auth.cluster.token
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
}

data "aws_eks_cluster" "cluster" {
  name = local.infra_workspace_name
}

data "aws_eks_cluster_auth" "cluster" {
  name = local.infra_workspace_name
}
