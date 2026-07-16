package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	client := NewClient("http://localhost:8080")

	switch cmd {
	case "api-resources":
		cmdAPIResources(client)
	case "api-versions":
		cmdAPIVersions(client)
	case "plugins":
		cmdPlugins(client)
	case "get":
		cmdGet(client, args)
	case "create":
		cmdCreate(client, args)
	case "delete":
		cmdDelete(client, args)
	case "apply":
		cmdApply(client, args)
	case "explain":
		cmdExplain(client, args)
	case "watch":
		cmdWatch(client, args)
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	usage := `apictl - CLI for the dynamic API server

USAGE:
  apictl <command> [options]

COMMANDS:
  api-resources        List all available resources
  api-versions         List all API versions
  plugins              List loaded plugins and count
  get <resource>       List all objects of a resource type
  get <resource> <id>  Get a specific object
  create -f <file>     Create a resource from a file
  delete <resource> <id> Delete a resource
  apply -f <file>      Apply a CRD or create/update a resource
  explain <resource>   Show resource schema
  watch <resource>     Stream events for a resource

EXAMPLES:
  apictl api-resources
  apictl get users
  apictl get users user1
  apictl create -f invoice.json
  apictl apply -f invoice-crd.yaml
  apictl delete invoices invoice-1
  apictl explain invoices
  apictl watch orders
`
	fmt.Print(usage)
}
