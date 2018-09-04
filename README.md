# My CNI plugin

Playing around with hacked CNI-plugins. 

## Objectives

1. Add Liz Rice's [Container from scratch](https://github.com/lizrice/containers-from-scratch) to a new network namespace and configure the network via a modified CNI-plugin. Lightning talk at Gophercon [slides](https://docs.google.com/presentation/d/16kJz9k3l9jyLk6v0y0FMgPkXWa7rJadJ6v9nKbzAScQ/edit?usp=sharing).
2. Deploy a modified CNI-plugin in a Kubernetes cluster to auto-provision IPv6 addresses.
...

## Resources

- [CNI-Plugins](https://github.com/containernetworking/plugins)

### TODO

Find a cloud provider that allows IPv6 subnetting.

- [IPv6 Subnet Routing to EC2 Instance](https://forums.aws.amazon.com/thread.jspa?messageID=799319#799319)
- [GCP does not support IPv6 destination ranges](https://cloud.google.com/vpc/docs/routes#individualroutes)
