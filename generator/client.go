package generator

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	plugin_go "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
)

const apiTemplate = `
{{- range .Imports}}
import '{{.Path}}';
{{- end}}

{{range .Enums}}
enum {{.Name}} {
	{{- range .Values -}}
		{{.Name}},
	{{- end}}
}


String to{{.Name}}JsonValue({{.Name}} e) {
	switch(e) {
	{{- range .Values -}}
		case {{.EnumName}}.{{.Name}}: return "{{.Name}}";
	{{- end}}
		default: throw Exception("Unknown enum value: $e");
	}
}

{{.Name}} from{{.Name}}JsonValue(String j) {
	{{- range .Values -}}
	if (j == "{{.Name}}") return {{.EnumName}}.{{.Name}};
	{{- end}}
	throw Exception("Unknown json value: $j");
}
{{end}}

{{- range .Models}}
{{- if not .Primitive}}

{{.Name}} parse{{.Name}}(String j) {
	final value = json.decode(j) as Map<String, dynamic>;
    return {{.Name}}.fromJson(value);
}

class {{.Name}} {

	{{.Name}}(
	{{range .Fields -}}
		this.{{.Name}},
	{{- end}});

    {{range .Fields -}}
    {{.Type}} {{.Name}};
    {{end}}
	
	factory {{.Name}}.fromJson(Map<String,dynamic> json) {
		{{- range .Fields -}}
			{{if .IsMap}}
			final {{.Name}}Map = {{.Type}}();
			(json['{{.JSONNameFrom}}'] as Map<String, dynamic>)?.forEach((key, val) {
				if (val == null) {
					{{.Name}}Map[key] = null;
					return;
				}
				{{if .MapValueField.IsEnum}}
				{{.Name}}Map[key] = from{{.MapValueField.Type}}JsonValue(val);
				{{else if .MapValueField.IsMessage}}
				{{.Name}}Map[key] = {{.MapValueField.Type}}.fromJson(val as Map<String,dynamic>);
				{{else}}
				if (val is String) {
					{{if eq .MapValueField.Type "double"}}
						{{.Name}}Map[key] = double.parse(val);
					{{end}}
					{{if eq .MapValueField.Type "int"}}
						{{.Name}}Map[key] = int.parse(val);
					{{end}}
				} else if (val is num) {
					{{if eq .MapValueField.Type "double"}}
						{{.Name}}Map[key] = val.toDouble();
					{{end}}
					{{if eq .MapValueField.Type "int"}}
						{{.Name}}Map[key] = val.toInt();
					{{end}}
				}
				{{end}}
			});
			{{end}}
		{{end}}

		return {{.Name}}(
		{{- range .Fields -}}
		{{if .IsMap}}
		{{.Name}}Map,
		{{else if and .IsRepeated .IsMessage}}
		json['{{.JSONNameFrom}}'] != null
          ? ((json['{{.JSONNameFrom}}'] as List).cast<Map<String, dynamic>>())
              .map((d) => {{.InternalType}}.fromJson(d))
              .toList()
		  : <{{.InternalType}}>[],
		{{else if and .IsRepeated .IsEnum}}
		  json['{{.JSONNameFrom}}'] != null
			? (json['{{.JSONNameFrom}}'] as List)
				.map((d) => from{{.InternalType}}JsonValue(d))
				.toList()
			: <{{.InternalType}}>[],
		{{else if .IsRepeated }}
		json['{{.JSONNameFrom}}'] != null ? (json['{{.JSONNameFrom}}'] as List).cast<{{.InternalType}}>() : <{{.InternalType}}>[],
		{{else if and (.IsMessage) (eq .Type "DateTime")}}
		json['{{.JSONNameFrom}}'] == null ? null : {{.Type}}.tryParse(json['{{.JSONNameFrom}}']),
		{{else if .IsMessage}}
		json['{{.JSONNameFrom}}'] == null ? null : {{.Type}}.fromJson(json['{{.JSONNameFrom}}'] as Map<String, dynamic>),
		{{else if .IsEnum}}
		json['{{.JSONNameFrom}}'] == null ? null : from{{.Type}}JsonValue(json['{{.JSONNameFrom}}'] as String),
		{{- else if eq .Type "double"}}
		json['{{.JSONNameFrom}}'] == null ? null : (json['{{.JSONNameFrom}}'] as num).toDouble(), 
		{{- else if eq .Type "int"}}
		json['{{.JSONNameFrom}}'] == null ? null : (json['{{.JSONNameFrom}}'] as num).toInt(), 
		{{else}}
		json['{{.JSONNameFrom}}'] == null ? null : json['{{.JSONNameFrom}}'] as {{.Type}}, 
		{{- end}}
		{{- end}}
		);	
	}

	Map<String,dynamic>toJson() {
		final map = <String, dynamic>{};
    	{{- range .Fields -}}
		{{- if .IsMap }}
		map['{{.JSONNameTo}}'] = {{.Name}} == null ? null : json.decode(json.encode({{.Name}}));
		{{- else if and .IsRepeated .IsMessage}}
		map['{{.JSONNameTo}}'] = {{.Name}}?.map((l) => l.toJson())?.toList();
		{{- else if and .IsRepeated .IsEnum}}
		map['{{.JSONNameTo}}'] = {{.Name}}?.map((l) => to{{.InternalType}}JsonValue(l))?.toList();
		{{- else if .IsRepeated }}
		map['{{.JSONNameTo}}'] = {{.Name}}?.map((l) => l)?.toList();
		{{- else if and (.IsMessage) (eq .Type "DateTime")}}
		map['{{.JSONNameTo}}'] = {{.Name}}?.toIso8601String();
		{{- else if .IsMessage}}
		map['{{.JSONNameTo}}'] = {{.Name}}?.toJson();
		{{- else if .IsEnum}}
		map['{{.JSONNameTo}}'] = {{.Name}} == null ? null : to{{.Type}}JsonValue({{.Name}});
		{{- else}}
    	map['{{.JSONNameTo}}'] = {{.Name}};
    	{{- end}}
		{{- end}}
		return map;
	}

  @override
  String toString() {
    return json.encode(toJson());
  }
}
{{end -}}
{{end -}}

{{range .Services}}
abstract class {{.Name}} {
	{{- range .Methods}}
	Future<{{.OutputType}}>{{.Name}}({{.InputType}} {{.InputArg}});
    {{- end}}
}

class Default{{.Name}} implements {{.Name}} {
	final String hostname;
    Requester _requester;
	final _pathPrefix = "/twirp/{{.Package}}.{{.Name}}/";

    Default{{.Name}}(this.hostname, {Requester requester}) {
		if (requester == null) {
      		_requester = Requester(Client());
    	} else {
			_requester = requester;
		}
	}

	{{range .Methods}}
	@override
	Future<{{.OutputType}}>{{.Name}}({{.InputType}} {{.InputArg}}) async {
		final url = "$hostname${_pathPrefix}{{.Path}}";
		final uri = Uri.parse(url);
    	final request = Request('POST', uri);
    	request.body = json.encode({{.InputArg}}.toJson());
		request.headers['Content-Type'] = 'application/json'; // comes after body to fix https://github.com/dart-lang/http/issues/184
    	final response = await _requester.send(request);
		if (response.statusCode != 200) {
     		throw twirpException(response);
		}
		return compute(parse{{.OutputType}}, response.body);	
	}
{{end}}

	Exception twirpException(Response response) {
    	try {
      		final value = json.decode(response.body) as Map<String, dynamic>;
      		return TwirpJsonException.fromJson(value);
    	} catch (e) {
      		return TwirpException(response.body);
    	}
  	}
}

{{end}}

`

