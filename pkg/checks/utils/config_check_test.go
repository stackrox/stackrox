package utils

//
//import (
//	"testing"
//)

//var testConfig = map[string]ConfigParams{}
//
//func getTestConfig() (FlattenedConfig, error) {
//	return FlattenedConfig(testConfig), nil
//}
//
//func addToTestConfig(k, v string) {
//	testConfig[k] = append(testConfig[k], v)
//}
//
//func TestCommandCheck(t *testing.T) {
//	cc := CommandCheck{
//		Name:        "name",
//		Description: "description",
//
//		Field:        "field",
//		Default:      "false",
//		EvalFunc:     Matches,
//		DesiredValue: "true",
//
//		ConfigGetter: getTestConfig,
//	}
//	//addToTestConfig("field", "false")
//	res := cc.Run()
//	log.Print(res)
//}
