// Copyright 2017 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"github.com/gamife/gin_valid/go-playground/locales/zh"
	ut "github.com/gamife/gin_valid/go-playground/universal-translator"
	"github.com/gamife/gin_valid/go-playground/validator/v10"
	zhTrans "github.com/gamife/gin_valid/go-playground/validator/v10/translations/zh"
	"reflect"
	"strings"
	"sync"
)

type defaultValidator struct {
	once     sync.Once
	validate *validator.Validate
}

var _ StructValidator = &defaultValidator{} //这是啥情况?yang

// ValidateStruct receives any kind of type, but only performed struct or pointer to struct type.
func (v *defaultValidator) ValidateStruct(obj interface{}) error {
	value := reflect.ValueOf(obj)
	valueType := value.Kind()
	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	if valueType == reflect.Struct {
		v.lazyinit()
		if err := v.validate.Struct(obj); err != nil {
			return err
		}
	}
	return nil
}

// Engine returns the underlying validator engine which powers the default
// Validator instance. This is useful if you want to register custom validations
// or struct level validations. See validator GoDoc for more info -
// https://godoc.org/gopkg.in/go-playground/validator.v8
func (v *defaultValidator) Engine() interface{} {
	v.lazyinit()
	return v.validate
}

var ValidTrans ut.Translator

func (v *defaultValidator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New()
		zh := zh.New()
		uni := ut.New(zh, zh)

		// this is usually know or extracted from http 'Accept-Language' header
		// also see uni.FindTranslator(...)
		ValidTrans, _ = uni.GetTranslator("zh")

		zhTrans.RegisterDefaultTranslations(v.validate, ValidTrans) // 为gin的校验 注册翻译
		v.validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("description"), ",", 2)[0]
			//if name == "-" {
			//	return ""
			//}
			return name
		})
		// 设置 tag 的名字
		v.validate.SetTagName("binding")
	})
}