type EnumValue struct {
	EnumName          string
	Name              string
	Value             int32
	ParentMessageName string
}

type Enum struct {
	Name              string
	Values            []EnumValue
	ParentMessageName string
}

type Model struct {
	Name         string
	Primitive    bool
	Fields       []ModelField
	CanMarshal   bool
	CanUnmarshal bool
}

type ModelField struct {
	Name          string
	Type          string
	InternalType  string
	JSONNameTo    string
	JSONType      string
	IsMessage     bool
	IsRepeated    bool
	IsMap         bool
	MapKeyField   *ModelField
	MapValueField *ModelField
	IsEnum        bool
	JSONNameFrom  string
}

type Service struct {
	Name    string
	Package string
	Methods []ServiceMethod
}

type ServiceMethod struct {
	Name       string
	Path       string
	InputArg   string
	InputType  string
	OutputType string
}

func NewAPIContext() APIContext {
	ctx := APIContext{}
	ctx.modelLookup = make(map[string]*Model)
	ctx.enumLookup = make(map[string]*Enum)

	return ctx
}

type APIContext struct {
	Enums       []*Enum
	Models      []*Model
	Services    []*Service
	Imports     []Import
	modelLookup map[string]*Model
	enumLookup  map[string]*Enum
}

type Import struct {
	Path string
}

