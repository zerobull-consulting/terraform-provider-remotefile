data "remotefile_sftp" "retrieve_etc_hostname" {
  host        = "your.hostname.tld"
  user        = "default"
  private_key = tls_private_key.sftp.private_key_pem
  path        = "/etc/hostname"
}

resource "tls_private_key" "sftp" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

output "hostname" {
  value = trimspace(data.remotefile_sftp.retrieve_etc_hostname.contents)
}
