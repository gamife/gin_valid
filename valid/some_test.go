package valid

import (
	"github.com/gamife/gin_valid/gin/binding"
	"github.com/gamife/gin_valid/go-playground/validator/v10"
	"github.com/go-playground/assert/v2"
	"log"
	"strings"
	"testing"
	"unsafe"
)

func TestSplit(t *testing.T) {
	slice := []string{"1,2,3", "1,2", "1,", "1", ""}
	for i, j := range slice {
		n := strings.SplitN(j, ",", 2)
		assert.Equal(t, len(n), 2)
		t.Log(i)
	}
}

type ObjA struct {
	Name     int `json:"name" binding:"eqfield=NikeName" description:"真名"`
	NikeName int `json:"nikeName" binding:"gte=1" description:"昵称"`
}

func TestValidTwoField(t *testing.T) {
	obj := &ObjA{
		Name:     1,
		NikeName: 2,
	}
	Valid(obj)
}

func Valid(obj interface{}) {
	err := binding.Validator.ValidateStruct(obj)
	if err == nil {
		log.Fatal("无错误")
	}
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		// 非validator.ValidationErrors类型错误直接返回
		log.Fatal("错误类型错误,", err)
	}
	err0 := errs.Translate(binding.ValidTrans)
	if err0 != nil {
		log.Fatalln(err0)
	}
}

func iferr(s string, err error) {
	if err != nil {
		log.Println(s)
		log.Fatal(err)
	}
}

//var (
//	text    = "{0}必须与{1}一致"
//	indexes = []int{0, 3, 12, 15}
//	params  = []string{"first", "second"}
//)
//
//func BenchmarkNameNoBufNoPointer(b *testing.B) {
//	for i := 0; i < b.N; i++ {
//		noBuf()
//	}
//}
//
//func noBuf() string {
//	b := make([]byte, 0, 64)
//	var start, end, count int
//	for i := 0; i < len(indexes); i++ {
//		end = indexes[i]
//		b = append(b, text[start:end]...)
//		b = append(b, params[count]...)
//		i++
//		start = indexes[i]
//		count++
//	}
//	b = append(b, text[start:]...)
//	return string(b)
//}

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
