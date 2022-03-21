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
	timestampType = reflect.TypeOf((*types.Timestamp)(nil))
)

type context struct {
	getter         string
	column         string
	searchDisabled bool
	ignorePK       bool
	ignoreUnique   bool
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

func (c context) childContext(name string, searchDisabled bool, opts PostgresOptions) context {
	return context{
		getter:         c.Getter(name),
		column:         c.Column(name),
		searchDisabled: c.searchDisabled || searchDisabled,
		ignorePK:       c.ignorePK || opts.IgnorePrimaryKey,
		ignoreUnique:   c.ignoreUnique || opts.IgnoreUniqueConstraint,
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

func getPostgresOptions(tag string, topLevel bool, ignorePK, ignoreUnique bool) PostgresOptions {
	opts := PostgresOptions{
		IgnorePrimaryKey:       ignorePK,
		IgnoreUniqueConstraint: ignoreUnique,
	}

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
		case field == "ignore_unique":
			// if this is an embedded entity that defines a unique constraint, then we want to ignore it as
			// it is not a unique constraint at the top level
			opts.IgnoreUniqueConstraint = true
		case field == "ignore_pk":
			// if this is an embedded entity with a primary key of its own, we do not want to use it as a
			// primary key since the owning entity's primary key is what we'd like to use
			opts.IgnorePrimaryKey = true

		case field == "pk":
			// if we have a child object, we don't want to propagate its primary key
			// an example of this is process_indicator.  It is its own object
			// at which times it needs to use a primary key.  It can also be a child of
			// alerts.  When it is a child of alerts, we want to ignore the pk of the
			// process_indicator in favor of the parent_id and generated idx field.
			opts.PrimaryKey = topLevel && !ignorePK
		case field == "unique":
			opts.Unique = !ignoreUnique
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

func typeIsEnum(typ reflect.Type) bool {
	enum, ok := reflect.Zero(typ).Interface().(protoreflect.ProtoEnum)
	if !ok {
		return false
	}
	_, err := protoreflect.GetEnumDescriptor(enum)
	if err != nil {
		panic(err)
	}
	return true
}

// handleStruct takes in a struct object and properly handles all of the fields
func handleStruct(ctx context, schema *Schema, original reflect.Type) {
	for i := 0; i < original.NumField(); i++ {
		structField := original.Field(i)
		if strings.HasPrefix(structField.Name, "XXX") {
			continue
		}
		opts := getPostgresOptions(structField.Tag.Get("sql"), len(schema.Parents) == 0, ctx.ignorePK, ctx.ignoreUnique)

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

			handleStruct(ctx.childContext(field.Name, searchOpts.Ignored, opts), schema, structField.Type.Elem())
		case reflect.Slice:
			elemType := structField.Type.Elem()

			switch elemType.Kind() {
			case reflect.String:
				schema.AddFieldWithType(field, StringArray)
				continue
			case reflect.Uint8:
				schema.AddFieldWithType(field, Bytes)
				continue
			case reflect.Uint32, reflect.Uint64, reflect.Int32, reflect.Int64:
				if typeIsEnum(elemType) {
					schema.AddFieldWithType(field, EnumArray)
				} else {
					schema.AddFieldWithType(field, IntArray)
				}
				continue
			}

			childSchema := &Schema{
				Parents:      []*Schema{schema},
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
			childSchema.AddFieldWithType(idxField, Integer)

			// Take all the primary keys of the parent and copy them into the child schema
			// with references to the parent so we that we can create
			schema.Children = append(schema.Children, childSchema)
			handleStruct(context{searchDisabled: ctx.searchDisabled || searchOpts.Ignored, ignorePK: opts.IgnorePrimaryKey, ignoreUnique: opts.IgnoreUniqueConstraint}, childSchema, structField.Type.Elem().Elem())
		case reflect.Struct:
			handleStruct(ctx.childContext(field.Name, searchOpts.Ignored, opts), schema, structField.Type)
		case reflect.Uint8:
			schema.AddFieldWithType(field, Bytes)
		case reflect.Uint32, reflect.Uint64, reflect.Int32, reflect.Int64:
			if typeIsEnum(structField.Type) {
				schema.AddFieldWithType(field, Enum)
			} else {
				schema.AddFieldWithType(field, Integer)
			}
		case reflect.Float32, reflect.Float64:
			schema.AddFieldWithType(field, Numeric)
		case reflect.Interface:
			// If it is a oneof then call XXX_OneofWrappers to get the types.
			// The return values is a slice of interfaces that are nil type pointers
			if structField.Tag.Get("protobuf_oneof") == "" {
				panic("non-oneof interface is not handled")

			}
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
		default:
			panic(fmt.Sprintf("Type %s for field %s is not currently handled", original.Kind(), field.Name))
		}
	}
}
