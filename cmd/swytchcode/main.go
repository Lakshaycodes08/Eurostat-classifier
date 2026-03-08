// main is the CLI entrypoint; it delegates to the cli package to run the swytchcode kernel.
package main

import "gitlab.com/swytchcode/cli/internal/cli"

func main() {
	cli.Execute()
}

