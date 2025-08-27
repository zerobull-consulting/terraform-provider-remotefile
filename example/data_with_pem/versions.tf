terraform {
  required_providers {
    remotefile = {
      source  = "zerobull-consulting/remotefile"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "3.0.0"
    }
  }
}
