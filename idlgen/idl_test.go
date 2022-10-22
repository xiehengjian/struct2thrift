package idlgen

import (
	"go/parser"
	"go/token"
	"io/ioutil"

	"github.com/xiehengjian/struct2thrift/program"

	"testing"
)

func TestGenerateCreateTableSql(t *testing.T) {
	fset := token.NewFileSet()
	data, err := ioutil.ReadFile("../testdata/sqlmodel/response.go") // 读取user.go
	if err != nil {
		t.Fatal(err)
	}
	// 解析go源代码，返回对应的ast.File节点
	f, err := parser.ParseFile(fset, "model.go", string(data), parser.ParseComments) //解析文件
	if err != nil {
		t.Fatal(err)
	}
	// 获取某个结构体
	typeSpec, err := program.GetStructByName(f, "ReportSheet")
	if err != nil {
		t.Fatal(err)
	}
	idl, err := Generate(f, typeSpec)

	t.Logf("IDL:\n%s", idl)
}
