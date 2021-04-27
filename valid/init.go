package valid

import (
	"fmt"
	"github.com/gamife/gin_valid/gin/binding"
	ut "github.com/gamife/gin_valid/go-playground/universal-translator"
	"github.com/gamife/gin_valid/go-playground/validator/v10"
	"log"
	"os"
	"reflect"
	"strings"
	"time"
)

// 验证的标志 baked_in.go
// go-playground/validator/v10/validator_instance.gok 各种分隔符

var (
	V *validator.Validate
)

func Init() {
	var err error
	defer func() {
		if err != nil {
			log.Print(err)
			os.Exit(1)
			return
		}
		log.Printf("valid库初始化完成")
	}()
	V = binding.Validator.Engine().(*validator.Validate)

	if err = V.RegisterValidation("setDate", setTimeFunc); err != nil {
		return
	}

	if err = V.RegisterValidation("default", defaultFunc); err != nil {
		return
	}

	if err = V.RegisterTranslation("setDate", binding.ValidTrans,
		registrationFunc("setDate", "{0}的格式错误", false),
		TranslateFunc,
	); err != nil {
		return
	}

}

// fl.GetStructFieldOKAdvanced2() 好像是提前验证其他字段 , 可以看 baked_in.go的requireCheckFieldKind()
//FieldName: 活动名称
//StructFieldName: Name
//Param: gaga g
//GetTag: checkDate

var (
	TimeFormatMap = map[string]string{
		"y":   "2006",
		"m":   "2006-01",
		"d":   "2006-01-02",
		"h":   "2006-01-02 15",
		"min": "2006-01-02 15:04",
		"s":   "2006-01-02 15:04:05",
	}
)

// 这个是将前端传的 string 放入一个 time.Time 类型的字段中, 默认会检查 StructName1 字段

/*
参数1:时间的格式, 默认按照2006-01-02 15:04:05, 可使用 TimeFormatMap 中的简写
参数2:反射到哪个字段,默认尝试当前字段名拼接1,形如 Name1
比如
type struct{
	StartTime string `binding:"setTime=s&Start"`
	StartTime1 time.Time
	Start time.Time
}
	这样会将传进来的 StartTime 按照字典中的s格式解析到 Start 字段
*/
func setTimeFunc(fl validator.FieldLevel) (bool, []string) {
	//fmt.Println("FieldName:", fl.FieldName())
	//fmt.Println("StructFieldName:", fl.StructFieldName())
	//fmt.Println("Param:", fl.Param())
	//fmt.Println("GetTag:", fl.GetTag())
	p := fl.Param()
	var timeFormat = "2006-01-02 15:04:05"
	var s []string
	s = strings.SplitN(p, "&", 2) // 空的也会有 s[0]
	if s[0] != "" {
		var ok bool
		if timeFormat, ok = TimeFormatMap[s[0]]; !ok {
			timeFormat = s[0]
		}
	}

	date, err := time.ParseInLocation(timeFormat, fl.Field().String(), time.Local)
	if err != nil {
		return false, nil
	}
	var alwaysTrue bool
	var fieldName string
	if len(s) > 1 || s[1] == "" {
		// 如果不设定 &NameXX , 就默认看一下 Name1
		alwaysTrue = true
		fieldName = fl.StructFieldName() + "1"
	} else {
		fieldName = strings.TrimSpace(s[1])
	}

	parentV := reflect.Indirect(fl.Parent())
	parentT := parentV.Type()

	// set到对应的字段
	if timeField, ok := parentT.FieldByName(fieldName); ok {
		parentV.Field(timeField.Index[0]).Set(reflect.ValueOf(date))
		return true, nil
	}
	if alwaysTrue {
		return true, nil
	}
	fmt.Println("setTime错误,没找到对应的字段")
	return false, nil
}

// 设定默认值,只支持string 和数字 ,可以自己扩展,这几个转换函数都是从 官方库拿的
// `binding:"default=111"`
func defaultFunc(fl validator.FieldLevel) (bool, []string) {
	field := fl.Field()
	param := fl.Param()
	if !field.IsZero() {
		return true, nil
	}

	switch field.Kind() {

	case reflect.String:
		field.SetString(param)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := AsIntFromType(field.Type(), param)
		field.SetInt(p)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := AsUint(param)
		field.SetUint(p)

	case reflect.Float32, reflect.Float64:
		p := AsFloat(param)
		field.SetFloat(p)
	default:
		panic(fmt.Sprintf("Bad field type %T", field.Interface()))
	}
	return true, nil
}

func registrationFunc(tag string, translation string, override bool) validator.RegisterTranslationsFunc {
	return func(ut ut.Translator) (err error) {
		if err = ut.Add(tag, translation, override); err != nil {
			return
		}
		return

	}
}

// TranslateFunc 自定义字段的翻译方法
func TranslateFunc(ut ut.Translator, fe validator.FieldError) string {
	t, err := ut.T(fe.Tag(), fe.Field())
	if err != nil {
		log.Printf("警告: 翻译字段错误: %#v", fe)
		return fe.(error).Error()
	}
	return t
}
