package mqttmatch

import "strings"

// Match checks whether the filter and topic match
func Match(filter, topic string) bool {
	filterArray := strings.Split(filter, "/")
	topicArray := strings.Split(topic, "/")
	for i := 0; i < len(filterArray); i++ {
		if i >= len(topicArray) {
			return false
		}
		if filterArray[i] == "#" {
			return true
		}
		if filterArray[i] != "+" && filterArray[i] != topicArray[i] {
			return false
		}
	}

	return len(filterArray) == len(topicArray)
}
