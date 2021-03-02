package b

import (
	"github.com/gamife/gin_valid/gin/binding"
	"github.com/gamife/gin_valid/go-playground/validator/v10"
	"mime"
	"net/http"
)

type ValidError struct {
	ErrString string
}

func (e ValidError) Error() string {
	return e.ErrString
}

func ShouldBind(req *http.Request, obj interface{}) error {
	content, err := contentType(req)
	if err != nil {
		return err
	}
	b := binding.Default(req.Method, content)
	return ShouldBindWith(req, obj, b)

}

func ShouldBindWith(req *http.Request, obj interface{}, b binding.Binding) error {
	err := b.Bind(req, obj)
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		// 非validator.ValidationErrors类型错误直接返回
		return err
	}
	err0 := errs.Translate(binding.ValidTrans)
	if err0 != nil {
		return ValidError{ErrString: err0.Error()}
	}
	return nil
}
func ShouldBindJSON(req *http.Request, obj interface{}) error {
	return ShouldBindWith(req, obj, binding.JSON)
}
func ShouldBindHeader(req *http.Request, obj interface{}) error {
	return ShouldBindWith(req, obj, binding.Header)
}
func ShouldBindQuery(req *http.Request, obj interface{}) error {
	return ShouldBindWith(req, obj, binding.Query)
}

func contentType(r *http.Request) (string, error) {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	ct, _, err := mime.ParseMediaType(ct)
	return ct, err
}

func filterFlags(content string) string {
	for i, char := range content {
		if char == ' ' || char == ';' {
			return content[:i]
		}
	}
	return content
}
