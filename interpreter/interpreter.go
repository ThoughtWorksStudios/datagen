package interpreter

import (
	"fmt"
	"github.com/ThoughtWorksStudios/datagen/dsl"
	"github.com/ThoughtWorksStudios/datagen/generator"
	"os"
	"strconv"
	"strings"
	"time"
)

type NamespaceCounter map[string]int

var AnonExtendNames NamespaceCounter = make(NamespaceCounter)

func (c NamespaceCounter) Next(key string) int {
	if ctr, hasKey := c[key]; hasKey {
		ctr += 1
		c[key] = ctr
		return ctr
	} else {
		c[key] = 1
		return 1
	}
}

func (c NamespaceCounter) NextAsStr(key string) string {
	return strconv.Itoa(c.Next(key))
}

func debug(format string, tokens ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", tokens...)
}

type Interpreter struct {
	entities map[string]*generator.Generator // TODO: should probably be a more generic symbol table or possibly the parent scope
	output   GenerationOutput
}

func New() *Interpreter {
	return &Interpreter{
		entities: make(map[string]*generator.Generator),
		output:   GenerationOutput{},
	}
}

func (i *Interpreter) WriteGeneratedContent(dest string, filePerEntity bool) error {
	if filePerEntity {
		return i.output.writeFilePerKey()
	} else {
		return i.output.writeToFile(dest)
	}
}

func (i *Interpreter) Visit(node dsl.Node) error {
	switch node.Kind {
	case "root":
		var err error
		node.Children.Each(func(env *dsl.IterEnv, node dsl.Node) {
			if err = i.Visit(node); err != nil {
				env.Halt()
			}
		})
		return err
	case "entity":
		_, err := i.EntityFromNode(node)
		return err
	case "generation":
		return i.GenerateFromNode(node)
	default:
		return node.Err("Unexpected token type %s", node.Kind)
	}
}

func (i *Interpreter) defaultArgumentFor(fieldType string) (interface{}, error) {
	switch fieldType {
	case "string":
		return 5, nil
	case "integer":
		return [2]int{1, 10}, nil
	case "decimal":
		return [2]float64{1, 10}, nil
	case "date":
		t1, _ := time.Parse("2006-01-02", "1945-01-01")
		t2, _ := time.Parse("2006-01-02", "2017-01-01")
		return [2]time.Time{t1, t2}, nil
	default:
		return nil, fmt.Errorf("Field of type `%s` requires arguments", fieldType)
	}
}

func (i *Interpreter) EntityFromNode(node dsl.Node) (*generator.Generator, error) {
	var entity *generator.Generator
	formalName := node.Name

	if node.HasRelation() {
		symbol := node.Related.ValStr()
		if parent, e := i.ResolveSymbol(*node.Related); nil == e {

			if formalName == "" {
				formalName = strings.Join([]string{"$" + AnonExtendNames.NextAsStr(symbol), symbol}, "::")
			}

			entity = generator.ExtendGenerator(formalName, parent.(*generator.Generator))
			entity.Base = symbol
		} else {
			formalName = "<anonymous>"
			return nil, fmt.Errorf("Cannot resolve parent entity %q for entity %q", symbol, formalName)
		}
	} else {
		if formalName == "" {
			formalName = "$" + AnonExtendNames.NextAsStr("$")
		}
		entity = generator.NewGenerator(formalName, nil)
	}

	for _, field := range node.Children {
		if field.Kind != "field" {
			return nil, field.Err("Expected a `field` declaration, but instead got `%s`", field.Kind) // should never get here
		}

		declType := field.Value.(dsl.Node).Kind

		switch {
		case declType == "builtin":
			if err := i.withDynamicField(entity, field); err != nil {
				return nil, field.WrapErr(err)
			}
		case strings.HasPrefix(declType, "literal-"):
			if err := i.withStaticField(entity, field); err != nil {
				return nil, field.WrapErr(err)
			}
		default:
			return nil, field.Err("Unexpected field type %s; field declarations must be either a built-in type or a literal value", declType)
		}
	}

	i.entities[formalName] = entity
	return entity, nil
}

func valStr(n dsl.Node) string {
	return n.Value.(string)
}

func valInt(n dsl.Node) int {
	return int(n.Value.(int64))
}

func valFloat(n dsl.Node) float64 {
	return n.Value.(float64)
}

func valTime(n dsl.Node) time.Time {
	return n.Value.(time.Time)
}

type Validator func(n dsl.Node) error

