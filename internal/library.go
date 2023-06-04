package internal

type ArrayFlags []string

func (i *ArrayFlags) String() string {
	return ""
}

func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)

	return nil
}

func CastInterfaceToMap(input interface{}) map[string]interface{} {
	return input.(map[string]interface{})
}

func CastInterfaceToList(input interface{}) []interface{} {
	return input.([]interface{})
}
