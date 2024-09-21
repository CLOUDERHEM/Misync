package comm

import (
	"encoding/json"
	"fmt"
	"github.com/clouderhem/micloud/utility/parallel"
	"os"
)

type ErrWrapper[T any] struct {
	Data T
	Err  string
}

func SaveErrOuts[T any](filePath string, errs []parallel.ErrOut[T]) error {
	if len(errs) == 0 {
		return nil
	}
	var r []ErrWrapper[T]
	for i := range errs {
		r = append(r, ErrWrapper[T]{
			Data: errs[i].In,
			Err:  fmt.Sprintf("%v", errs[i].Err),
		})
	}
	bytes, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, bytes, os.ModePerm)
}
