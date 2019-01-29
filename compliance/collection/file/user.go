package file

import (
	"io/ioutil"
	"strconv"
	"strings"
)

var (
	groupMap = make(map[uint32]string)
	userMap  = make(map[uint32]string)
)

func parseSystemUserMap(file string, userMap map[uint32]string) {
	contents, err := ioutil.ReadFile(containerPath(file))
	if err != nil {
		log.Error(err)
		return
	}
	lines := strings.Split(string(contents), "\n")
	for _, l := range lines {
		if spl := strings.Split(l, ":"); len(spl) > 2 {
			ui, err := strconv.ParseUint(spl[2], 10, 32)
			if err != nil {
				log.Error(err)
				continue
			}
			userMap[uint32(ui)] = spl[0]
		}
	}
}

func init() {
	parseSystemUserMap("/etc/passwd", userMap)
	parseSystemUserMap("/etc/group", groupMap)
}
