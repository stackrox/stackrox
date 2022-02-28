package walker

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/protoreflect"
	"github.com/stackrox/rox/pkg/stringutils"
)

var (
	timestampType = reflect.TypeOf(&types.Timestamp{})
)

type context struct {
	getter         string
	column         string
	searchDisabled bool
}

func (c context) Getter(name string) string {
	get := fmt.Sprintf("Get%s()", name)
	if c.getter == "" {
		return get
	}
	return c.getter + "." + get
}

func (c context) Column(name string) string {
	if c.column == "" {
		return name
	}
	return c.column + "_" + name
}

func (c context) childContext(name string, searchDisabled bool) context {
	return context{
		getter:         c.Getter(name),
		column:         c.Column(name),
		searchDisabled: c.searchDisabled || searchDisabled,
	}
}

// Walk iterates over the obj and creates a search.Map object from the found struct tags
func Walk(obj reflect.Type, table string) *Schema {
	schema := &Schema{
		Table: table,
		Type:  obj.String(),
	}
	handleStruct(context{}, schema, obj.Elem())
	return schema
}

const defaultIndex = "btree"

func getPostgresOptions(tag string) PostgresOptions {
	var opts PostgresOptions

	for _, field := range strings.Split(tag, ",") {
		switch {
		case field == "-":
			opts.Ignored = true
		case strings.HasPrefix(field, "index"):
			if strings.Contains(field, "=") {
				opts.Index = stringutils.GetAfter(field, "=")
			} else {
				opts.Index = defaultIndex
			}
		case field == "pk":
			opts.PrimaryKey = true
		case field == "unique":
			opts.Unique = true
		case field == "":
		default:
			// ignore for just right now
			panic(fmt.Sprintf("unknown case: %s", field))
		}
	}
	return opts
}

func getSearchOptions(ctx context, searchTag string) SearchField {
	ignored := searchTag == "-"
	if ignored || searchTag == "" {
		return SearchField{
			Ignored: ignored,
		}
	}
	fields := strings.Split(searchTag, ",")
	return SearchField{
		FieldName: fields[0],
		Enabled:   !ctx.searchDisabled,
	}
}

var simpleFieldsMap = map[reflect.Kind]DataType{
	reflect.Map:    Map,
	reflect.String: String,
	reflect.Bool:   Bool,
}

func tableName(parent, child string) string {
	return fmt.Sprintf("%s_%s", parent, child)
}

// handleStruct takes in a struct object and properly handles all of the fields
func handleStruct(ctx context, schema *Schema, original reflect.Type) {
	for i := 0; i < original.NumField(); i++ {
		structField := original.Field(i)
		if strings.HasPrefix(structField.Name, "XXX") {
			continue
		}
		opts := getPostgresOptions(structField.Tag.Get("sql"))
		if opts.Ignored {
			continue
		}
		searchOpts := getSearchOptions(ctx, structField.Tag.Get("search"))

		field := Field{
			Schema:  schema,
			Name:    structField.Name,
			Search:  searchOpts,
			Type:    structField.Type.String(),
			Options: opts,
			ObjectGetter: ObjectGetter{
				value: ctx.Getter(structField.Name),
			},
			ColumnName: ctx.Column(structField.Name),
		}
		if dt, ok := simpleFieldsMap[structField.Type.Kind()]; ok {
			schema.AddFieldWithType(field, dt)
			continue
		}

		switch structField.Type.Kind() {
		case reflect.Ptr:
			if structField.Type == timestampType {
				schema.AddFieldWithType(field, DateTime)
				continue
			}

			handleStruct(ctx.childContext(field.Name, searchOpts.Ignored), schema, structField.Type.Elem())
		case reflect.Slice:
			elemType := structField.Type.Elem()

			switch elemType.Kind() {
			case reflect.String:
				schema.AddFieldWithType(field, StringArray)
				continue
			case reflect.Uint32, reflect.Uint64, reflect.Int32, reflect.Int64:
				schema.AddFieldWithType(field, IntArray)
				continue
			}

			childSchema := &Schema{
				ParentSchema: schema,
				Table:        tableName(schema.Table, field.Name),
				Type:         elemType.String(),
				ObjectGetter: ctx.Getter(field.Name),
			}
			idxField := Field{
				Schema: childSchema,
				Name:   "idx",
				ObjectGetter: ObjectGetter{
					variable: true,
					value:    "idx",
				},
				ColumnName: "idx",
				Type:       "int",
				Options: PostgresOptions{
					Ignored:    false,
					Index:      "btree",
					PrimaryKey: true,
				},
			}
			childSchema.AddFieldWithType(idxField, Numeric)

			// Take all the primary keys of the parent and copy them into the child schema
			// with references to the parent so we that we can create
			schema.Children = append(schema.Children, childSchema)

			handleStruct(context{searchDisabled: ctx.searchDisabled || searchOpts.Ignored}, childSchema, structField.Type.Elem().Elem())
			continue
		case reflect.Struct:
			handleStruct(ctx.childContext(field.Name, searchOpts.Ignored), schema, structField.Type)
		case reflect.Uint32, reflect.Uint64, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
			enum, ok := reflect.Zero(structField.Type).Interface().(protoreflect.ProtoEnum)
			if !ok {
				schema.AddFieldWithType(field, Numeric)
				continue
			}
			_, err := protoreflect.GetEnumDescriptor(enum)
			if err != nil {
				panic(err)
			}
			schema.AddFieldWithType(field, Enum)
			continue
		case reflect.Interface:
			// If it is a oneof then call XXX_OneofWrappers to get the types.
			// The return values is a slice of interfaces that are nil type pointers
			if structField.Tag.Get("protobuf_oneof") != "" {
				ptrToOriginal := reflect.PtrTo(original)

				methodName := fmt.Sprintf("Get%s", field.Name)
				oneofGetter, ok := ptrToOriginal.MethodByName(methodName)
				if !ok {
					panic("didn't find oneof function, did the naming change?")
				}
				oneofInterfaces := oneofGetter.Func.Call([]reflect.Value{reflect.New(original)})
				if len(oneofInterfaces) != 1 {
					panic(fmt.Sprintf("found %d interfaces returned from oneof getter", len(oneofInterfaces)))
				}

				oneofInterface := oneofInterfaces[0].Type()

				method, ok := ptrToOriginal.MethodByName("XXX_OneofWrappers")
				if !ok {
					panic(fmt.Sprintf("XXX_OneofWrappers should exist for all protobuf oneofs, not found for %s", original.Name()))
				}
				out := method.Func.Call([]reflect.Value{reflect.New(original)})
				actualOneOfFields := out[0].Interface().([]interface{})
				for _, f := range actualOneOfFields {
					typ := reflect.TypeOf(f)
					if typ.Implements(oneofInterface) {
						handleStruct(ctx, schema, typ.Elem())
					}
				}
				continue
			}
			panic("non-oneof interface is not handled")
		default:
			panic(fmt.Sprintf("Type %s for field %s is not currently handled", original.Kind(), field.Name))
		}
	}
}
