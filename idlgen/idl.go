package idlgen

import (
	"errors"
	"fmt"
	"go/ast"
	"strings"

	"github.com/xiehengjian/struct2thrift/program"
	"github.com/xiehengjian/struct2thrift/util"

	log "github.com/liudanking/goutil/logutil"
)

var alreadyGen = map[string]bool{}

func Generate(f *ast.File, typ *ast.TypeSpec) (idls []string, err error) {
	ms, err := NewIDLGenerator(typ)
	if err != nil {
		log.Warning("create model struct failed:%v", err)
		return idls, err
	}

	idl, structs, err := ms.GetCreateIDL()
	if err != nil {
		log.Warning("generate sql failed:%v", err)
		return idls, err
	}
	for _, structName := range structs {
		if _, ok := alreadyGen[structName]; ok {
			continue
		}
		alreadyGen[structName] = true
		typeSpec, _ := program.GetStructByName(f, structName)
		subIDLs, _ := Generate(f, typeSpec)
		idls = append(idls, subIDLs...)

	}
	idls = append(idls, idl)
	return idls, nil
}

type IDLGenerator struct {
	structName string
	modelType  *ast.StructType
}

func NewIDLGenerator(typeSpec *ast.TypeSpec) (*IDLGenerator, error) {
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil, errors.New("typeSpec is not struct type")
	}

	return &IDLGenerator{
		structName: typeSpec.Name.Name,
		modelType:  structType,
	}, nil
}

func (ms *IDLGenerator) GetCreateIDL() (idl string, structs []string, err error) {
	var fieldLines []string
	fieldNums := ms.getStructFieds(ms.modelType)
	for idx, field := range fieldNums { //遍历这个结构体的每一个字段
		switch t := field.Type.(type) { // 根据字段类型区分
		case *ast.Ident:
			// if t.Obj != nil {
			// 	typeSpec, err := program.GetStructByName(f, getColumnName(field))
			// 	if err != nil {
			// 		idl, _ := Generate(f, typeSpec)
			// 		idls = append(idls, idl...)
			// 	}
			// }
			fieldLine, isStruct, err := generateFieldLine(idx, field)
			if err != nil {
				log.Warning("generate field line [%v] failed:%v", t, err)
			} else {
				fieldLines = append(fieldLines, fieldLine)
			}
			if isStruct {
				structs = append(structs, t.Name)
			}

		case *ast.ArrayType:
			fieldLine, isStruct, err := generateFieldLine(idx, field)
			if err != nil {
				log.Warning("generate field line [%v] failed:%v", t, err)
			} else {
				fieldLines = append(fieldLines, fieldLine)
			}
			if isStruct {
				structs = append(structs, t.Elt.(*ast.Ident).Name)
			}
		case *ast.MapType: // 标识符
			fieldLine, isStruct, err := generateFieldLine(idx, field)
			if err != nil {
				log.Warning("generate field line [%v] failed:%v", t, err)
			} else {
				fieldLines = append(fieldLines, fieldLine)
			}
			if isStruct {
				if value, ok := t.Value.(*ast.Ident); ok {
					structs = append(structs, value.Name)
				}
				if value, ok := t.Value.(*ast.StarExpr); ok {
					if value, ok := value.X.(*ast.Ident); ok {
						structs = append(structs, value.Name)
					}
				}

			}
		case *ast.SelectorExpr:
			fieldLine, _, err := generateFieldLine(idx, field)
			if err != nil {
				log.Warning("generate field line [%s] failed:%v", t.Sel.Name, err)
			} else {
				fieldLines = append(fieldLines, fieldLine)
			}
		default:
			fieldLine, _, err := generateFieldLine(idx, field)
			if err != nil {
				log.Warning("generate field line failed:%v", err)
			} else {
				fieldLines = append(fieldLines, fieldLine)
			}
			log.Warning("field %s not supported, ignore", util.GetFieldName(field))
		}
	}

	idl = fmt.Sprintf(`struct %v{
  %v
}`, ms.tableName(), strings.Join(append(fieldLines), ",\n  "))

	return idl, structs, nil

}

func (ms *IDLGenerator) getStructFieds(node ast.Node) []*ast.Field {
	var fields []*ast.Field
	nodeType, ok := node.(*ast.StructType)
	if !ok {
		return nil
	}
	for _, field := range nodeType.Fields.List {
		switch field.Type.(type) {
		case *ast.Ident:
			fields = append(fields, field)
		case *ast.SelectorExpr:
			fields = append(fields, field)
		default:
			fields = append(fields, field)
			log.Warning("filed %s not supported, ignore", util.GetFieldName(field))
		}
	}

	return fields
}

func (ms *IDLGenerator) tableName() string {
	return ms.structName
}

func generateFieldLine(idx int, field *ast.Field) (string, bool, error) {
	fieldType, isStruct, err := getFieldType(field) // 获取字段类型
	if err != nil {
		log.Warning("get mysql field tag failed:%v", err)
		return "", isStruct, err
	}

	fieldName := getColumnName(field)
	httpFiedName := util.GetFieldTag(field, "json").Name //获取tag
	if httpFiedName == "" {
		httpFiedName = fieldName
	}
	return fmt.Sprintf(` %d: %s %s (api.body = "%s")`, idx, fieldType, fieldName, httpFiedName), isStruct, nil
}

func getColumnName(field *ast.Field) string {
	if len(field.Names) > 0 {
		return fmt.Sprintf("%s", field.Names[0].Name)
	}
	return ""
}

func getFieldType(field *ast.Field) (string, bool, error) {
	switch t := field.Type.(type) {
	case *ast.Ident:
		return basicType(t)
	case *ast.SelectorExpr:
		// typeName = t.Sel.Name
	case *ast.ArrayType:
		return listType(t.Elt.(*ast.Ident))
	case *ast.MapType:
		if value, ok := t.Value.(*ast.Ident); ok {
			return mapType(t.Key.(*ast.Ident), value)
		}
		if value, ok := t.Value.(*ast.StarExpr); ok {
			if value, ok := value.X.(*ast.Ident); ok {
				return mapType(t.Key.(*ast.Ident), value)
			}
		}
	default:
		return "", false, errors.New(fmt.Sprintf("field %s not supported", util.GetFieldName(field)))
	}
	return "", false, errors.New(fmt.Sprintf("field %s not supported", util.GetFieldName(field)))
}

func basicType(t *ast.Ident) (string, bool, error) {
	if t.Obj != nil {
		return t.Name, true, nil
	}
	switch t.Name {
	case "bool":
		return "bool", false, nil
	case "int":
		return "i32", false, nil
	case "int64":
		return "i64", false, nil
	case "string":
		return "string", false, nil
	case "float64":
		return "double", false, nil
	}
	return "", false, errors.New("unsupport type")
}

func listType(t *ast.Ident) (string, bool, error) {
	basicType, isStruct, err := basicType(t)
	if err != nil {
		return "", isStruct, err
	}
	return fmt.Sprintf("list<%s>", basicType), isStruct, nil
}

func mapType(keyIdent, valueIdent *ast.Ident) (string, bool, error) {
	keyType, _, err := basicType(keyIdent)
	if err != nil {
		return "", false, err
	}
	valueType, isStruct, err := basicType(valueIdent)
	if err != nil {
		return "", isStruct, err
	}
	return fmt.Sprintf("map<%s,%s>", keyType, valueType), isStruct, nil
}
