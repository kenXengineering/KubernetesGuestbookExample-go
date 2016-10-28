# Kubernetes Guestbook Example in Golan

This repo is and implementation of the [Kubernetes Guestbook Example](https://github.com/kubernetes/kubernetes/tree/master/examples/guestbook-go) using Go.

* The [V1](v1/) folder is an implementation using Replication Controllers and Services.
* The [V2](v2/) folder is an implementaiton using Deployments and Services. (Currently not working)

### Running

From either folder, run `go run main.go` to execute or `go build` to create a binary.  It expects the Kubernetes `config` file to be present in the same directoy as the binary.  It will not prompt for any input and will start creation of the example right away.  It uses the `default` namespace for all items.

If it runs into any errors, it will print out the error message and continue executing.  This is a very simple and dumb app that is more of and example on how to use the `client-go` library then anything actually usefull. 