package validator

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

type tagType uint8

const (
	typeDefault tagType = iota
	typeOmitEmpty
	typeIsDefault
	typeNoStructLevel
	typeStructOnly
	typeDive
	typeOr
	typeKeys
	typeEndKeys
)

const (
	invalidValidation   = "Invalid validation tag on field '%s'"
	undefinedValidation = "Undefined validation function '%s' on field '%s'"
	keysTagNotDefined   = "'" + endKeysTag + "' tag encountered without a corresponding '" + keysTag + "' tag"
)

type structCache struct {
	lock sync.Mutex
	m    atomic.Value // map[reflect.Type]*cStruct
}

func (sc *structCache) Get(key reflect.Type) (c *cStruct, found bool) {
	c, found = sc.m.Load().(map[reflect.Type]*cStruct)[key] //有一个断言的封装
	return
}

func (sc *structCache) Set(key reflect.Type, value *cStruct) {
	m := sc.m.Load().(map[reflect.Type]*cStruct)
	nm := make(map[reflect.Type]*cStruct, len(m)+1)
	for k, v := range m {
		nm[k] = v
	}
	nm[key] = value
	sc.m.Store(nm)
}

type tagCache struct {
	lock sync.Mutex
	m    atomic.Value // map[string]*cTag
}

func (tc *tagCache) Get(key string) (c *cTag, found bool) {
	c, found = tc.m.Load().(map[string]*cTag)[key]
	return
}

func (tc *tagCache) Set(key string, value *cTag) {
	m := tc.m.Load().(map[string]*cTag)
	nm := make(map[string]*cTag, len(m)+1)
	for k, v := range m {
		nm[k] = v
	}
	nm[key] = value
	tc.m.Store(nm)
}

type cStruct struct {
	name   string
	fields []*cField
	fn     StructLevelFuncCtx
}

type cField struct {
	idx        int
	name       string // field.name
	altName    string // 如果没有自定义 tagNameFunc,就是field.name, 如果有,比如可以按 json_tag中的, 报错的时候就会按照这个 报错
	namesEqual bool   // field.name 和 altName是否相同
	cTags      *cTag  //以链表的形式存在
}

type cTag struct {
	tag                  string // (gte=10) 实际是 gte,不带参数
	aliasTag             string // 应该是别名, 不过在没有别名的时候等同于 tag
	actualAliasTag       string // 实际名字,而不是在func *Validate.parseFieldTagsRecursive 解析后的 别名
	param                string
	keys                 *cTag // only populated when using tag's 'keys' and 'endkeys' for map key validation
	next                 *cTag
	fn                   FuncCtx
	typeof               tagType
	hasTag               bool
	hasAlias             bool
	hasParam             bool // true if parameter used eg. eq= where the equal sign has been set
	isBlockEnd           bool // indicates the current tag represents the last validation in the block , 表示当前field的最后一个tag,比如 `require,gte=10`,那么到gte就是true
	runValidationWhenNil bool
}

func (v *Validate) extractStructCache(current reflect.Value, sName string) *cStruct {
	v.structCache.lock.Lock()
	defer v.structCache.lock.Unlock() // leave as defer! because if inner panics, it will never get unlocked otherwise!

	typ := current.Type()

	// could have been multiple trying to access, but once first is done this ensures struct
	// isn't parsed again.
	cs, ok := v.structCache.Get(typ)
	if ok {
		return cs
	}

	cs = &cStruct{name: sName, fields: make([]*cField, 0), fn: v.structLevelFuncs[typ]}

	numFields := current.NumField()

	var ctag *cTag
	var fld reflect.StructField
	var tag string
	var customName string

	for i := 0; i < numFields; i++ {

		fld = typ.Field(i)

		if !fld.Anonymous && len(fld.PkgPath) > 0 { // 如果不是 嵌套field 且是 未导出字段, (大写的字段 pakPath为空)
			continue
		}

		tag = fld.Tag.Get(v.tagName)

		if tag == skipValidationTag {
			continue
		}

		customName = fld.Name

		if v.hasTagNameFunc { // 自定义的获取 名字的func
			name := v.tagNameFunc(fld)
			if len(name) > 0 {
				customName = name
			}
		}

		// NOTE: cannot use shared tag cache, because tags may be equal, but things like alias may be different
		// and so only struct level caching can be used instead of combined with Field tag caching

		if len(tag) > 0 {
			ctag, _ = v.parseFieldTagsRecursive(tag, fld.Name, "", false) //返回链头
		} else {
			// even if field doesn't have validations need cTag for traversing to potential inner/nested
			// elements of the field.
			ctag = new(cTag)
		}

		cs.fields = append(cs.fields, &cField{
			idx:        i,
			name:       fld.Name,
			altName:    customName,
			cTags:      ctag,
			namesEqual: fld.Name == customName,
		})
	}
	v.structCache.Set(typ, cs)
	return cs
}

