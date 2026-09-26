package ai

import "fmt"

// blackDigits defines the number of black digits (integer part) in the water meter reading.
const blackDigits = 5

// redDigits defines the number of red digits (fractional part) in the water meter reading.
const redDigits = 2

// expectedDigits defines the total number of digits expected in the water meter reading.
const expectedDigits = blackDigits + redDigits

// answerMarker delimits the final digit answer from the reasoning that precedes it in the model's response.
const answerMarker = "ANSWER:"

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
