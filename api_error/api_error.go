package api_error

// вы можете использовать ApiError в коде, который получается в результате генерации
// считаем что это какая-то общеизвестная структура
type ApiError struct {
	HTTPStatus int
	Err        error
}

func (ae ApiError) Error() string {
	return ae.Err.Error()
}

// ----------------
