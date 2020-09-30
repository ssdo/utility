package utility

import (
	"github.com/ssgo/log"
	"github.com/ssgo/u"
	"reflect"
	"regexp"
	"strings"
)

type VerifyType uint8

const (
	Unknown VerifyType = iota
	Regex
	StringLength
	GreaterThan
	LessThan
	Between
	InList
	ByFunc
)

type VerifySet struct {
	Type       VerifyType
	Regex      *regexp.Regexp
	StringArgs []string
	IntArgs    []int
	FloatArgs  []float64
	Func       func(interface{}, []string) bool
}

var verifySets = map[string]*VerifySet{}
var verifyFunctions = map[string]func(interface{}, []string) bool{}

func RegisterVerify(name string, f func(in interface{}, args []string) bool) {
	verifyFunctions[name] = f
}

// 验证一个结构
func VerifyStruct(in interface{}) (ok bool, field string) {
	// 查找最终对象
	v := u.FinalValue(reflect.ValueOf(in))
	if v.Kind() != reflect.Struct {
		return false, ""
	}

	// 处理每个字段
	for i := v.NumField() - 1; i >= 0; i-- {
		ft := v.Type().Field(i)
		fv := v.Field(i)
		if ft.Anonymous {
			// 处理继承
			ok, field = VerifyStruct(fv.Interface())
			if !ok {
				return ok, field
			}
		} else {
			// 处理字段
			tag := ft.Tag.Get("verify")
			if len(tag) >= 2 {
				// 有效的验证信息
				ok = Verify(fv.Interface(), tag)
				if !ok {
					return ok, ft.Name
				}
			}
		}
	}
	return true, ""
}

// 验证一个数据
func Verify(in interface{}, setting string) bool {
	if len(setting) < 2 {
		return false
	}

	set := verifySets[setting]
	if set == nil {
		set = compileVerifySet(setting)
		verifySets[setting] = set
	}

	switch set.Type {
	case ByFunc:
		return set.Func(in, set.StringArgs)
	case Regex:
		return set.Regex.MatchString(u.String(in))
	case StringLength:
		if set.StringArgs != nil && set.StringArgs[0] == "+" {
			return len(u.String(in)) >= set.IntArgs[0]
		} else if set.StringArgs != nil && set.StringArgs[0] == "-" {
			return len(u.String(in)) <= set.IntArgs[0]
		} else {
			return len(u.String(in)) == set.IntArgs[0]
		}
	case GreaterThan:
		return u.Float64(in) > set.FloatArgs[0]
	case LessThan:
		return u.Float64(in) < set.FloatArgs[0]
	case Between:
		return u.Float64(in) >= set.FloatArgs[0] && u.Float64(in) <= set.FloatArgs[1]
	case InList:
		found := false
		inStr := u.String(in)
		for _, item := range set.StringArgs {
			if item == inStr {
				found = true
				break
			}
		}
		return found
	}
	return false
}

// 编译验证设置
func compileVerifySet(setting string) *VerifySet {
	set := new(VerifySet)
	set.Type = Unknown

	made := false
	if setting[0] != '^' {
		key := setting
		args := ""
		if pos := strings.IndexByte(setting, ':'); pos != -1 {
			key = setting[0:pos]
			args = setting[pos+1:]
		}
		// 查找是否有注册Func
		if !made && verifyFunctions[key] != nil {
			made = true
			set.Type = ByFunc
			if args == "" {
				set.StringArgs = make([]string, 0)
			} else {
				set.StringArgs = strings.Split(args, ",")
			}
			set.Func = verifyFunctions[key]
		}

		// 处理默认支持的类型
		if !made {
			made = true
			switch key {
			case "length":
				// 判断字符串长度
				set.Type = StringLength
				if args == "" {
					args = "1+"
				}
				lastChar := args[len(args)-1]
				if lastChar == '+' || lastChar == '-' {
					set.StringArgs = []string{string(lastChar)}
					args = args[0 : len(args)-1]
				}
				set.IntArgs = []int{u.Int(args)}
			case "between":
				// 判断数字范围
				set.Type = Between
				if args == "" {
					args = "1-100000000"
				}
				a2 := strings.Split(args, "-")
				if len(a2) == 1 {
					// 如果只设置一个参数，范围为1-指定数字
					tempStr := a2[0]
					a2[0] = "0"
					a2 = append(a2, tempStr)
				}
				set.FloatArgs = []float64{u.Float64(a2[0]), u.Float64(a2[1])}
			case "gt":
				// 大于
				set.Type = GreaterThan
				if args == "" {
					args = "0"
				}
				set.FloatArgs = []float64{u.Float64(args)}
			case "lt":
				// 小于
				set.Type = LessThan
				if args == "" {
					args = "0"
				}
				set.FloatArgs = []float64{u.Float64(args)}
			case "in":
				// 枚举
				set.Type = InList
				if args == "" {
					set.StringArgs = make([]string, 0)
				} else {
					set.StringArgs = strings.Split(args, ",")
				}
			default:
				made = false
			}
		}
	}

	if !made {
		rx, err := regexp.Compile(setting)
		if err != nil {
			log.DefaultLogger.Error(err.Error())
		} else {
			set.Type = Regex
			set.Regex = rx
		}
	}

	return set
}
