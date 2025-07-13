package hello

import (
	"fmt"
)

func SayHello(x string) string {
	return fmt.Sprintf("Hello %s!", x)
}
