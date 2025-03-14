package sensor

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/messagestream/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type bufferedStreamSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	innerStream *mocks.MockSensorMessageStream
}

func TestBufferedStream(t *testing.T) {
	suite.Run(t, new(bufferedStreamSuite))
}

func (s *bufferedStreamSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.innerStream = mocks.NewMockSensorMessageStream(s.ctrl)
}

func (s *bufferedStreamSuite) Test_NoBuffer() {
	msgC := make(chan *central.MsgFromSensor)
	defer close(msgC)
	stopSignal := concurrency.NewErrorSignal()
	stream, errC := NewBufferedStream(s.innerStream, msgC, &stopSignal)
	s.Require().NotNil(stream)
	s.Assert().Nil(errC)

	s.innerStream.EXPECT().Send(gomock.Any()).Times(2).DoAndReturn(func(msg *central.MsgFromSensor) error {
		return handleExpectSend(msg)
	})

	s.Assert().Error(stream.Send(nil))
	s.Assert().Nil(stream.Send(&central.MsgFromSensor{}))
}

func (s *bufferedStreamSuite) Test_BufferedStream() {
	msgC := make(chan *central.MsgFromSensor, 1)
	defer close(msgC)
	stopSignal := concurrency.NewErrorSignal()
	stream, errC := NewBufferedStream(s.innerStream, msgC, &stopSignal)
	s.Require().NotNil(stream)
	s.Require().NotNil(errC)

	// This is triggered when the inner stream Send function is called
	messageReadSignal := make(chan struct{})
	defer close(messageReadSignal)
	// This allows us to control when the inner stream Send function returns
	finishSendSignal := make(chan struct{})
	defer close(finishSendSignal)

	s.innerStream.EXPECT().Send(gomock.Any()).Times(4).DoAndReturn(func(msg *central.MsgFromSensor) error {
		// Send was called in the inner stream
		messageReadSignal <- struct{}{}
		// Wait for the finishSendSignal to be triggered
		assertSignalIsTriggered(s.T(), finishSendSignal, "timeout waiting for the finishSendSignal to be triggered")
		return handleExpectSend(msg)
	})

	s.Assert().Nil(stream.Send(&central.MsgFromSensor{}))
	assertSignalIsTriggered(s.T(), messageReadSignal, "timeout waiting for the message to be read")

	// This message will be buffered because we already read once, thus the buffer is empty
	s.Assert().Nil(stream.Send(&central.MsgFromSensor{}))
	// These messages will be dropped because the finishSendSignal has not been triggered yet,
	// thus the Send function is blocking the reads to the buffer
	s.Assert().Nil(stream.Send(&central.MsgFromSensor{}))
	s.Assert().Nil(stream.Send(&central.MsgFromSensor{}))
	s.Assert().Len(msgC, 1)
	finishSendSignal <- struct{}{}
	select {
	case err := <-errC:
		s.Assert().Nil(err)
	case <-time.After(500 * time.Millisecond):
		s.FailNow("timeout waiting for message to be sent")
	}

	assertSignalIsTriggered(s.T(), messageReadSignal, "timeout waiting for the message to be read")

	finishSendSignal <- struct{}{}
	select {
	case err := <-errC:
		s.Assert().Nil(err)
	case <-time.After(500 * time.Millisecond):
		s.FailNow("timeout waiting for message to be sent")
	}
	s.Assert().Len(msgC, 0)
	s.Assert().Nil(stream.Send(nil)) // nil message will return an error in the inner stream

	assertSignalIsTriggered(s.T(), messageReadSignal, "timeout waiting for the message to be read")

	// This message will be buffered because we already read once, thus the buffer is empty
	s.Assert().Nil(stream.Send(&central.MsgFromSensor{}))
	// These messages will be dropped because the finishSendSignal has not been triggered yet,
	// thus the Send function is blocking the reads to the buffer
	s.Assert().Nil(stream.Send(&central.MsgFromSensor{}))
	s.Assert().Nil(stream.Send(&central.MsgFromSensor{}))
	s.Assert().Len(msgC, 1)
	finishSendSignal <- struct{}{}
	select {
	case err := <-errC:
		s.Assert().Error(err)
	case <-time.After(500 * time.Millisecond):
		s.FailNow("timeout waiting for message to be sent")
	}

	assertSignalIsTriggered(s.T(), messageReadSignal, "timeout waiting for the message to be read")

	finishSendSignal <- struct{}{}
	select {
	case err := <-errC:
		s.Assert().Nil(err)
	case <-time.After(500 * time.Millisecond):
		s.FailNow("timeout waiting for message to be sent")
	}
	s.Assert().Len(msgC, 0)
}

func (s *bufferedStreamSuite) Test_Stop() {
	msgC := make(chan *central.MsgFromSensor, 1)
	defer close(msgC)
	stopSignal := concurrency.NewErrorSignal()
	stream, errC := NewBufferedStream(s.innerStream, msgC, &stopSignal)
	s.Require().NotNil(stream)
	s.Require().NotNil(errC)

	// This is triggered when the inner stream Send function is called
	messageReadSignal := make(chan struct{})
	defer close(messageReadSignal)

	// Most of the time, this should be called once
	// but if the select chooses to write in errC instead of the stopC,
	// We might reach Send again thus EXPECT AnyTimes.
	s.innerStream.EXPECT().Send(gomock.Any()).AnyTimes().DoAndReturn(func(msg *central.MsgFromSensor) error {
		// Send was called in the inner stream
		messageReadSignal <- struct{}{}
		return handleExpectSend(msg)
	})

	s.Assert().Nil(stream.Send(&central.MsgFromSensor{}))
	assertSignalIsTriggered(s.T(), messageReadSignal, "timeout waiting for the message to be read")
	s.Assert().Nil(stream.Send(&central.MsgFromSensor{}))
	s.Assert().Len(msgC, 1)

	// At this point we buffered stream should be locked writing in errC
	// Trigger the stop signal
	stopSignal.Signal()

	s.Assert().Eventually(func() bool {
		select {
		case _, ok := <-errC:
			// At this point we are reading from errC.
			// Since stopC and errC are active, the select will choose one randomly.
			// We shouldn't fail the test if the channel is not closed immediately.
			return ok == false
		case <-time.After(500 * time.Millisecond):
			return false
		}
	}, 500*time.Millisecond, 10*time.Millisecond, "timeout waiting for the stream to stop")

}

func handleExpectSend(msg *central.MsgFromSensor) error {
	if msg == nil {
		return errors.New("msg is nil")
	}
	return nil
}

func assertSignalIsTriggered(t *testing.T, signal chan struct{}, errMsg string) {
	select {
	case <-signal:
	case <-time.After(500 * time.Millisecond):
		t.Error(errMsg)
		t.FailNow()
	}
}
