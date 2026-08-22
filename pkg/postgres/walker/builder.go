package walker

func MakeObjectGetter(value string, variable bool) ObjectGetter {
	return ObjectGetter{
		value:    value,
		variable: variable,
	}
}

func (o ObjectGetter) GetValue() string {
	return o.value
}

func (o ObjectGetter) GetVariable() bool {
	return o.variable
}

func (f *Field) SetReference(typeName, protoBufField string, noConstraint, restrictDelete, directional, nullable bool) {
	f.Options.Reference = &foreignKeyRef{
		TypeName:       typeName,
		ProtoBufField:  protoBufField,
		NoConstraint:   noConstraint,
		RestrictDelete: restrictDelete,
		Directional:    directional,
		Nullable:       nullable,
	}
}

func (f *Field) SetParentReference(parentSchema *Schema, columnName string) {
	f.Options.Reference = &foreignKeyRef{
		OtherSchema: parentSchema,
		ColumnName:  columnName,
	}
}

func (f *Field) HasReference() bool {
	return f.Options.Reference != nil
}

func (f *Field) RefTypeName() string {
	if f.Options.Reference == nil {
		return ""
	}
	return f.Options.Reference.TypeName
}

func (f *Field) RefProtoBufField() string {
	if f.Options.Reference == nil {
		return ""
	}
	return f.Options.Reference.ProtoBufField
}

func (f *Field) RefNoConstraint() bool {
	if f.Options.Reference == nil {
		return false
	}
	return f.Options.Reference.NoConstraint
}

func (f *Field) RefRestrictDelete() bool {
	if f.Options.Reference == nil {
		return false
	}
	return f.Options.Reference.RestrictDelete
}

func (f *Field) RefDirectional() bool {
	if f.Options.Reference == nil {
		return false
	}
	return f.Options.Reference.Directional
}

func (f *Field) RefNullable() bool {
	if f.Options.Reference == nil {
		return false
	}
	return f.Options.Reference.Nullable
}

func (f *Field) RefOtherSchema() *Schema {
	if f.Options.Reference == nil {
		return nil
	}
	return f.Options.Reference.OtherSchema
}

func (f *Field) RefColumnName() string {
	if f.Options.Reference == nil {
		return ""
	}
	return f.Options.Reference.ColumnName
}
