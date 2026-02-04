package central

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	jsonFormat string = "json"
	rawFormat  string = "raw"
)

type sensorMessageJSONOutput struct {
	ScenarioStart      string                   `json:"scenario_start"`
	ScenarioEnd        string                   `json:"scenario_end"`
	MessagesFromSensor []*central.MsgFromSensor `json:"messages_from_sensor"`
}

func writeOutputInJSONFormat(messages []*central.MsgFromSensor, start, end time.Time, outfile string) {
	dateFormat := "02.01.15 11:06:39"
	data, err := json.Marshal(&sensorMessageJSONOutput{
		ScenarioStart:      start.Format(dateFormat),
		ScenarioEnd:        end.Format(dateFormat),
		MessagesFromSensor: messages,
	})
	utils.CrashOnError(err)
	utils.CrashOnError(os.WriteFile(outfile, data, 0644))
}

func writeOutputInBinaryFormat(messages []*central.MsgFromSensor, _, _ time.Time, outfile string) {
	file, err := os.OpenFile(outfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer func() {
		utils.CrashOnError(file.Close())
	}()
	utils.CrashOnError(err)
	for _, m := range messages {
		d, err := m.MarshalVT()
		utils.CrashOnError(err)
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(len(d)))
		_, err = file.Write(buf)
		utils.CrashOnError(err)
		_, err = file.Write(d)
		utils.CrashOnError(err)
	}
	if outfile != "/dev/null" {
		utils.CrashOnError(file.Sync())
	}
}

var validFormats = map[string]func([]*central.MsgFromSensor, time.Time, time.Time, string){
	jsonFormat: writeOutputInJSONFormat,
	rawFormat:  writeOutputInBinaryFormat,
}

func IsValidOutputFormat(format string) bool {
	_, ok := validFormats[format]
	return ok
}

func (s *FakeService) DumpAllMessages(start, end time.Time, outfile string, outputFormat string) {
	log.Printf("Dumping all sensor messages to file: %s\n", outfile)
	f, ok := validFormats[outputFormat]
	if !ok {
		log.Fatalf("invalid format '%s'", outputFormat)
	}
	f(s.GetAllMessages(), start, end, outfile)
}