func (ctx *APIContext) AddEnum(e *Enum) {
	ctx.Enums = append(ctx.Enums, e)
	ctx.enumLookup[e.Name] = e
}

func (ctx *APIContext) AddModel(m *Model) {
	ctx.Models = append(ctx.Models, m)
	ctx.modelLookup[m.Name] = m
}

func (ctx *APIContext) ApplyImports(d *descriptor.FileDescriptorProto) {
	var deps []Import

	if len(ctx.Services) > 0 {
		deps = append(deps, Import{"dart:async"})
		deps = append(deps, Import{"package:http/http.dart"})
		deps = append(deps, Import{"package:flutter/foundation.dart"})
		deps = append(deps, Import{"package:requester/requester.dart"})
		deps = append(deps, Import{"package:twirp_dart_core/twirp_dart_core.dart"})
	}
	deps = append(deps, Import{"dart:convert"})

	for _, dep := range d.Dependency {
		if dep == "google/protobuf/timestamp.proto" {
			continue
		}
		importPath := path.Dir(dep)
		sourceDir := path.Dir(*d.Name)
		sourceComponents := strings.Split(sourceDir, fmt.Sprintf("%c", os.PathSeparator))
		distanceFromRoot := len(sourceComponents)
		for _, pathComponent := range sourceComponents {
			if strings.HasPrefix(importPath, pathComponent) {
				importPath = strings.TrimPrefix(importPath, pathComponent)
				distanceFromRoot--
			}
		}
		fileName := dartFilename(dep)
		fullPath := fileName
		fullPath = path.Join(importPath, fullPath)
		if distanceFromRoot > 0 {
			for i := 0; i < distanceFromRoot; i++ {
				fullPath = path.Join("..", fullPath)
			}
		}
		deps = append(deps, Import{fullPath})
	}
	ctx.Imports = deps
}

// ApplyMarshalFlags will inspect the CanMarshal and CanUnmarshal flags for models where
// the flags are enabled and recursively set the same values on all the models that are field types.

func (ctx *APIContext) ApplyMarshalFlags() {
	for _, m := range ctx.Models {
		for _, f := range m.Fields {
			// skip primitive types and WKT Timestamps
			if !f.IsMessage || f.Type == "DateTime" {
				continue
			}

			baseType := f.Type
			if f.IsRepeated {
				baseType = strings.Replace(baseType, "List<", "", -1)
				baseType = strings.Replace(baseType, ">", "", -1)
			}
			if m.CanMarshal {
				ctx.enableMarshal(ctx.modelLookup[baseType])
			}

			if m.CanUnmarshal {
				m, ok := ctx.modelLookup[baseType]
				if !ok {
					log.Fatalf("could not find model of type %s for field %s", baseType, f.Name)
				}
				ctx.enableUnmarshal(m)
			}
		}
	}
}

func (ctx *APIContext) enableMarshal(m *Model) {
	m.CanMarshal = true

	for _, f := range m.Fields {
		// skip primitive types and WKT Timestamps
		if !f.IsMessage || f.Type == "DateTime" {
			continue
		}
		mm, ok := ctx.modelLookup[f.Type]
		if !ok {
			log.Fatalf("could not find model of type %s for field %s", f.Type, f.Name)
		}
		ctx.enableMarshal(mm)
	}
}

func (ctx *APIContext) enableUnmarshal(m *Model) {
	m.CanUnmarshal = true

	for _, f := range m.Fields {
		// skip primitive types and WKT Timestamps
		if !f.IsMessage || f.Type == "DateTime" {
			continue
		}
		mm, ok := ctx.modelLookup[f.Type]
		if !ok {
			log.Fatalf("could not find model of type %s for field %s", f.Type, f.Name)
		}
		ctx.enableUnmarshal(mm)
	}
}

