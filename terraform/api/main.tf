resource "linode_object_storage_key" "api" {
  label = "7tv-${trimprefix(terraform.workspace, "7tv-api-")}-api-access"

  bucket_access {
    bucket_name = var.s3_external_bucket
    cluster     = var.s3_region
    permissions = "read_write"
  }

  bucket_access {
    bucket_name = var.s3_internal_bucket
    cluster     = var.s3_region
    permissions = "read_write"
  }
}

resource "kubernetes_namespace" "api" {
  metadata {
    name = var.namespace
  }
}

resource "kubernetes_secret" "api" {
  metadata {
    name      = "api"
    namespace = kubernetes_namespace.api.metadata[0].name
  }
  data = {
    "config.yaml" = templatefile("${path.module}/config.yaml", {
      website_url = var.website_url
      cookie_domain = var.cookie_domain
      proxy_endpoint_bypass_token = var.proxy_endpoint_bypass_token
      proxy_endpoint_url = var.proxy_endpoint_url
      twitch_client_id = var.twitch_client_id
      twitch_client_secret = var.twitch_client_secret
      twitch_redirect_uri = var.twitch_redirect_uri
      discord_api = var.discord.hostname
      discord_client_id = var.discord_client_id
      discord_client_secret = var.discord_client_secret
      mongo_db = "seventv"
      mongo_uri = "mongodb://root:${var.infra.mongo_password}@${var.infra.mongo_host}/?authSource=admin&readPreference=secondaryPreferred"
      s3_access_token = linode_object_storage_key.api.access_key
      s3_secret_key = linode_object_storage_key.api.secret_key
      s3_region = var.s3_region
      s3_endpoint = var.s3_endpoint
      s3_internal_bucket = var.s3_internal_bucket
      s3_external_bucket = var.s3_external_bucket
      rmq_uri = "ampq://user:${var.infra.rmq_password}@${var.infra.rmq_host}/seventv"
      redis_address = var.infra.redis_host
      redis_password = var.infra.redis_password
    })
  }
}

resource "kubernetes_deployment" "api" {
  metadata {
    name = "api"
    labels = {
      app = "api"
    }
    namespace = kubernetes_namespace.api.metadata[0].name
  }

  timeouts {
    create = "2m"
    update = "2m"
    delete = "2m"
  }

  spec {
    selector {
      match_labels = {
        app = "api"
      }
    }
    template {
      metadata {
        labels = {
          app = "api"
        }
      }
      spec {
        container {
          name              = "api"
          image             = var.docker_image
          image_pull_policy = "Always"
          port {
            container_port = 3000
            name           = "gql"
          }
          port {
            container_port = 3100
            name           = "rest"
          }
          port {
            container_port = 3200
            name           = "portal"
          }
          port {
            container_port = 9100
            name           = "metrics"
          }
          port {
            container_port = 9300
            name           = "pprof"
          }
          port {
            container_port = 9500
            name           = "health"
          }
          readiness_probe {
            initial_delay_seconds = 5
            period_seconds        = 5
            tcp_socket {
              port = "health"
            }
          }
          liveness_probe {
            initial_delay_seconds = 5
            period_seconds        = 5
            tcp_socket {
              port = "health"
            }
          }
          startup_probe {
            initial_delay_seconds = 5
            period_seconds        = 5
            tcp_socket {
              port = "health"
            }
          }
          security_context {
            allow_privilege_escalation = false
            privileged                 = false
            read_only_root_filesystem  = true
            run_as_non_root            = true
            run_as_user                = 1000
            run_as_group               = 1000
            capabilities {
              drop = ["ALL"]
            }
          }
          resources {
            limits = {
              "cpu"    = "1000m"
              "memory" = "1.5Gi"
            }
            requests = {
              "cpu"    = "750m"
              "memory" = "1.5Gi"
            }
          }
          volume_mount {
            name       = "api"
            mount_path = "/app/config.yaml"
            sub_path   = "config.yaml"
            read_only  = true
          }
        }
        volume {
          name = "api"
          secret {
            secret_name  = kubernetes_secret.api.metadata[0].name
            default_mode = "0644"
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "api" {
  metadata {
    name      = "api"
    namespace = kubernetes_namespace.api.metadata[0].name
  }

  spec {
    selector = {
      app = "api"
    }
    port {
      name = "gql"
      port = 3000
      target_port = "gql"
    }
    port {
      name = "rest"
      port = 3100
      target_port = "rest"
    }
    port {
      name = "portal"
      port = 3200
      target_port = "portal"
    }
    port {
      name = "metrics"
      port = 9100
      target_port = "metrics"
    }
    port {
      name = "pprof"
      port = 9300
      target_port = "pprof"
    }
    port {
      name = "health"
      port = 9500
      target_port = "health"
    }
  }
}

resource "kubernetes_ingress_v1" "api" {
  metadata {
    name      = "api"
    namespace = kubernetes_namespace.api.metadata[0].name
    annotations = {
      "external-dns.alpha.kubernetes.io/hostname" = "${var.io_seventv_domain},${var.seventv_domain}"
      "kubernetes.io/ingress.class"               = "nginx"
      "cert-manager.io/cluster-issuer"            = "cloudflare"
    }
  }
  spec {
    rule {
      host = var.seventv_domain
      http {
        path {
          path      = "/"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.api.metadata[0].name
              port {
                name = "portal"
              }
            }
          }
        }
        path {
          path      = "/v3/gql"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.api.metadata[0].name
              port {
                name = "gql"
              }
            }
          }
        }
        path {
          path      = "/v2/gql"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.api.metadata[0].name
              port {
                name = "gql"
              }
            }
          }
        }
        path {
          path      = "/v3"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.api.metadata[0].name
              port {
                name = "rest"
              }
            }
          }
        }
        path {
          path      = "/v2"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.api.metadata[0].name
              port {
                name = "rest"
              }
            }
          }
        }
      }
    }
    rule {
      host = var.io_seventv_domain
      http {
        path {
          path      = "/"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.api.metadata[0].name
              port {
                name = "portal"
              }
            }
          }
        }
        path {
          path      = "/v3/gql"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.api.metadata[0].name
              port {
                name = "gql"
              }
            }
          }
        }
        path {
          path      = "/v2/gql"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.api.metadata[0].name
              port {
                name = "gql"
              }
            }
          }
        }
        path {
          path      = "/v3"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.api.metadata[0].name
              port {
                name = "rest"
              }
            }
          }
        }
        path {
          path      = "/v2"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.api.metadata[0].name
              port {
                name = "rest"
              }
            }
          }
        }
      }
    }
    tls {
      hosts       = [var.seventv_domain, var.io_seventv_domain]
      secret_name = "api-tls"
    }
  }
}

resource "kubernetes_horizontal_pod_autoscaler_v2" "api" {
  metadata {
    name      = "api"
    namespace = kubernetes_namespace.api.metadata[0].name
  }

  spec {
    max_replicas = 15
    min_replicas = 3
    scale_target_ref {
      api_version = "apps/v1"
      kind        = "Deployment"
      name        = kubernetes_deployment.api.metadata[0].name
    }
    metric {
      type = "Resource"
      resource {
        name = "memory"
        target {
          type                = "Utilization"
          average_utilization = 75
        }
      }
    }
    metric {
      type = "Resource"
      resource {
        name = "cpu"
        target {
          type                = "Utilization"
          average_utilization = 75
        }
      }
    }
  }
}
