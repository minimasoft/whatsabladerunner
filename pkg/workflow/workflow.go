package workflow

import (
	"context"
	"fmt"
)

// Step defines a single unit of work in a workflow.
type Step interface {
	Execute(ctx context.Context, input interface{}) (interface{}, error)
	Name() string
}

// Workflow represents a sequence of steps.
type Workflow struct {
	Name  string
	Steps []Step
}

// NewWorkflow creates a new workflow.
func NewWorkflow(name string, steps ...Step) *Workflow {
	return &Workflow{
		Name:  name,
		Steps: steps,
	}
}

// Execute runs the workflow steps sequentially.
func (w *Workflow) Execute(ctx context.Context, initialInput interface{}) (interface{}, error) {
	currentInput := initialInput
	var err error

	fmt.Printf("[Workflow: %s] Starting\n", w.Name)

	for _, step := range w.Steps {
		// Check for cancellation before starting step
		select {
		case <-ctx.Done():
			fmt.Printf("[Workflow: %s] Cancelled before step %s\n", w.Name, step.Name())
			return nil, ctx.Err()
		default:
		}

		fmt.Printf("[Workflow: %s] Running step: %s\n", w.Name, step.Name())
		currentInput, err = step.Execute(ctx, currentInput)
		if err != nil {
			fmt.Printf("[Workflow: %s] Step %s failed: %v\n", w.Name, step.Name(), err)
			return nil, err
		}
	}

	fmt.Printf("[Workflow: %s] Completed\n", w.Name)
	return currentInput, nil
}