func assertValStr(n dsl.Node) error {
	if _, ok := n.Value.(string); !ok {
		return n.Err("Expected %v to be a string, but was %T.", n.Value, n.Value)
	}
	return nil
}

func assertValInt(n dsl.Node) error {
	if _, ok := n.Value.(int64); !ok {
		return n.Err("Expected %v to be an integer, but was %T.", n.Value, n.Value)
	}
	return nil
}

func assertValFloat(n dsl.Node) error {
	if _, ok := n.Value.(float64); !ok {
		return n.Err("Expected %v to be a decimal, but was %T.", n.Value, n.Value)
	}
	return nil
}

func assertValTime(n dsl.Node) error {
	if _, ok := n.Value.(time.Time); !ok {
		return n.Err("Expected %v to be a datetime, but was %T.", n.Value, n.Value)
	}
	return nil
}

func expectsArgs(num int, fn Validator, fieldType string, args dsl.NodeSet) error {
	if l := len(args); num != l {
		return args[0].Err("Field type `%s` expected %d args, but %d found.", fieldType, num, l)
	}

	var er error

	args.Each(func(env *dsl.IterEnv, node dsl.Node) {
		if er = fn(node); er != nil {
			env.Halt()
		}
	})

	return er
}

func (i *Interpreter) withStaticField(entity *generator.Generator, field dsl.Node) error {
	fieldValue := field.Value.(dsl.Node).Value
	return entity.WithStaticField(field.Name, fieldValue)
}

func (i *Interpreter) withDynamicField(entity *generator.Generator, field dsl.Node) error {
	var err error

	fieldType, ok := field.Value.(dsl.Node).Value.(string)
	if !ok {
		return field.Err("Could not parse field-type for field `%s`. Expected one of the builtin generator types, but instead got: %v", field.Name, field.Value.(dsl.Node).Value)
	}

	if 0 == len(field.Args) {
		arg, e := i.defaultArgumentFor(fieldType)
		if e != nil {
			return field.WrapErr(e)
		} else {
			return entity.WithField(field.Name, fieldType, arg)
		}
	}

	switch fieldType {
	case "integer":
		if err = expectsArgs(2, assertValInt, fieldType, field.Args); err == nil {
			return entity.WithField(field.Name, fieldType, [2]int{valInt(field.Args[0]), valInt(field.Args[1])})
		}
	case "decimal":
		if err = expectsArgs(2, assertValFloat, fieldType, field.Args); err == nil {
			return entity.WithField(field.Name, fieldType, [2]float64{valFloat(field.Args[0]), valFloat(field.Args[1])})
		}
	case "string":
		if err = expectsArgs(1, assertValInt, fieldType, field.Args); err == nil {
			return entity.WithField(field.Name, fieldType, valInt(field.Args[0]))
		}
	case "dict":
		if err = expectsArgs(1, assertValStr, fieldType, field.Args); err == nil {
			return entity.WithField(field.Name, fieldType, valStr(field.Args[0]))
		}
	case "date":
		if err = expectsArgs(2, assertValTime, fieldType, field.Args); err == nil {
			return entity.WithField(field.Name, fieldType, [2]time.Time{valTime(field.Args[0]), valTime(field.Args[1])})
		}
	}
	return err
}

func (i *Interpreter) ResolveSymbol(identiferNode dsl.Node) (interface{}, error) {
	g, ok := i.entities[identiferNode.ValStr()]
	if !ok {
		return nil, identiferNode.Err("Cannot resolve symbol %q", identiferNode.ValStr())
	}
	return g, nil
}

func (i *Interpreter) GenerateFromNode(generationNode dsl.Node) error {
	var entityGenerator *generator.Generator

	entity := generationNode.ValNode()
	switch entity.Kind {
	case "identifier":
		if g, e := i.ResolveSymbol(entity); nil != e {
			return e
		} else {
			entityGenerator = g.(*generator.Generator)
		}
	case "entity":
		if g, e := i.EntityFromNode(entity); e != nil {
			return e
		} else {
			entityGenerator = g
		}
	default:
		return generationNode.Err("Unexpected node type %q; node is %v", entity.Kind, entity)
	}

	if 0 == len(generationNode.Args) {
		return generationNode.Err("generate requires an argument")
	}
	count, ok := generationNode.Args[0].Value.(int64)

	if !ok {
		return generationNode.Err("generate %q takes an integer count", entityGenerator.Name)
	}

	if count < int64(1) {
		return generationNode.Err("Must generate at least 1 `%s` entity", entityGenerator.Name)
	}

	i.output.addAndAppend(entityGenerator.Name, entityGenerator.Generate(count))
	return nil
}
