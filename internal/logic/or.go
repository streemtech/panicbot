package logic

func Or(b1, b2 bool) bool {
	if !b1 && !b2 {
		return false
	}
	return true
}
