package myfsm

type BulkEvent struct {
	time     string
	duration string
	name     string
	level    string
	Fields   map[string]string
}

func (e BulkEvent) GetField(fieldName string) *string {
	var result *string
	var buf string

	switch fieldName {
	case "time":
		result = &e.time
	case "duration":
		result = &e.duration
	case "name":
		result = &e.name
	case "level":
		result = &e.level
	default:
		val, ok := e.Fields[fieldName]
		if ok {
			result = &val
		}
	}

	if result != nil {
		buf = *result
		result = &buf
	}
	return result
}

func (e *BulkEvent) SetField(fieldName, value string) {

	switch fieldName {
	case "time":
		e.time = value
	case "duration":
		e.duration = value
	case "name":
		e.name = value
	case "level":
		e.level = value
	default:
		e.Fields[fieldName] = value
	}
}
