package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/upgrader/config"
	"github.com/stackrox/rox/sensor/upgrader/snapshot"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

var (
	log = logging.LoggerForModule()
)

func main() {
	log.Infof("StackRox Sensor Upgrader, version %s", version.GetMainVersion())

	upgraderCfg, err := config.Create()
	utils.Must(err)

	upgradeCtx, err := upgradectx.Create(upgraderCfg)
	utils.Must(err)

	objs, err := snapshot.TakeOrReadSnapshot(upgradeCtx)
	utils.Must(err)

	encoder := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)
	for _, obj := range objs {
		var strW strings.Builder
		utils.Must(encoder.Encode(obj, &strW))
		fmt.Println(strW.String())
		fmt.Println("---")
	}
	time.Sleep(10 * time.Second)
}
