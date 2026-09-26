package ai

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// maxResponseTokens defines the maximum number of tokens each provider may generate for a response.
// It must leave enough room for the model to reason about each digit wheel before giving its final
// answer, per the chain-of-thought prompt in prompt.go.
const maxResponseTokens = 2048

// cleanResult takes the raw result from the AI model and extracts the relevant digits, ensuring that the output is in the expected format for water meter readings.
// result: The raw result string obtained from the AI model, which may contain extraneous characters and formatting.
func cleanResult(result string) string {
	answer := result
	if idx := strings.LastIndex(strings.ToUpper(result), answerMarker); idx != -1 {
		answer = result[idx+len(answerMarker):]
	}

	re := regexp.MustCompile(`[^0-9]`)
	digits := re.ReplaceAllString(answer, "")

	if len(digits) != expectedDigits {
		logrus.Errorf(
			"unexpected result length: got %d digits, expected %d. result: '%s'",
			len(digits),
			expectedDigits,
			result,
		)
		return digits
	}

	formatted := fmt.Sprintf("%s.%s", digits[:blackDigits], digits[blackDigits:])
	val, err := strconv.ParseFloat(formatted, 64)
	if err != nil {
		logrus.Errorf("error parsing float from cleaned result: %v", err)
		return formatted
	}

	return strconv.FormatFloat(val, 'f', -1, 64)
}
