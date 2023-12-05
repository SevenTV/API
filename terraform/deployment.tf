data "kubernetes_namespace" "app" {
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
      website_url           = "https://${local.infra.primary_zone}"
      cdn_url               = var.cdn_url
      http_addr             = var.http_addr
      http_port_gql         = var.http_port_gql
      http_port_rest        = var.http_port_rest
      cookie_domain         = local.infra.secondary_zone
      cookie_secure         = true
      twitch_client_id      = var.twitch_client_id
      twitch_client_secret  = var.twitch_client_secret
      twitch_redirect_uri   = "https://${local.infra.secondary_zone}/v3/auth?platform=twitch&callback=true"
      discord_client_id     = var.discord_client_id
      discord_client_secret = var.discord_client_secret
      discord_redirect_uri  = var.discord_redirect_uri
      discord_api           = "http://compactdisc:3000"
      discord_channels      = yamlencode([])
      kick_challenge_token  = var.kick_challenge_token
      mongo_uri             = local.infra.mongodb_uri
      mongo_username        = local.infra.mongodb_user_app.username
      mongo_password        = local.infra.mongodb_user_app.password
      mongo_database        = "7tv"
      meili_url             = "http://meilisearch.database.svc.cluster.local:7700"
      meili_key             = var.meilisearch_key
      meili_index           = "emotes"
      nats_url              = "nats.database.svc.cluster.local:4222"
      nats_subject          = var.nats_events_subject
      redis_username        = "default"
      redis_password        = local.infra.redis_password
      rmq_uri               = local.infra.rabbitmq_uri
      s3_region             = local.s3.region
      s3_access_key         = local.s3.ak
      s3_secret_key         = local.s3.sk
      s3_internal_bucket    = local.s3.internal_bucket
      s3_public_bucket      = local.s3.public_bucket
      s3_endpoint           = local.s3.endpoint != null ? local.s3.endpoint : ""
      jwt_secret            = var.credentials_jwt_secret
    })
  }
}

resource "kubernetes_deployment" "app" {
  metadata {
    name      = "api"
    namespace = data.kubernetes_namespace.app.metadata[0].name
    labels    = {
      app = "api"
    }
  }

  lifecycle {
    replace_triggered_by = [kubernetes_secret.app]
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

    strategy {
      type = "RollingUpdate"
      rolling_update {
        max_surge       = "0"
        max_unavailable = "1"
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
          image = local.image_url

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

          resources {
            requests = {
              cpu    = local.infra.production ? "1000m" : "100m"
              memory = local.infra.production ? "4.5Gi" : "600Mi"
            }
            limits = {
              cpu    = local.infra.production ? "1000m" : "100m"
              memory = local.infra.production ? "4.5Gi" : "600Mi"
            }
          }

          volume_mount {
            name       = "config"
            mount_path = "/app/config.yaml"
            sub_path   = "config.yaml"
          }

          liveness_probe {
            http_get {
              port = "health"
              path = "/"
            }
            initial_delay_seconds = 10
            timeout_seconds       = 5
            period_seconds        = 5
            success_threshold     = 1
            failure_threshold     = 4
          }

          readiness_probe {
            http_get {
              port = "health"
              path = "/"
            }
            initial_delay_seconds = 10
            timeout_seconds       = 5
            period_seconds        = 5
            success_threshold     = 1
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
    namespace = data.kubernetes_namespace.app.metadata[0].name
  }

  spec {
    selector = {
      app = "api"
    }

    port {
      name        = "gql"
      port        = 3000
      target_port = "gql"
    }

    port {
      name        = "rest"
      port        = 3100
      target_port = "rest"
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
    name        = "api"
    namespace   = data.kubernetes_namespace.app.metadata[0].name
    annotations = {
      "kubernetes.io/ingress.class"                         = "nginx"
      "external-dns.alpha.kubernetes.io/target"             = local.infra.cloudflare_tunnel_hostname.regular
      "external-dns.alpha.kubernetes.io/cloudflare-proxied" = "true"
      "nginx.ingress.kubernetes.io/limit-connections" : "64"
      "nginx.ingress.kubernetes.io/proxy-body-size" : "7m"
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
      }
    }
  }
}

resource "kubernetes_horizontal_pod_autoscaler_v2" "api" {
  metadata {
    name      = "api"
    namespace = data.kubernetes_namespace.app.metadata[0].name
  }

  spec {
    scale_target_ref {
      api_version = "apps/v1"
      kind        = "Deployment"
      name        = kubernetes_deployment.app.metadata[0].name
    }

    min_replicas = 2
    max_replicas = 14

    metric {
      type = "Resource"
      resource {
        name = "cpu"
        target {
          type                = "Utilization"
          average_utilization = 50
        }
      }
    }

    behavior {
      scale_down {
        stabilization_window_seconds = 300
        select_policy                = "Min"
        policy {
          type           = "Pods"
          value          = 1
          period_seconds = 15
        }
      }

      scale_up {
        stabilization_window_seconds = 120
        select_policy                = "Min"
        policy {
          type           = "Pods"
          value          = 1
          period_seconds = 15
        }
      }
    }
  }
}
