package generator

import (
	"fmt"
	. "github.com/ThoughtWorksStudios/bobcat/common"
	. "github.com/ThoughtWorksStudios/bobcat/emitter"
	"strings"
	"time"
)

var DEFAULT_PK_CONFIG *PrimaryKey = &PrimaryKey{name: "$id", kind: "uid"}

type Generator struct {
	name            string
	extends         string
	declaredType    string
	fields          FieldSet
	disableMetadata bool
	pkey            *PrimaryKey
}

func (g *Generator) HasField(fieldName string) bool {
	_, ok := g.fields[fieldName]
	return ok
}

func ExtendGenerator(name string, parent *Generator, pkey *PrimaryKey, disableMetadata bool) *Generator {
	var gen *Generator

	if pkey == nil {
		gen = NewGenerator(name, parent.pkey, disableMetadata)
		parent.pkey.Inherit(gen, parent) // must reference parent's primary key; this is important for serial fields to continue the sequence
	} else {
		gen = NewGenerator(name, pkey, disableMetadata)
	}

	gen.extends = parent.Type()

	gen.recalculateType()

	if !disableMetadata {
		gen.fields["$extends"] = NewField(&LiteralType{value: gen.extends}, nil, false)
		gen.fields["$type"] = NewField(&LiteralType{value: gen.Type()}, nil, false)
	}

	for key, f := range parent.fields {
		if _, hasField := gen.fields[key]; !hasField && !strings.HasPrefix(key, "$") && key != parent.PrimaryKeyName() {
			gen.fields[key] = NewField(&ReferenceType{referred: parent, fieldName: key}, f.count, false)
		}
	}

	return gen
}

func NewGenerator(name string, pkey *PrimaryKey, disableMetadata bool) *Generator {
	if name == "" {
		name = "$"
	}

	g := &Generator{name: name, fields: make(FieldSet), disableMetadata: disableMetadata}

	g.recalculateType()

	if !disableMetadata {
		g.fields["$type"] = NewField(&LiteralType{value: g.Type()}, nil, false)
	}

	if pkey == nil {
		pkey = DEFAULT_PK_CONFIG
	}

	pkey.Attach(g)

	return g
}

func (g *Generator) PrimaryKeyName() string {
	return g.pkey.name
}

func (g *Generator) newStaticField(fieldName, fieldValue interface{}) *Field {
	return NewField(&LiteralType{value: fieldValue}, nil, false)
}

func (g *Generator) WithDeferredField(fieldName string, fieldValue DeferredResolver) error {
	g.fields[fieldName] = NewField(&DeferredType{closure: fieldValue}, nil, false)
	return nil
}

func (g *Generator) WithLiteralField(fieldName string, fieldValue interface{}) error {
	g.fields[fieldName] = g.newStaticField(fieldName, fieldValue)
	return nil
}

func (g *Generator) WithEntityField(fieldName string, entityGenerator *Generator, fieldArgs interface{}, countRange *CountRange) error {
	g.fields[fieldName] = NewField(&EntityType{entityGenerator: entityGenerator}, countRange, false)
	return nil
}

func (g *Generator) newFieldType(fieldName, fieldType string, fieldArgs interface{}, countRange *CountRange, uniqueValue bool) (*Field, error) {
	switch fieldType {
	case "string":
		if ln, ok := fieldArgs.(int64); ok {
			return NewField(&StringType{length: ln}, countRange, uniqueValue), nil
		} else {
			return nil, fmt.Errorf("expected field args to be of type 'int' for field %s (%s), but got %v",
				fieldName, fieldType, fieldArgs)
		}
	case "integer":
		if bounds, ok := fieldArgs.([2]int64); ok {
			min, max := bounds[0], bounds[1]
			if max < min {
				return nil, fmt.Errorf("max %v cannot be less than min %v", max, min)
			}

			return NewField(&IntegerType{min: min, max: max}, countRange, uniqueValue), nil
		} else {
			return nil, fmt.Errorf("expected field args to be of type '(min:int, max:int)' for field %s (%s), but got %v", fieldName, fieldType, fieldArgs)
		}
	case "decimal":
		if bounds, ok := fieldArgs.([2]float64); ok {
			min, max := bounds[0], bounds[1]
			if max < min {
				return nil, fmt.Errorf("max %v cannot be less than min %v", max, min)
			}
			return NewField(&FloatType{min: min, max: max}, countRange, uniqueValue), nil
		} else {
			return nil, fmt.Errorf("expected field args to be of type '(min:float64, max:float64)' for field %s (%s), but got %v", fieldName, fieldType, fieldArgs)
		}
	case "date":
		if bounds, ok := fieldArgs.([]interface{}); ok {
			min, max, format := bounds[0].(time.Time), bounds[1].(time.Time), bounds[2].(string)
			dateType := &DateType{min: min, max: max, format: format}
			if !dateType.ValidBounds() {
				return nil, fmt.Errorf("max %v cannot be before min %v", max, min)
			}
			return NewField(dateType, countRange, uniqueValue), nil
		} else {
			return nil, fmt.Errorf("expected field args to be of type 'time.Time' for field %s (%s), but got %v", fieldName, fieldType, fieldArgs)
		}
	case "uid":
		return NewField(&MongoIDType{}, nil, false), nil
	case "bool":
		if uniqueValue {
			return nil, fmt.Errorf("boolean fields cannot be unique")
		}
		return NewField(&BoolType{}, countRange, false), nil
	case "dict":
		if dict, ok := fieldArgs.(string); ok {
			return NewField(&DictType{category: dict}, countRange, uniqueValue), nil
		} else {
			return nil, fmt.Errorf("expected field args to be of type 'string' for field %s (%s), but got %v", fieldName, fieldType, fieldArgs)
		}
	case "enum":
		if values, ok := fieldArgs.([]interface{}); ok {
			return NewField(&EnumType{values: values, size: int64(len(values))}, countRange, uniqueValue), nil
		} else {
			return nil, fmt.Errorf("expected field args to be of type 'collection' for field %s (%s), but got %v", fieldName, fieldType, fieldArgs)
		}
	case "serial":
		if countRange != nil {
			return nil, fmt.Errorf("serial fields can only have a single value")
		}
		return NewField(&SerialType{}, nil, false), nil
	default:
		return nil, fmt.Errorf("Invalid field type '%v'", fieldType)
	}

	return nil, nil
}

