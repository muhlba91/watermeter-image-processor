package ai

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// maxResponseTokens defines the maximum number of tokens each provider may generate for a response.
// It must leave enough room for the model to reason about each digit wheel before giving its final
// answer, per the chain-of-thought prompt in prompt.go.
const maxResponseTokens = 2048

// nonDigits matches every character that is not a decimal digit.
var nonDigits = regexp.MustCompile(`[^0-9]`)

// cleanResult extracts the reading from the line following the model's final answer marker. It
// rejects the answer unless it has exactly the expected number of digits: a miscounted or malformed
// answer (e.g. a skipped or extra wheel) is an obviously wrong reading that must not be published.
// result: The raw result string obtained from the AI model.
func cleanResult(result string) (string, error) {
	idx := strings.LastIndex(strings.ToUpper(result), answerMarker)
	if idx == -1 {
		return "", fmt.Errorf("no %q marker in model response: %q", answerMarker, result)
	}

	answer, _, _ := strings.Cut(result[idx+len(answerMarker):], "\n")
	digits := nonDigits.ReplaceAllString(answer, "")
	if len(digits) != expectedDigits {
		return "", fmt.Errorf(
			"invalid answer %q: got %d digits, expected %d. response: %q",
			strings.TrimSpace(answer),
			len(digits),
			expectedDigits,
			result,
		)
	}

	formatted := fmt.Sprintf("%s.%s", digits[:blackDigits], digits[blackDigits:])
	val, err := strconv.ParseFloat(formatted, 64)
	if err != nil {
		return "", fmt.Errorf("error parsing float from answer %q: %w", formatted, err)
	}

	return strconv.FormatFloat(val, 'f', -1, 64), nil
}
