package sensor

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/metrics"
)

type Processor interface {
	Process(ctx context.Context, msg *central.MsgToSensor)
}

type StopableProcessor interface {
	Processor
	Stop()
}

type componentsProcessor struct {
	receivers []common.SensorComponent
	msgChan   chan *central.MsgToSensor
	cancel    context.CancelFunc
}

func NewProcessor(receivers ...common.SensorComponent) *componentsProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	p := &componentsProcessor{
		cancel:    cancel,
		receivers: receivers,
		msgChan:   make(chan *central.MsgToSensor),
	}
	p.start(ctx)
	return p

}

func (p *componentsProcessor) start(ctx context.Context) {
	componentsNames := make([]string, len(p.receivers))
	for _, r := range p.receivers {
		componentsNames = append(componentsNames, r.Name())
	}
	msgChan := make(chan *central.MsgToSensor)
	componentsQueues := sendToAll(ctx, msgChan, componentsNames)
	for _, receiver := range p.receivers {
		go process(ctx, componentsQueues[receiver.Name()], receiver)
	}

	// Start periodic queue size updates
	queueSizeTicker := time.NewTicker(5 * time.Second)
	defer queueSizeTicker.Stop()
	go func() {
		for {
			select {
			case <-queueSizeTicker.C:
				for componentName, ch := range componentsQueues {
					metrics.SetCentralReceiverComponentQueueSize(componentName, len(ch))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (p *componentsProcessor) Process(ctx context.Context, msg *central.MsgToSensor) {
	select {
	case <-ctx.Done():
		log.Infof("Dropping message to sensor components: %s", msg.GetMsg())
	case p.msgChan <- msg:
	}
}

func (p *componentsProcessor) Stop() {
	close(p.msgChan)
	p.cancel()
}

func sendToAll(ctx context.Context, msgChan <-chan *central.MsgToSensor, componentNames []string) map[string]<-chan *central.MsgToSensor {
	componentsQueues := make(map[string]chan *central.MsgToSensor, len(componentNames))
	returnQueues := make(map[string]<-chan *central.MsgToSensor, len(componentsQueues))
	for _, n := range componentNames {
		metrics.SetCentralReceiverComponentQueueSize(n, 0)
		ch := make(chan *central.MsgToSensor, 10)
		returnQueues[n], componentsQueues[n] = ch, ch
	}

	go func() {
		localWg := &sync.WaitGroup{}
		defer func() {
			localWg.Wait()
			for _, ch := range componentsQueues {
				close(ch)
			}
		}()
		for msg := range msgChan {
			localWg.Add(len(componentsQueues))
			for name, ch := range componentsQueues {
				ctx, cancel := context.WithTimeout(ctx, time.Second)
				go func() {
					defer cancel()
					defer localWg.Done()
					sendStart := time.Now()
					select {
					case <-ctx.Done():
						log.Infof("Context %s for %s, not multiplexing messages. Dropping %s", ctx.Err(), name, msg.String())
						metrics.IncrementCentralReceiverMessagesDropped(name, "timeout")
						return
					case ch <- msg:
						metrics.ObserveCentralReceiverChannelSendDuration(name, time.Since(sendStart))
					}
				}()
			}
		}

	}()

	return returnQueues
}

func process(ctx context.Context, ch <-chan *central.MsgToSensor, r common.SensorComponent) {
	for {
		select {
		case msg := <-ch:
			start := time.Now()
			if err := r.ProcessMessage(ctx, msg); err != nil {
				log.Errorf("%s: %+v", r.Name(), err)
			}
			metrics.ObserveCentralReceiverProcessMessageDuration(r.Name(), time.Since(start))
		case <-ctx.Done():
			return
		}
	}
}
