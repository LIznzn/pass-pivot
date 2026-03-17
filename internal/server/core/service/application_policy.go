package service

func TokenTypesContain(values []string, expected string) bool {
	for _, item := range values {
		if item == expected {
			return true
		}
	}
	return false
}

func AppGrantTypesContain(values []string, expected string) bool {
	for _, item := range values {
		if item == expected {
			return true
		}
	}
	return false
}

func AppTokenTypesContain(values []string, expected string) bool {
	return TokenTypesContain(values, expected)
}
