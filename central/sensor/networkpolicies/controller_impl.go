package networkpolicies

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

type controller struct {
	stopSig concurrency.ReadOnlyErrorSignal

	returnChans      map[int64]chan *central.NetworkPoliciesResponse_Payload
	returnChansMutex sync.Mutex

	currSeqID int64

	injector common.MessageInjector
}

func newController(injector common.MessageInjector, stopSig concurrency.ReadOnlyErrorSignal) *controller {
	return &controller{
		stopSig:     stopSig,
		returnChans: make(map[int64]chan *central.NetworkPoliciesResponse_Payload),
		injector:    injector,
	}
}

func (c *controller) ApplyNetworkPolicies(ctx context.Context, mod *storage.NetworkPolicyModification) (*storage.NetworkPolicyModification, error) {
	seqID := atomic.AddInt64(&c.currSeqID, 1)

	applyID := uuid.NewV4().String()

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_NetworkPoliciesCommand{
			NetworkPoliciesCommand: &central.NetworkPoliciesCommand{
				SeqId: seqID,
				Payload: &central.NetworkPoliciesCommand_Payload{
					Cmd: &central.NetworkPoliciesCommand_Payload_Apply{
						Apply: &central.NetworkPoliciesCommand_Apply{
							ApplyId:      applyID,
							Modification: mod,
						},
					},
				},
			},
		},
	}

	retC := make(chan *central.NetworkPoliciesResponse_Payload, 1)
	concurrency.WithLock(&c.returnChansMutex, func() {
		c.returnChans[seqID] = retC
	})
	defer concurrency.WithLock(&c.returnChansMutex, func() {
		delete(c.returnChans, seqID)
	})

	if err := c.injector.InjectMessage(ctx, msg); err != nil {
		return nil, errors.Wrap(err, "could not send network policies modification")
	}

	var resp *central.NetworkPoliciesResponse_Payload

	select {
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "context error")
	case <-c.stopSig.Done():
		return nil, errors.Wrap(c.stopSig.Err(), "lost connection to sensor")
	case resp = <-retC:
	}

	if errProto := resp.GetError(); errProto != nil {
		return nil, fmt.Errorf("sensor returned error: %s", errProto.GetMessage())
	}

	if resp.GetApply() == nil {
		return nil, fmt.Errorf("sensor returned an invalid apply of type %T", resp.GetCmd())
	}
	if resp.GetApply().GetApplyId() != applyID {
		return nil, fmt.Errorf("sensor returned response with an invalid apply id (got %q, expected %q)", resp.GetApply().GetApplyId(), applyID)
	}

	return resp.GetApply().GetUndoModification(), nil
}

func (c *controller) ProcessNetworkPoliciesResponse(resp *central.NetworkPoliciesResponse) error {
	seqID := resp.GetSeqId()
	retC := concurrency.WithLock1(&c.returnChansMutex, func() chan *central.NetworkPoliciesResponse_Payload {
		retC := c.returnChans[seqID]
		delete(c.returnChans, seqID)
		return retC
	})

	if retC == nil {
		return fmt.Errorf("could not dispatch response: no return channel registered for sequence id %d", seqID)
	}

	select {
	case <-c.stopSig.Done():
		return errors.Wrap(c.stopSig.Err(), "sensor connection stopped while waiting for network policies response")
	case retC <- resp.GetPayload():
		return nil
	}
}
