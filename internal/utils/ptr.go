package utils

func DerefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func DerefBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}

func DerefInt(i *int) int {
	if i != nil {
		return *i
	}
	return 0
}

func DerefInt32(i *int32) int32 {
	if i != nil {
		return *i
	}
	return 0
}
