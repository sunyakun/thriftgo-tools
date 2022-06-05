package example

import (
	"context"

	"github.com/sunyakun/thriftgo-tools/example/httpgen/example"
)

type ExampleService struct{}

func NewExampleService() *ExampleService {
	return &ExampleService{}
}

func (e *ExampleService) Get(ctx context.Context, request *example.GetExampleRequest) (r *example.Example, err error) {
	return &example.Example{
		ID:      1,
		Name:    "foo",
		Address: "bar",
		Age:     18,
	}, nil
}

func (e *ExampleService) Create(ctx context.Context, request *example.CreateExampleRequest) (r *example.Example, err error) {
	return &example.Example{
		ID:      1,
		Name:    "foo",
		Address: "bar",
		Age:     18,
	}, nil
}
