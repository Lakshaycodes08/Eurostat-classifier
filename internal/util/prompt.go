// prompt.go provides interactive selection for setup commands (init, get); not used by exec.
package util

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Select prompts the user to select from a list of options.
// Returns the selected option index (0-based) and the selected value.
// If the user enters an invalid choice, it returns an error.
func Select(prompt string, options []string) (int, string, error) {
	if len(options) == 0 {
		return -1, "", fmt.Errorf("no options provided")
	}

	if prompt != "" {
		fmt.Println(prompt)
	}
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}
	fmt.Print("Select [1-", len(options), "]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return -1, "", err
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil {
		return -1, "", fmt.Errorf("invalid input: %w", err)
	}

	if choice < 1 || choice > len(options) {
		return -1, "", fmt.Errorf("choice %d is out of range (1-%d)", choice, len(options))
	}

	return choice - 1, options[choice-1], nil
}

// SelectWithRetry prompts the user with retry logic.
// Keeps prompting until a valid selection is made.
func SelectWithRetry(prompt string, options []string) (int, string) {
	for {
		idx, val, err := Select(prompt, options)
		if err == nil {
			return idx, val
		}
		fmt.Printf("Error: %v. Please try again.\n\n", err)
	}
}
