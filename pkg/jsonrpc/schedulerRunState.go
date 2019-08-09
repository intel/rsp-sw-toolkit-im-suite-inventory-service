package jsonrpc

import "github.com/pkg/errors"

type SchedulerRunState struct {
	Notification
	Params SchedulerRunStateParams `json:"params"`
}

type SchedulerRunStateParams struct {
	RunState string `json:"run_state"`
	// currently do not care about the other properties
}

func (notif *SchedulerRunState) Validate() error {
	if notif.Params.RunState == "" {
		return errors.New("missing run_state parameter")
	}

	return notif.Notification.Validate()
}
