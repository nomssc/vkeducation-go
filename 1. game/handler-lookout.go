package main

/*
 * TRANSPORT LAYER
 *
 * Responsible for user
 * input and game output,
 * routing and validation.
 */

// Lookout Route

type LookoutUsecaseInterface interface{}

type LookoutHandler struct {
	Usecase LookoutUsecaseInterface
}

func (l *LookoutHandler) LookoutHandlerFunc(c *Command) (*Result, error) {
	return &Result{}, nil
}

/*
 * SERVICE LAYER
 *
 * Responsible for business logic.
 */

// Lookout Logic

type LookoutUsecase struct{}

func (l *LookoutUsecase) LookoutUsecaseFunc() {

}
