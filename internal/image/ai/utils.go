package ai

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// blackDigits defines the number of black digits (integer part) in the water meter reading.
const blackDigits = 5

// redDigits defines the number of red digits (fractional part) in the water meter reading.
const redDigits = 2

// expectedDigits defines the total number of digits expected in the water meter reading.
const expectedDigits = blackDigits + redDigits

// answerMarker delimits the final digit answer from the reasoning that precedes it in the model's response.
const answerMarker = "ANSWER:"

// maxResponseTokens defines the maximum number of tokens each provider may generate for a response.
// It must leave enough room for the model to reason about each digit wheel before giving its final
// answer, per the chain-of-thought prompt below. Reasoning-tier models (e.g. OpenAI's o-series/gpt-5
// family) additionally spend hidden reasoning tokens out of this same budget before producing any
// visible output, so this is set generously to avoid truncating the response before it starts.
const maxResponseTokens = 2048

// getPrompt returns the instruction given to the AI model to process the image of the water meter and extract the numeric value.
func getPrompt() string {
	return fmt.Sprintf(
		"Locate the horizontal row of %d digit counter wheels (rollers) on the water meter. "+
			"The first %d wheels (left) show the whole cubic-meter reading in black digits on a white background. "+
			"The last %d wheels (right) show the decimal/liter reading in white digits on a red background. "+
			"Wheels are mechanical and are frequently caught mid-rotation, showing parts of two adjacent digits at once; "+
			"in that case, read the digit that is more fully in frame and aligned with the center line of the neighboring wheels, "+
			"not the digit that is only partially visible at the top or bottom edge. "+
			"Examine each of the %d wheels one at a time, from left to right, and briefly state which digit you see on each one, "+
			"including leading zeros. "+
			"Then, on its own final line, output exactly: %s followed by the %d digits with no spaces, punctuation, or other characters.",
		expectedDigits,
		blackDigits,
		redDigits,
		expectedDigits,
		answerMarker,
		expectedDigits,
	)
}

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
		logrus.Warnf("error parsing float from cleaned result: %v", err)
		return formatted
	}

	return strconv.FormatFloat(val, 'f', -1, 64)
}
