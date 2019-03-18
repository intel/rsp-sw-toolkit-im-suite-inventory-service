package slices

import "testing"

func TestRemoveDuplicates(t *testing.T) {
	dupArray := []string{"sensor1", "sensor2", "sensor1"}
	noDups := RemoveDuplicates(dupArray)
	if len(noDups) != 2 {
		t.Errorf("Failed remove dups")
	}
}

func TestFilter(t *testing.T) {
	array := []string{"sensor1", "sensor2", "sensor3"}
	filterArray := []string{"sensor1", "sensor2"}
	filteredArray := Filter(array, func(v string) bool {
		return !Contains(filterArray, v)
	})

	if len(filteredArray) != 1 {
		t.Errorf("Failed to filter")
	}
	if filteredArray[0] != "sensor3" {
		t.Errorf("Failed to filter")
	}
}
