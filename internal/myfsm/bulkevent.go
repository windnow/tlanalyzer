package myfsm

type BulkEvent struct {
	Time     string
	Duration string
	Name     string
	Level    string
	Fields   map[string]string
}

func (e BulkEvent) GetField(fieldName string) *string {
	var result *string
	var buf string

	switch fieldName {
	case "time":
		result = &e.Time
	case "duration":
		result = &e.Duration
	case "name":
		result = &e.Name
	case "level":
		result = &e.Level
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
		e.Time = value
	case "duration":
		e.Duration = value
	case "name":
		e.Name = value
	case "level":
		e.Level = value
	default:
		e.Fields[fieldName] = value
	}
}
