package main

import (
	"fmt"
	"reflect"
	"google.golang.org/genai"
)

func main() {
	t := reflect.TypeOf(genai.ImageConfig{})
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fmt.Printf("Field: %s Type: %s\n", f.Name, f.Type)
	}
}

