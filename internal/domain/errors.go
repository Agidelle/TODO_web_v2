package domain

import "errors"

var (
	ErrID             = errors.New("некорректный id")
	ErrBadTitle       = errors.New("не указан заголовок задачи")
	ErrDate           = errors.New("неправильный формат даты")
	ErrInternalServer = errors.New("внутренняя ошибка сервера")
)

type CustomError struct {
	Code       int
	Err        error
	ErrStorage error
}

func NewCustomError(code int, err error, errStorage error) *CustomError {
	return &CustomError{
		Code:       code,
		Err:        err,
		ErrStorage: errStorage,
	}
}
