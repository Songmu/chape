package chapel

import (
	"fmt"
)

type Chapel struct {
	audio string
}

func New(audio string) *Chapel {
	return &Chapel{
		audio: audio,
	}
}

func (c *Chapel) Edit() error {
	// Implement the logic to edit the audio file here.
	return fmt.Errorf("not implemented yet")
}
