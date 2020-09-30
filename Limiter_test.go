package utility_test

import (
	utility "github.com/ssege/user"
	"testing"
	"time"
)

func TestLimit2ms(t *testing.T){
	// 200ms内同一key限制10次
	l := utility.NewLimiter("test", time.Millisecond*20, 10, nil)
	// 前10次应该允许
	for i:=0; i<10; i++ {
		if !l.Check("aaa", nil){
			t.Fatal("第", i ,"次被拒绝")
		}
		//time.Sleep(time.Millisecond*1)

	}
	// 第11次应该拒绝
	if l.Check("aaa", nil){
		t.Fatal("第11次被接受")
	}
	// sleep 200ms 后，应该允许
	time.Sleep(time.Millisecond*200)
	for i:=0; i<10; i++ {
		if !l.Check("aaa", nil) {
			t.Fatal("第二轮，第", i ,"次被拒绝")
		}
	}
}