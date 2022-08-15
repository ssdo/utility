package utility_test

import (
	"github.com/ssdo/utility"
	"github.com/ssgo/log"
	"github.com/ssgo/u"
	"regexp"
	"testing"
)

type InType struct {
	Name      string `verify:"length:6+"`
	Phone     string `verify:"^1\\d{10}$"`
	Phone2     string `verify:"phone"`
	ImageCode string `verify:"length:4"`
	Age       int    `verify:"between:1-100"`
	Sex       string `verify:"in:男,女"`
}

var internationalPhoneMatcher *regexp.Regexp

func init() {
	internationalPhoneMatcher = regexp.MustCompile("^\\+?[0-9][0-9\\-]+$")
}

func TestVerify(t *testing.T) {
	utility.RegisterVerifyFunc("phone", func(in interface{}, args []string) bool {
		supportInternational := false
		if len(args) > 0 && args[0] == "international" {
			supportInternational = true
		}
		phone := u.String(in)
		if supportInternational {
			return internationalPhoneMatcher.MatchString(phone)
		} else if len(phone) == 11 && phone[0] == '1' {
			return true
		}
		return false
	})

	tests := [][]interface{}{
		{"abc", "length:6+", false},
		{"abcdef", "length:6+", true},
		{"abcdefg", "length:6+", true},
		{123, "length:4", false},
		{1234, "length:4", true},
		{"139", "^1\\d{10}$", false},
		{"1234567890X", "^1\\d{10}$", false},
		{"12345678900", "^1\\d{10}$", true},
		{2, "gt", true},
		{2, "gt:5", false},
		{2, "lt", false},
		{2, "lt:5", true},
		{2, "between:5", true},
		{2, "between:1-5", true},
		{2, "between:3-5", false},
		{"1", "in:1,2,3", true},
		{2, "in:1,2,3", true},
		{"xxx", "in:1,2,3", false},
		{"1391234567", "phone", false},
		{"13912345678", "phone", true},
		{"+8612345678", "phone", false},
		{"13912345678", "phone:international", true},
		{"+8612345678", "phone:international", true},
		{"+86-021-12345678", "phone:international", true},
	}

	for _, a := range tests {
		if ok, _ := utility.Verify(a[0], u.String(a[1]), log.DefaultLogger); ok != u.Bool(a[2]) {
			t.Fatal("failed", a)
		}
	}

	ok, field := utility.VerifyStruct(InType{
		Name:      "abc",
		Phone:     "139",
		Phone2:     "139",
		ImageCode: "123",
		Age:       300,
		Sex:       "不男不女",
	}, log.DefaultLogger)
	if ok || field != "Sex" {
		t.Fatal("failed not Sex", ok, field)
	}

	ok, field = utility.VerifyStruct(InType{
		Name:      "abcdefg",
		Phone:     "13912345678",
		Phone2:     "13912345678",
		ImageCode: "1234",
		Age:       55,
		Sex:       "男",
	}, log.DefaultLogger)
	if !ok || field != "" {
		t.Fatal("not ok", ok, field)
	}
}
