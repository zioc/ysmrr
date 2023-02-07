package main

import (
	"fmt"
	"time"

	"github.com/zioc/ysmrr"
	"github.com/zioc/ysmrr/pkg/animations"
)

func main() {
	availableAnimations := animations.GetAnimations()

	for _, animation := range availableAnimations {
		manager := ysmrr.NewSpinnerManager(
			ysmrr.WithAnimation(animation),
		)

		_ = manager.AddSpinner(fmt.Sprintf("This is spinner %d...", animation))
		manager.Start()
		time.Sleep(2 * time.Second)
		manager.Stop()
	}
}
