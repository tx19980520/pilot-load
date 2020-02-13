package simulation

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"
)

type Args struct {
	PilotAddress string
}

func Simple(a Args) error {
	numWorkloads := 1
	ns := NewNamespace(NamespaceSpec{
		Name: "workload",
	})
	sa := NewServiceAccount(ServiceAccountSpec{
		Namespace: ns.Spec.Name,
		Name:      "default",
	})

	scaler := NewScaler(ScalerSpec{
		scaler: func(ctx Context, n int) error {
			if n < numWorkloads {
				log.Println("cannot scale down yet")
				return nil
			}
			log.Println("Scaling workloads", numWorkloads, "->", n)
			newSims := []Simulation{}
			for n > numWorkloads {
				numWorkloads++
				w := NewWorkload(WorkloadSpec{
					App:            fmt.Sprintf("app-%d", numWorkloads),
					Node:           "node",
					Namespace:      ns.Spec.Name,
					ServiceAccount: sa.Spec.Name,
					Instances:      1,
					Scaling: &ScalerSpec{
						start:    1,
						step:     1,
						stop:     10,
						interval: time.Second * 3,
					},
				})
				newSims = append(newSims, w)
			}

			return NewAggregateSimulation(nil, newSims).Run(ctx)
		},
		start:    0,
		step:     1,
		stop:     100,
		interval: time.Second * 1,
	})

	sim := NewAggregateSimulation([]Simulation{ns, sa}, []Simulation{scaler})
	if err := ExecuteSimulations(a, sim); err != nil {
		log.Println("waiting for deletions because of error: ", err)
		time.Sleep(time.Second * 10)
		return fmt.Errorf("error executing: %v", err)
	}
	return nil
}

func ExecuteSimulations(a Args, simulation Simulation) error {
	ctx, cancel := context.WithCancel(context.Background())
	go captureTermination(ctx, cancel)
	return simulation.Run(Context{ctx, a})
}

func captureTermination(ctx context.Context, cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
	}()
	select {
	case <-c:
		cancel()
	case <-ctx.Done():
	}
}