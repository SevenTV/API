resource "kubernetes_namespace" "app" {
  metadata {
    name = var.namespace
  }
}

resource "kubernetes_secret" "app" {
  metadata {
    name      = "api"
    namespace = var.namespace
  }

  data = {
    "config.yaml" = templatefile("${path.module}/config.template.yaml", {
      bind                  = "0.0.0.0:3000"
      http_addr             = var.http_addr
      http_port_gql         = var.http_port_gql
      http_port_rest        = var.http_port_rest
      cookie_domain         = local.infra.secondary_zone
      cookie_secure         = true
      cookie_whitelist      = var.cookie_whitelist
      twitch_client_id      = var.twitch_client_id
      twitch_client_secret  = var.twitch_client_secret
      twitch_redirect_uri   = var.twitch_redirect_uri
      discord_client_id     = var.discord_client_id
      discord_client_secret = var.discord_client_secret
      discord_redirect_uri  = var.discord_redirect_uri
      mongo_uri             = local.infra.mongodb_uri
      mongo_username        = local.infra.mongodb_user_app.username
      mongo_password        = local.infra.mongodb_user_app.password
      redis_address         = local.infra.redis_host
      redis_username        = "default"
      redis_password        = local.infra.redis_password
    })
  }
}

resource "kubernetes_deployment" "app" {
  metadata {
    name      = "api"
    namespace = kubernetes_namespace.app.metadata[0].name
    labels = {
      app = "api"
    }
  }

  timeouts {
    create = "4m"
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
          name  = "api"
          image = var.image_url

          port {
            name           = "gql"
            container_port = 3000
            protocol       = "TCP"
          }

          port {
            name           = "rest"
            container_port = 3100
            protocol       = "TCP"
          }

          port {
            name           = "portal"
            container_port = 3200
            protocol       = "TCP"
          }

          port {
            name           = "metrics"
            container_port = 9100
            protocol       = "TCP"
          }

          port {
            name           = "health"
            container_port = 9200
            protocol       = "TCP"
          }

          port {
            name           = "pprof"
            container_port = 9300
            protocol       = "TCP"
          }

          port {
            name           = "eventbridge"
            container_port = 9700
            protocol       = "TCP"
          }

          env {
            name = "API_K8S_POD_NAME"
            value_from {
              field_ref {
                field_path = "metadata.name"
              }
            }
          }

          lifecycle {
            // Pre-stop hook is used to send a fallback signal to the container
            // to gracefully remove all connections ahead of shutdown
            pre_stop {
              exec {
                command = ["sh", "-c", "sleep 5 && echo \"1\" >> shutdown"]
              }
            }
          }

          resources {
            requests = {
              cpu    = "1500m"
              memory = "4Gi"
            }
            limits = {
              cpu    = "1250m"
              memory = "4.25Gi"
            }
          }

          volume_mount {
            name       = "config"
            mount_path = "/app/config.yaml"
            sub_path   = "config.yaml"
          }

          liveness_probe {
            tcp_socket {
              port = "health"
            }
            initial_delay_seconds = 10
            timeout_seconds       = 5
            period_seconds        = 5
            success_threshold     = 2
            failure_threshold     = 4
          }

          readiness_probe {
            tcp_socket {
              port = "health"
            }
            initial_delay_seconds = 10
            timeout_seconds       = 5
            period_seconds        = 5
            success_threshold     = 2
            failure_threshold     = 3
          }

          image_pull_policy = var.image_pull_policy
        }

        volume {
          name = "config"
          secret {
            secret_name = kubernetes_secret.app.metadata[0].name
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "app" {
  metadata {
    name      = "api"
    namespace = kubernetes_namespace.app.metadata[0].name
  }

  spec {
    selector = {
      app = "api"
    }

    port {
      name        = "gql"
      port        = 3000
      target_port = "http"
    }

    port {
      name        = "rest"
      port        = 3100
      target_port = "http"
    }

    port {
      name        = "portal"
      port        = 3200
      target_port = "portal"
    }

    port {
      name        = "metrics"
      port        = 9100
      target_port = "metrics"
    }

    port {
      name        = "health"
      port        = 9200
      target_port = "health"
    }

    port {
      name        = "pprof"
      port        = 9300
      target_port = "pprof"
    }

    port {
      name        = "eventbridge"
      port        = 9700
      target_port = "eventbridge"
    }
  }
}

resource "kubernetes_ingress_v1" "app" {
  metadata {
    name      = "api"
    namespace = kubernetes_namespace.app.metadata[0].name
    annotations = {
      "kubernetes.io/ingress.class"                         = "nginx"
      "external-dns.alpha.kubernetes.io/target"             = local.infra.cloudflare_tunnel_hostname
      "external-dns.alpha.kubernetes.io/cloudflare-proxied" = "true"
    }
  }

  spec {
    rule {
      host = local.infra.secondary_zone
      http {
        // Developer Portal
        path {
          path      = "/"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.app.metadata[0].name
              port {
                name = "portal"
              }
            }
          }
        }

        // GraphQL API - V3
        path {
          path      = "/v3/gql"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.app.metadata[0].name
              port {
                name = "gql"
              }
            }
          }
        }

        // REST API - V3
        path {
          path      = "/v3"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.app.metadata[0].name
              port {
                name = "rest"
              }
            }
          }
        }

        // GraphQL API - V2
        path {
          path      = "/v2/gql"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.app.metadata[0].name
              port {
                name = "gql"
              }
            }
          }
        }

        // REST API - V2
        path {
          path      = "/v2"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.app.metadata[0].name
              port {
                name = "rest"
              }
            }
          }
        }
      }
    }
  }
}

resource "kubernetes_horizontal_pod_autoscaler_v2" "apil" {
  metadata {
    name      = "api"
    namespace = kubernetes_namespace.app.metadata[0].name
  }

  spec {
    scale_target_ref {
      api_version = "apps/v1"
      kind        = "Deployment"
      name        = kubernetes_deployment.app.metadata[0].name
    }

    min_replicas = 2
    max_replicas = 8

    metric {
      type = "Resource"
      resource {
        name = "memory"
        target {
          type                = "Utilization"
          average_utilization = 60
        }
      }
    }

    metric {
      type = "Resource"
      resource {
        name = "cpu"
        target {
          type                = "Utilization"
          average_utilization = 80
        }
      }
    }

    behavior {
      scale_down {
        stabilization_window_seconds = 300
        policies {
          type  = "Pods"
          value = 1
        }
      }

      scale_up {
        stabilization_window_seconds = 120
        policies {
          type  = "Pods"
          value = 1
        }
      }
    }
  }
}
