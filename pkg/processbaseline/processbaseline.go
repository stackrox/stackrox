package processbaseline

import (
	"fmt"
	"strings"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/stringutils"
)

type keyPrefix string

const (
	deploymentContainerKeyPrefix keyPrefix = "DC"
)

var (
	genDuration = env.BaselineGenerationDuration.DurationSetting()
)

func KeyToID(key *storage.ProcessBaselineKey) (string, error) {
	if stringutils.AllNotEmpty(key.GetClusterId(), key.GetNamespace(), key.GetDeploymentId(), key.GetContainerName()) {
		return fmt.Sprintf("%s:%s:%s:%s:%s", deploymentContainerKeyPrefix, key.GetClusterId(), key.GetNamespace(), key.GetDeploymentId(), key.GetContainerName()), nil
	}
	return "", fmt.Errorf("invalid key %+v: doesn't match any of our known patterns", key)
}

// IDToKey converts a string process baseline key to its proto object.
func IDToKey(id string) (*storage.ProcessBaselineKey, error) {
	if strings.HasPrefix(id, string(deploymentContainerKeyPrefix)) {
		keys := strings.Split(id, ":")
		if len(keys) == 5 {
			resKey := &storage.ProcessBaselineKey{
				ClusterId:     keys[1],
				Namespace:     keys[2],
				DeploymentId:  keys[3],
				ContainerName: keys[4],
			}

			return resKey, nil
		}
	}

	return nil, fmt.Errorf("invalid id %s: doesn't match any of our known patterns", id)
}

// BaselineItemFromProcess returns what we baseline for a given process.
// It exists to make sure that we're using the same thing in every place (name vs execfilepath).
func BaselineItemFromProcess(process *storage.ProcessIndicator) string {
	return process.GetSignal().GetExecFilePath()
}

func makeElementMap(elementList []*storage.BaselineElement) map[string]*storage.BaselineElement {
	elementMap := make(map[string]*storage.BaselineElement, len(elementList))
	for _, listItem := range elementList {
		elementMap[listItem.GetElement().GetProcessName()] = listItem
	}
	return elementMap
}

func makeElementList(elementMap map[string]*storage.BaselineElement) []*storage.BaselineElement {
	elementList := make([]*storage.BaselineElement, 0, len(elementMap))
	for _, process := range elementMap {
		elementList = append(elementList, process)
	}
	return elementList
}

// Change the name of this function
func AddAndRemoveElementsFromBaseline(baseline *storage.ProcessBaseline, addElements []*storage.BaselineItem, removeElements []*storage.BaselineItem, auto bool) *storage.ProcessBaseline {
	baselineMap := makeElementMap(baseline.GetElements())
	graveyardMap := makeElementMap(baseline.GetElementGraveyard())

	for _, element := range addElements {
		// Don't automatically add anything which has been previously removed
		if _, ok := graveyardMap[element.GetProcessName()]; auto && ok {
			continue
		}
		existing, ok := baselineMap[element.GetProcessName()]
		if !ok || existing.Auto {
			delete(graveyardMap, element.GetProcessName())
			baselineMap[element.GetProcessName()] = &storage.BaselineElement{
				Element: element,
				Auto:    auto,
			}
		}
	}

	for _, removeElement := range removeElements {
		delete(baselineMap, removeElement.GetProcessName())
		existing, ok := graveyardMap[removeElement.GetProcessName()]
		if !ok || existing.Auto {
			graveyardMap[removeElement.GetProcessName()] = &storage.BaselineElement{
				Element: removeElement,
				Auto:    auto,
			}
		}
	}

	baseline.Elements = makeElementList(baselineMap)
	baseline.ElementGraveyard = makeElementList(graveyardMap)

	return baseline
}

func BaselineFromKeysItemsAndExistingBaseline(baseline *storage.ProcessBaseline, key *storage.ProcessBaselineKey, addElements []*storage.BaselineItem, auto bool, lock bool, user_lock bool) (*storage.ProcessBaseline, error) {
	timestamp := protocompat.TimestampNow()
	id, err := KeyToID(key)

	if err != nil {
		return nil, err
	}

	if baseline != nil {
		AddAndRemoveElementsFromBaseline(baseline, addElements, nil, auto)
	} else {
		var elements []*storage.BaselineElement
		for _, element := range addElements {
			elements = append(elements, &storage.BaselineElement{Element: &storage.BaselineItem{Item: &storage.BaselineItem_ProcessName{ProcessName: element.GetProcessName()}}, Auto: auto})
		}

		baseline = &storage.ProcessBaseline{
			Key:      key,
			Elements: elements,
			Created:  timestamp,
		}
	}

	baseline.Id = id
	baseline.LastUpdate = timestamp

	if lock {
		baseline.StackRoxLockedTimestamp = timestamp
	} else {
		lockTime := GenerateLockTimestamp()
		baseline.StackRoxLockedTimestamp = protocompat.ConvertTimeToTimestampOrNil(&lockTime)
	}

	if user_lock && baseline.UserLockedTimestamp == nil {
		baseline.UserLockedTimestamp = timestamp
	}

	return baseline, nil
}

func GenerateLockTimestamp() time.Time {
	lockTimestamp := time.Now().Add(genDuration)
	_, err := protocompat.ConvertTimeToTimestampOrError(lockTimestamp)
	// This should not occur unless genDuration is in a bad state.  If that happens just
	// set it to one hour in the future.
	if err != nil {
		lockTimestamp = time.Now().Add(1 * time.Hour)
	}
	return lockTimestamp
}