func CreateClientAPI(d *descriptor.FileDescriptorProto, generator *generator.Generator) (*plugin_go.CodeGeneratorResponse_File, error) {
	ctx := NewAPIContext()
	pkg := d.GetPackage()

	// Parse all the enums

	for _, e := range d.GetEnumType() {
		enum := &Enum{
			Name:              e.GetName(),
			ParentMessageName: "",
		}
		for _, v := range e.GetValue() {
			enum.Values = append(enum.Values, EnumValue{
				EnumName:          e.GetName(),
				Name:              *v.Name,
				Value:             *v.Number,
				ParentMessageName: "",
			})
		}
		ctx.AddEnum(enum)
	}

	// Parse all Messages for generating typescript interfaces

	for _, m := range d.GetMessageType() {
		model := &Model{
			Name: m.GetName(),
		}
		for _, f := range m.GetField() {
			model.Fields = append(model.Fields, newField(f, m, d, generator))
		}
		ctx.AddModel(model)

		for _, e := range m.GetEnumType() {
			enum := &Enum{
				Name:              e.GetName(),
				ParentMessageName: m.GetName(),
			}
			for _, v := range e.GetValue() {
				enum.Values = append(enum.Values, EnumValue{
					EnumName:          e.GetName(),
					Name:              *v.Name,
					Value:             *v.Number,
					ParentMessageName: m.GetName(),
				})
			}
			ctx.AddEnum(enum)
		}
	}

	// Parse all Services for generating typescript method interfaces and default client implementations
	for _, s := range d.GetService() {
		service := &Service{
			Name:    s.GetName(),
			Package: pkg,
		}

		for _, m := range s.GetMethod() {
			methodPath := m.GetName()
			methodName := strings.ToLower(methodPath[0:1]) + methodPath[1:]
			in := removePkg(m.GetInputType())
			arg := strings.ToLower(in[0:1]) + in[1:]

			method := ServiceMethod{
				Name:       methodName,
				Path:       methodPath,
				InputArg:   arg,
				InputType:  in,
				OutputType: removePkg(m.GetOutputType()),
			}

			service.Methods = append(service.Methods, method)
		}
		ctx.Services = append(ctx.Services, service)
	}
	// Only include the custom 'ToJSON' and 'JSONTo' methods in generated code
	// if the Model is part of an rpc method input arg or return type.
	for _, m := range ctx.Models {
		for _, s := range ctx.Services {
			for _, sm := range s.Methods {
				if m.Name == sm.InputType {
					m.CanMarshal = true
				}

				if m.Name == sm.OutputType {
					m.CanUnmarshal = true
				}
			}
		}
	}

	ctx.AddModel(&Model{
		Name:      "Date",
		Primitive: true,
	})

	ctx.ApplyImports(d)
	//ctx.ApplyMarshalFlags()

	funcMap := template.FuncMap{
		"stringify": stringify,
		"parse":     parse,
	}

	t, err := template.New("client_api").Funcs(funcMap).Parse(apiTemplate)
	if err != nil {
		return nil, err
	}

	b := bytes.NewBufferString("")
	err = t.Execute(b, ctx)
	if err != nil {
		return nil, err
	}

	cf := &plugin_go.CodeGeneratorResponse_File{}
	cf.Name = proto.String(dartModuleFilename(d))
	cf.Content = proto.String(b.String())

	return cf, nil
}

func newField(f *descriptor.FieldDescriptorProto,
	m *descriptor.DescriptorProto,
	d *descriptor.FileDescriptorProto,
	gen *generator.Generator) ModelField {
	dartType, internalType, jsonType := protoToDartType(f, m)
	fieldName := f.GetName()
	jsonNameFrom := fieldName
	jsonNameTo := camelCase(fieldName)
	name := camelCase(fieldName)

	field := ModelField{
		Name:         name,
		Type:         dartType,
		InternalType: internalType,
		JSONNameFrom: jsonNameFrom,
		JSONNameTo:   jsonNameTo,
		JSONType:     jsonType,
	}

	for _, nested := range m.GetNestedType() {
		if !strings.HasSuffix(f.GetTypeName(), nested.GetName()) {
			continue
		}
		keyField, valueField := nested.GetMapFields()
		if keyField != nil && valueField != nil {
			field.IsMap = true
			mapKeyField := newField(keyField, nested, d, gen)
			field.MapKeyField = &mapKeyField
			mapValueField := newField(valueField, nested, d, gen)
			field.MapValueField = &mapValueField
			field.Type = fmt.Sprintf("Map<%s,%s>", mapKeyField.Type, mapValueField.Type)
		}
	}
	field.IsMessage = f.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE
	field.IsEnum = f.GetType() == descriptor.FieldDescriptorProto_TYPE_ENUM
	field.IsRepeated = isRepeated(f)

	return field
}

