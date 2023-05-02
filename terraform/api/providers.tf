terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "2.18.1"
    }
    random = {
      source  = "hashicorp/random"
      version = "3.4.3"
    }
    linode = {
      source  = "linode/linode"
      version = "1.30.0"
    }
  }
}
