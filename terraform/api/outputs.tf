output "hostname" {
  value = "${kubernetes_service.api.metadata[0].name}.${var.namespace}.svc.cluster.local"
}

output "bridge_url" {
  value = "https://${kubernetes_service.api.metadata[0].name}.${var.namespace}.svc.cluster.local:9700"
}
