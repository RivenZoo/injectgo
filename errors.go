package injectgo

import "fmt"

var (
	errValueNotPtrOrInterface = fmt.Errorf("value should be pointer to struct or interface")
	errValueNotFunction       = fmt.Errorf("value should be function")
)