func (v *Validate) parseFieldTagsRecursive(tag string, fieldName string, alias string, hasAlias bool) (firstCtag *cTag, current *cTag) {
	var t string               // 最开始如果有别名,比如 r 代表 required,那么t为 `r`,如果有多个条件  gte=1,gt=10   那么 t 依次等于 gte=1  gt=10
	noAlias := len(alias) == 0 // 别名这个东西,只是用来 判断当前是否是别名转过来的
	tags := strings.Split(tag, tagSeparator)

	for i := 0; i < len(tags); i++ {
		t = tags[i]
		if noAlias {
			alias = t
		}

		// check map for alias and process new tags, otherwise process as usual
		if tagsVal, found := v.aliases[t]; found {
			if i == 0 {
				firstCtag, current = v.parseFieldTagsRecursive(tagsVal, fieldName, t, true)
			} else {
				next, curr := v.parseFieldTagsRecursive(tagsVal, fieldName, t, true)
				current.next, current = next, curr //链表,可能返回的是一串,比如  123  接  456, current=3  接 next=4 ,current.next=curr=6
			}
			continue
		}

		var prevTag tagType

		if i == 0 {
			current = &cTag{aliasTag: alias, hasAlias: hasAlias, hasTag: true, typeof: typeDefault}
			firstCtag = current
		} else {
			prevTag = current.typeof // 此刻之前current 还是 上一个的状态
			current.next = &cTag{aliasTag: alias, hasAlias: hasAlias, hasTag: true}
			current = current.next
		}

		switch t {
		case diveTag:
			current.typeof = typeDive
			continue

		case keysTag:
			current.typeof = typeKeys

			if i == 0 || prevTag != typeDive {
				panic(fmt.Sprintf("'%s' tag must be immediately preceded by the '%s' tag", keysTag, diveTag))
			}

			current.typeof = typeKeys

			// need to pass along only keys tag
			// need to increment i to skip over the keys tags
			b := make([]byte, 0, 64)

			i++

			for ; i < len(tags); i++ {

				b = append(b, tags[i]...)
				b = append(b, ',')

				if tags[i] == endKeysTag {
					break
				}
			}

			current.keys, _ = v.parseFieldTagsRecursive(string(b[:len(b)-1]), fieldName, "", false)
			continue

		case endKeysTag:
			current.typeof = typeEndKeys

			// if there are more in tags then there was no keysTag defined
			// and an error should be thrown
			if i != len(tags)-1 {
				panic(keysTagNotDefined)
			}
			return

		case omitempty:
			current.typeof = typeOmitEmpty
			continue

		case structOnlyTag:
			current.typeof = typeStructOnly
			continue

		case noStructLevelTag:
			current.typeof = typeNoStructLevel
			continue

		default:
			if t == isdefault {
				current.typeof = typeIsDefault
			}
			// if a pipe character is needed within the param you must use the utf8Pipe representation "0x7C"
			orVals := strings.Split(t, orSeparator)

			for j := 0; j < len(orVals); j++ {
				vals := strings.SplitN(orVals[j], tagKeySeparator, 2) // binding:"gte=10" 的 =
				if noAlias {
					alias = vals[0] //如果没有别名就 gte , 前面的是 gte=10
					current.aliasTag = alias
				} else {
					current.actualAliasTag = t // 原始名字 gte=10
				}

				if j > 0 { //这里的current.next 不应该是 .pre 前一个么,不过前一个也不对啊,后面的赋值还是给 current赋值啊
					current.next = &cTag{aliasTag: alias, actualAliasTag: current.actualAliasTag, hasAlias: hasAlias, hasTag: true}
					current = current.next
				}
				current.hasParam = len(vals) > 1

				current.tag = vals[0]
				if len(current.tag) == 0 {
					panic(strings.TrimSpace(fmt.Sprintf(invalidValidation, fieldName)))
				}

				if wrapper, ok := v.validations[current.tag]; ok {
					current.fn = wrapper.fn
					current.runValidationWhenNil = wrapper.runValidatinOnNil
				} else {
					panic(strings.TrimSpace(fmt.Sprintf(undefinedValidation, current.tag, fieldName)))
				}

				if len(orVals) > 1 {
					current.typeof = typeOr
				}

				if len(vals) > 1 {
					// 转义
					current.param = strings.Replace(strings.Replace(vals[1], utf8HexComma, ",", -1), utf8Pipe, "|", -1)
				}
			}
			current.isBlockEnd = true
		}
	}
	return
}

func (v *Validate) fetchCacheTag(tag string) *cTag {
	// find cached tag
	ctag, found := v.tagCache.Get(tag)
	if !found {
		v.tagCache.lock.Lock()
		defer v.tagCache.lock.Unlock()

		// could have been multiple trying to access, but once first is done this ensures tag
		// isn't parsed again.
		ctag, found = v.tagCache.Get(tag)
		if !found {
			ctag, _ = v.parseFieldTagsRecursive(tag, "", "", false)
			v.tagCache.Set(tag, ctag)
		}
	}
	return ctag
}
