# Terraform Provider 

Attempting to learn how to write a custom provider by. The goal is to have a provider create a storage account on either AWS,Azure, or GCP.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

