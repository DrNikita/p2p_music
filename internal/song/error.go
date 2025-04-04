package song

type PromoteSongError struct {
	errMsg string
}

func (e PromoteSongError) Error() string {
	return e.errMsg
}
