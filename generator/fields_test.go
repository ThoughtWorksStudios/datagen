package generator

import (
	. "github.com/ThoughtWorksStudios/bobcat/common"
	. "github.com/ThoughtWorksStudios/bobcat/emitter"
	. "github.com/ThoughtWorksStudios/bobcat/test_helpers"
	"testing"
	"time"
)

func TestGenerateEntity(t *testing.T) {
	g := NewGenerator("testEntity", false)
	fieldType := &EntityType{g}
	emitter := NewTestEmitter()
	subId := fieldType.One("", emitter)

	e := emitter.Shift()

	if nil == e {
		t.Errorf("Expected to generate an entity but got %T %v", e, e)
	}

	AssertEqual(t, "testEntity", e["$type"], "Should have generated an entity of type \"testEntity\"")
	AssertEqual(t, subId, e["$id"])
}

func TestGenerateFloat(t *testing.T) {
	min, max := 4.25, 4.3
	FieldType := &FloatType{min, max}
	actual := FieldType.One("", NewDummyEmitter()).(float64)

	if actual < min || actual > max {
		t.Errorf("Generated value '%v' is outside of expected range min: '%v', max: '%v'", actual, min, max)
	}
}

func TestGenerateEnum(t *testing.T) {
	args := []interface{}{"one", "two", "three"}
	FieldType := &EnumType{values: args, size: int64(len(args))}
	actual := FieldType.One("", NewDummyEmitter()).(string)

	if actual != "one" && actual != "two" && actual != "three" {
		t.Errorf("Generated value '%v' enum value list: %v", actual, args)
	}
}

func TestMultiValueGenerate(t *testing.T) {
	field := NewField(&IntegerType{1, 10}, &CountRange{3, 3}, false)
	actual := len(field.GenerateValue("", NewDummyEmitter()).([]interface{}))

	AssertEqual(t, 3, actual)
}

func Test_NumberOfPossibilities_Integer(t *testing.T) {
	field := NewField(&IntegerType{1, 10}, nil, true)
	AssertEqual(t, int64(10), field.numberOfPossibilities())
}

func Test_NumberOfPossibilities_String(t *testing.T) {
	field := NewField(&StringType{length: 5}, nil, true)
	AssertEqual(t, int64(1073741824), field.numberOfPossibilities())
}

func Test_NumberOfPossibilities_Float(t *testing.T) {
	field := NewField(&FloatType{1.0, 2.0}, nil, true)
	AssertEqual(t, int64(-1), field.numberOfPossibilities())
}

func Test_NumberOfPossibilities_Float_WithSinglePossibility(t *testing.T) {
	field := NewField(&FloatType{1.0, 1.0}, nil, true)
	AssertEqual(t, int64(1), field.numberOfPossibilities())
}

func Test_NumberOfPossibilities_Bool(t *testing.T) {
	field := NewField(&BoolType{}, nil, true)
	AssertEqual(t, int64(2), field.numberOfPossibilities())
}

func Test_NumberOfPossibilities_Date(t *testing.T) {
	timeMin, _ := time.Parse("2006-01-02", "1945-01-01")
	timeMax, _ := time.Parse("2006-01-02", "1945-01-02")
	field := NewField(&DateType{timeMin, timeMax}, nil, true)
	AssertEqual(t, int64(86400), field.numberOfPossibilities())
}

func Test_NumberOfPossibilities_Enum(t *testing.T) {
	field := NewField(&EnumType{size: 4}, nil, true)
	AssertEqual(t, int64(4), field.numberOfPossibilities())
}

func Test_NumberOfPossibilities_Reference(t *testing.T) {
	gen := NewGenerator("Cat", false)
	gen.WithField("name", "string", int64(5), nil, true)
	eGen := ExtendGenerator("kitty", false, gen)
	field := eGen.fields["name"]

	AssertEqual(t, int64(1073741824), field.numberOfPossibilities())
}

func Test_NumberOfPossibilities_Dict(t *testing.T) {
	field := NewField(&DictType{category: "name_prefixes"}, nil, true)
	AssertEqual(t, int64(5), field.numberOfPossibilities())
}
