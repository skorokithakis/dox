package runtime

import (
	"testing"
)

func TestContainerOptions(t *testing.T) {
	opts := ContainerOptions{
		Image:       "test:latest",
		Command:     []string{"echo", "hello"},
		Env:         []string{"TEST=value"},
		Volumes:     []string{"/host:/container"},
		WorkingDir:  "/workspace",
		User:        "1000:1000",
		Interactive: true,
		TTY:         true,
		Remove:      true,
		Network:     "host",
	}
	
	if opts.Image != "test:latest" {
		t.Errorf("Image = %s, want test:latest", opts.Image)
	}
	
	if len(opts.Command) != 2 {
		t.Errorf("len(Command) = %d, want 2", len(opts.Command))
	}
	
	if opts.Command[0] != "echo" {
		t.Errorf("Command[0] = %s, want echo", opts.Command[0])
	}
}