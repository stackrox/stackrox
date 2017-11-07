package uuid

//
//import "testing"
//
//const (
//	id = `b455a167-2302-4d37-b41e-f1b4092da5e9`
//)
//
//func TestFromString(t *testing.T) {
//	first := FromStringOrNil(id)
//	second := FromStringOrNil(id)
//
//	if first != second {
//		t.Errorf("Identical UUID were not equal; %s; %s", first, second)
//	}
//
//	idMap := make(map[UUID]bool)
//
//	idMap[first] = true
//
//	if _, found := idMap[second]; !found {
//		t.Errorf("Couldn't find UUID, %s, in map", second)
//	}
//}
