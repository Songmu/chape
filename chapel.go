package chapel

import (
	"fmt"
	"io"
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

func (c *Chapel) Apply(input io.Reader) error {
	// Implement the logic to apply changes to the audio file here.
	return fmt.Errorf("not implemented yet")
}
