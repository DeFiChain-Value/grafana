package notifier

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	apimodels "github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
	"golang.org/x/sync/errgroup"
)

const (
	maxTestReceiversWorkers = 10
)

type TestReceiversResult struct {
	Receivers []TestReceiverResult
	NotifedAt time.Time
}

type TestReceiverResult struct {
	Name    string
	Configs []TestReceiverConfigResult
}

type TestReceiverConfigResult struct {
	Name   string
	UID    string
	Status string
	Error  error
}

type InvalidReceiverError struct {
	Receiver *apimodels.PostableGrafanaReceiver
	Err      error
}

func (e InvalidReceiverError) Error() string {
	return fmt.Sprintf("the receiver is invalid: %s", e.Err)
}

type ReceiverTimeoutError struct {
	Receiver *apimodels.PostableGrafanaReceiver
	Err      error
}

func (e ReceiverTimeoutError) Error() string {
	return fmt.Sprintf("the receiver timed out: %s", e.Err)
}

func (am *Alertmanager) TestReceivers(ctx context.Context, c apimodels.TestReceiversConfigParams) (*TestReceiversResult, error) {
	// now represents the start time of the test
	now := time.Now()
	testAlert := &types.Alert{
		Alert: model.Alert{
			Labels: model.LabelSet{
				model.LabelName("alertname"): "TestAlert",
				model.LabelName("instance"):  "Grafana",
			},
			Annotations: model.LabelSet{
				model.LabelName("summary"): "Notification test",
			},
			StartsAt: now,
		},
		UpdatedAt: now,
	}

	// we must set a group key that is unique per test as some receivers use this key to deduplicate alerts
	ctx = notify.WithGroupKey(ctx, testAlert.Labels.String()+now.String())

	tmpl, err := am.getTemplate()
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// job contains all metadata required to test a receiver
	type job struct {
		Config       *apimodels.PostableGrafanaReceiver
		ReceiverName string
		Notifier     notify.Notifier
	}

	// result contains the receiver that was tested and an error that is non-nil if the test failed
	type result struct {
		Config       *apimodels.PostableGrafanaReceiver
		ReceiverName string
		Error        error
	}

	invalid := make([]result, 0, len(c.Receivers))
	jobs := make([]job, 0, len(c.Receivers))

	for _, receiver := range c.Receivers {
		for _, next := range receiver.GrafanaManagedReceivers {
			n, err := am.buildReceiverIntegration(next, tmpl)
			if err != nil {
				invalid = append(invalid, result{
					Config:       next,
					ReceiverName: receiver.Name,
					Error:        err,
				})
			} else {
				jobs = append(jobs, job{
					Config:       next,
					ReceiverName: receiver.Name,
					Notifier:     n,
				})
			}
		}
	}

	numWorkers := maxTestReceiversWorkers
	if numWorkers > len(jobs) {
		numWorkers = len(jobs)
	}

	resultCh := make(chan result, len(jobs))
	workCh := make(chan job, len(jobs))
	for _, job := range jobs {
		workCh <- job
	}
	close(workCh)

	g, ctx := errgroup.WithContext(ctx)
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for next := range workCh {
				v := result{
					Config:       next.Config,
					ReceiverName: next.ReceiverName,
				}
				if _, err := next.Notifier.Notify(ctx, testAlert); err != nil {
					v.Error = err
				}
				resultCh <- v
			}
			return nil
		})
	}
	g.Wait() // nolint
	close(resultCh)

	// m keeps track of the results for each of the receivers
	m := make(map[string]TestReceiverResult)
	for _, receiver := range c.Receivers {
		// set up the result for this receiver
		m[receiver.Name] = TestReceiverResult{
			Name: receiver.Name,
			// A Grafana receiver can have multiple nested receivers
			Configs: make([]TestReceiverConfigResult, 0, len(receiver.GrafanaManagedReceivers)),
		}
	}
	for _, next := range invalid {
		v := m[next.ReceiverName]
		v.Configs = append(v.Configs, TestReceiverConfigResult{
			Name:   next.Config.Name,
			UID:    next.Config.UID,
			Status: "failed",
			Error:  next.Error,
		})
		m[next.ReceiverName] = v
	}
	for next := range resultCh {
		v := m[next.ReceiverName]
		status := "ok"
		if next.Error != nil {
			status = "failed"
		}
		v.Configs = append(v.Configs, TestReceiverConfigResult{
			Name:   next.Config.Name,
			UID:    next.Config.UID,
			Status: status,
			Error:  processNotifierError(next.Config, next.Error),
		})
		m[next.ReceiverName] = v
	}

	results := TestReceiversResult{
		Receivers: make([]TestReceiverResult, 0, len(c.Receivers)),
		NotifedAt: now,
	}
	for _, next := range m {
		results.Receivers = append(results.Receivers, next)
	}
	return &results, nil
}

func processNotifierError(config *apimodels.PostableGrafanaReceiver, err error) error {
	if err == nil {
		return nil
	}

	var urlError *url.Error
	if errors.As(err, &urlError) {
		if urlError.Timeout() {
			return ReceiverTimeoutError{
				Receiver: config,
				Err:      err,
			}
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return ReceiverTimeoutError{
			Receiver: config,
			Err:      err,
		}
	}

	return err
}