// generates the (Type, JSONType) tuple for a ModelField so marshal/unmarshal functions
// will work when converting between TS interfaces and protobuf JSON.
func protoToDartType(f *descriptor.FieldDescriptorProto, m *descriptor.DescriptorProto) (string, string, string) {
	dartType := "String"
	jsonType := "string"
	internalType := "String"

	switch f.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		name := f.GetTypeName()
		dartType = removePkg(name)
		jsonType = "string"
		break
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		dartType = "double"
		jsonType = "number"
		break
	case descriptor.FieldDescriptorProto_TYPE_FIXED32,
		descriptor.FieldDescriptorProto_TYPE_FIXED64,
		descriptor.FieldDescriptorProto_TYPE_INT32,
		descriptor.FieldDescriptorProto_TYPE_INT64:
		dartType = "int"
		jsonType = "number"
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		dartType = "String"
		jsonType = "string"
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		dartType = "bool"
		jsonType = "boolean"
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		name := f.GetTypeName()

		// Google WKT Timestamp is a special case here:
		//
		// Currently the value will just be left as jsonpb RFC 3339 string.
		// JSON.stringify already handles serializing Date to its RFC 3339 format.
		//
		if name == ".google.protobuf.Timestamp" {
			dartType = "DateTime"
			jsonType = "string"
		} else {
			dartType = removePkg(name)
			jsonType = removePkg(name) + "JSON"
		}
	}
	internalType = dartType

	if isRepeated(f) {
		dartType = "List<" + dartType + ">"
		jsonType = jsonType + "[]"
	}

	return dartType, internalType, jsonType
}

func isRepeated(field *descriptor.FieldDescriptorProto) bool {
	return field.Label != nil && *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED
}

func removePkg(s string) string {
	p := strings.Split(s, ".")
	return p[len(p)-1]
}

func camelCase(s string) string {
	parts := strings.Split(s, "_")

	for i, p := range parts {
		if i == 0 {
			parts[i] = p
		} else {
			parts[i] = strings.ToUpper(p[0:1]) + strings.ToLower(p[1:])
		}
	}

	return strings.Join(parts, "")
}

func stringify(f ModelField) string {
	if f.IsRepeated {
		singularType := f.Type[0 : len(f.Type)-2] // strip array brackets from type

		if f.Type == "Date" {
			return fmt.Sprintf("m.%s.map((n) => n.toISOString())", f.Name)
		}

		if f.IsMessage {
			return fmt.Sprintf("m.%s.map(%sToJSON)", f.Name, singularType)
		}
	}

	if f.Type == "Date" {
		return fmt.Sprintf("m.%s.toISOString()", f.Name)
	}

	if f.IsMessage {
		return fmt.Sprintf("%sToJSON(m.%s)", f.Type, f.Name)
	}

	return "m." + f.Name
}

func parse(f ModelField, modelName string) string {
	field := "(((m as " + modelName + ")." + f.Name + ") ? (m as " + modelName + ")." + f.Name + " : (m as " + modelName + "JSON)." + f.JSONNameFrom + ")"
	if strings.Compare(f.Name, f.JSONNameFrom) == 0 {
		field = "m." + f.Name
	}

	if f.IsRepeated {
		singularType := f.Type[0 : len(f.Type)-2] // strip array brackets from type

		if f.Type == "Date" {
			return fmt.Sprintf("%s.map((n) => Date(n))", field)
		}

		if f.IsMessage {
			return fmt.Sprintf("%s.map(JSONTo%s)", field, singularType)
		}
	}

	if f.Type == "Date" {
		return fmt.Sprintf("Date(%s)", field)
	}

	if f.IsMessage {
		return fmt.Sprintf("JSONTo%s(%s)", f.Type, field)
	}

	return field
}
