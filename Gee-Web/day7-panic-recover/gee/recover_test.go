package gee

import (
	"fmt"
	"testing"
)

func test_recover() {
	defer func() {
		fmt.Println("defer func")
		if err := recover(); err != nil {
			fmt.Println("recover success")
		}
	}()
	arr := []int{1, 2, 3}
	fmt.Println(arr[4])
	fmt.Println("after panic")
}

func TestRecover(t *testing.T) {
	test_recover()
	fmt.Println("After recover")
}
