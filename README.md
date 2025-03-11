# Rundeck Terraform Provider

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

## Overview

The Rundeck Terraform Provider enables infrastructure automation teams to manage Rundeck resources using HashiCorp Terraform. This provider is maintained by the community in the spirit of open source collaboration, with oversight from Rundeck/PagerDuty staff who review and approve contributions.

## Community Support

This provider is community-supported. While Rundeck/PagerDuty staff review and approve pull requests, new feature development is driven by community contributions. We welcome and encourage community involvement through:

- Bug reports and feature requests via GitHub Issues
- Code contributions via Pull Requests
- Documentation improvements
- Usage questions and discussions

## Documentation

- Provider Usage Documentation: [Terraform Registry](https://registry.terraform.io/providers/rundeck/rundeck/latest/docs)
- Community Discussion: [Google Groups](http://groups.google.com/group/terraform-tool)
- Chat: [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
- [Go](https://golang.org/doc/install) >= 1.18

## Building The Provider

1. Clone the repository
2. Enter the repository directory
3. Build the provider using the Go `install` command:

```sh
$ go install
```

## Development
### Contributing

If you wish to work on the provider, you'll first need [Go](https://www.golang.org) installed on your machine (see Requirements above).

To compile the provider:

Run `go install` - This will build the provider and put the provider binary in the `$GOPATH/bin` directory
To generate or update documentation, run `go generate`

### Testing
To run the full suite of Acceptance tests:

```sh
$ make testacc
```

Note: Acceptance tests create real resources, and often cost money to run.