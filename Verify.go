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
func VerifyStruct(in interface{}, logger *log.Logger) (ok bool, field string) {
	// 查找最终对象
	v := u.FinalValue(reflect.ValueOf(in))
	if v.Kind() != reflect.Struct {
		logger.Error("verify input is not struct", "in", in)
		return false, ""
	}

	// 处理每个字段
	for i := v.NumField() - 1; i >= 0; i-- {
		ft := v.Type().Field(i)
		fv := v.Field(i)
		if fv.Kind() == reflect.Ptr && fv.IsNil() {
			// 不校验为nil的指针类型
			continue
		}
		if ft.Anonymous {
			// 处理继承
			ok, field = VerifyStruct(fv.Interface(), logger)
			if !ok {
				logger.Warning("verify failed", "in", in, "field", field)
				return false, field
			}
		} else {
			// 处理字段
			tag := ft.Tag.Get("verify")
			if len(tag) >= 2 {
				// 有效的验证信息
				ok, err := Verify(fv.Interface(), tag)
				if !ok {
					if err != nil {
						logger.Error(err.Error(), "in", in, "field", ft.Name)
					} else {
						logger.Warning("verify failed", "in", in, "field", ft.Name)
					}
					return false, ft.Name
				}
			}
		}
	}
	return true, ""
}

// 验证一个数据
func Verify(in interface{}, setting string) (bool, error) {
	if len(setting) < 2 {
		return false, nil
	}

	set := verifySets[setting]
	if set == nil {
		set2, err := compileVerifySet(setting)
		if err != nil {
			return false, err
		}
		set = set2
		verifySets[setting] = set
	}

	switch set.Type {
	case ByFunc:
		return set.Func(in, set.StringArgs), nil
	case Regex:
		return set.Regex.MatchString(u.String(in)), nil
	case StringLength:
		if set.StringArgs != nil && set.StringArgs[0] == "+" {
			return len(u.String(in)) >= set.IntArgs[0], nil
		} else if set.StringArgs != nil && set.StringArgs[0] == "-" {
			return len(u.String(in)) <= set.IntArgs[0], nil
		} else {
			return len(u.String(in)) == set.IntArgs[0], nil
		}
	case GreaterThan:
		return u.Float64(in) > set.FloatArgs[0], nil
	case LessThan:
		return u.Float64(in) < set.FloatArgs[0], nil
	case Between:
		return u.Float64(in) >= set.FloatArgs[0] && u.Float64(in) <= set.FloatArgs[1], nil
	case InList:
		found := false
		inStr := u.String(in)
		for _, item := range set.StringArgs {
			if item == inStr {
				found = true
				break
			}
		}
		return found, nil
	}
	return false, nil
}

// 编译验证设置
func compileVerifySet(setting string) (*VerifySet, error) {
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
		if verifyFunctions[key] != nil {
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
			//log.DefaultLogger.Error(err.Error())
			return nil, err
		} else {
			set.Type = Regex
			set.Regex = rx
		}
	}

	return set, nil
}
