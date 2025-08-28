data "remotefile_sftp" "retrieve_etc_hostname" {
  host        = "your.hostname.tld"
  user        = "default"
  private_key = <<EOK
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACD7eCj38B+jSu55qM04tX5zlnMqKyLfshMXVfj7lqr80AAAALid44kzneOJ
MwAAAAtzc2gtZWQyNTUxOQAAACD7eCj38B+jSu55qM04tX5zlnMqKyLfshMXVfj7lqr80A
AAAEDgKipn5m912IXu2Iyn9vAUcvLhlJ/N9yfv2n/yN2ysRft4KPfwH6NK7nmozTi1fnOW
cyorIt+yExdV+PuWqvzQAAAAM2RldmVsb3BlckB6ZXJvYnVsbC10ZXJyYWZvcm0tcmVtb3
RlZmlsZS1kZXZlbG9wbWVudAEC
-----END OPENSSH PRIVATE KEY-----
EOK
  path        = "/etc/hostname"
}

output "hostname" {
  value = trimspace(data.remotefile_sftp.retrieve_etc_hostname.contents)
}