func (g *Generator) WithField(fieldName, fieldType string, fieldArgs interface{}, countRange *CountRange, uniqueValue bool) error {
	if field, err := g.newFieldType(fieldName, fieldType, fieldArgs, countRange, uniqueValue); err == nil {
		g.fields[fieldName] = field
	} else {
		return err
	}
	return nil
}

func (g *Generator) WithStaticDistribution(fieldName, distribution string, fieldValues []interface{}, weights []float64) error {
	distributionType := g.newDistribution(distribution, weights)

	if !distributionType.isCompatibleDomain("literal") {
		return fmt.Errorf("Invalid distribution Domain: %v is not a valid domain for %v distributions", "static", distributionType.Type())
	}

	if !distributionType.supportsMultipleDomains() && len(fieldValues) > 1 {
		return fmt.Errorf("Distribution does not support multiple domains")
	}

	bins := make([]*Field, len(fieldValues))
	for i, fieldValue := range fieldValues {
		bins[i] = g.newStaticField(fieldName, fieldValue)
	}

	g.fields[fieldName] = NewField(&DistributionType{bins: bins, dist: distributionType}, nil, false)
	return nil
}

func (g *Generator) WithDistribution(fieldName, distribution, fieldType string, fieldArgs []interface{}, weights []float64) error {
	distributionType := g.newDistribution(distribution, weights)

	if !distributionType.supportsMultipleDomains() && len(fieldArgs) > 1 {
		return fmt.Errorf("Distribution does not support multiple domains")
	}

	bins := make([]*Field, len(fieldArgs))

	for i, fieldArg := range fieldArgs {
		if field, err := g.newFieldType(fieldName, fieldType, fieldArg, nil, false); err == nil {
			if i == 0 && !distributionType.isCompatibleDomain(field.Type()) {
				return fmt.Errorf("Invalid distribution Domain: %v is not a valid domain for %v distributions", field.Type(), distributionType.Type())
			}
			bins[i] = field
		} else {
			return err
		}
	}
	g.fields[fieldName] = NewField(&DistributionType{bins: bins, dist: distributionType}, nil, false)

	return nil
}

func (g *Generator) newDistribution(distType string, weights []float64) Distribution {
	switch distType {
	case "normal":
		return &NormalDistribution{}
	case "uniform":
		return &UniformDistribution{}
	case "weighted":
		return &WeightedDistribution{weights: weights}
	case "percent":
		return &PercentageDistribution{weights: weights, bins: make([]int64, len(weights))}
	default:
		return &UniformDistribution{}
	}
}

func (g *Generator) Type() string {
	return g.declaredType
}

func (g *Generator) recalculateType() {
	if (strings.HasPrefix(g.name, "$") || g.name == "") && g.extends != "" {
		g.declaredType = g.extends
	} else {
		g.declaredType = g.name
	}
}

func (g *Generator) EnsureGeneratable(count int64) error {
	for name, field := range g.fields {
		if field.Uniquable() && field.UniqueValue {
			numberOfPossibilities := field.numberOfPossibilities()
			if numberOfPossibilities != int64(-1) && numberOfPossibilities < count {
				return fmt.Errorf("Not enough unique values for field '%v': There are only %v unique values available for the '%v' field, and you're trying to generate %v entities", name, numberOfPossibilities, name, count)
			}
		}
	}
	return nil
}

func (g *Generator) Generate(count int64, emitter Emitter, scope *Scope) []interface{} {
	ids := make([]interface{}, count)
	idKey := g.PrimaryKeyName()
	for i := int64(0); i < count; i++ {
		ids[i] = g.One(nil, emitter, scope)[idKey]
	}
	return ids
}

func (g *Generator) One(parentId interface{}, emitter Emitter, scope *Scope) EntityResult {
	entity := EntityResult{}
	childScope := TransientScope(scope, SymbolTable(entity))

	idKey := g.PrimaryKeyName()
	id := g.fields[idKey].GenerateValue(nil, emitter, childScope)
	entity[idKey] = id // create this first because we may use it as the parentId when generating child entities

	if parentId != nil {
		entity["$parent"] = parentId
	}

	for name, field := range g.fields {
		if field.fieldType.Type() != "deferred" {
			if name != idKey { // don't GenerateValue() more than once for id -- it throws off the sequence for serial types
				// GenerateValue MAY populate entity[name] for entity fields
				value := field.GenerateValue(id, emitter.NextEmitter(entity, name, field.MultiValue()), childScope)
				if _, isAlreadySet := entity[name]; !isAlreadySet {
					entity[name] = value
					childScope.SetSymbol(name, value)
				}
			}
		}
	}

	// these fields rely on previously generated field values
	for name, field := range g.fields {
		if field.fieldType.Type() == "deferred" {
			entity[name] = field.GenerateValue(id, emitter.NextEmitter(entity, name, field.MultiValue()), childScope)
			childScope.SetSymbol(name, entity[name])
		}
	}

	emitter.Emit(entity, g.Type())
	return entity
}

func (g *Generator) String() string {
	return fmt.Sprintf("%s{}", g.name)
}
