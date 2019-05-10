package rundeck

type NotFoundError struct{}

func (err NotFoundError) Error() string {
	return "not found"
}
