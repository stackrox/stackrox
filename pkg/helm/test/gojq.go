package test

import (
	"fmt"

	"github.com/itchyny/gojq"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var (
	errAssumptionViolation = errors.New("assumption violation")
)

// postProcessQuery processes query to allow the realization of the special `assertThat` function.
// Generally, functions defined via `def` may receive filters as arguments, but builtin functions will only ever see
// the concrete values. We therefore patch the AST to change every invocation of `assertThat` such that it passes
// the string representation of the first argument as the second argument. This allows us to give a more specific
// error message, stating what the predicate was that caused the assertion to be violated.
func postProcessQuery(query *gojq.Query) error {
	if query == nil {
		return nil
	}

	if err := postProcessQuery(query.Left); err != nil {
		return err
	}
	if err := postProcessQuery(query.Right); err != nil {
		return err
	}
	if query.Term != nil {
		if fn := query.Term.Func; fn != nil && fn.Name == "assertThat" && len(fn.Args) != 2 {
			if len(fn.Args) != 1 {
				return errors.Errorf("incorrect number of arguments for assertThat: %d, expected 1 or 2", len(fn.Args))
			}
			filterStr := &gojq.Query{
				Term: &gojq.Term{
					Type: gojq.TermTypeString,
					Str: &gojq.String{
						Str: query.Term.Func.Args[0].String(),
					},
				},
			}
			fn.Args = append(fn.Args, filterStr)
		}

		if arr := query.Term.Array; arr != nil {
			if err := postProcessQuery(arr.Query); err != nil {
				return err
			}
		}
		if un := query.Term.Unary; un != nil {
			if err := postProcessQuery(un.Term.Query); err != nil {
				return err
			}
		}

		if err := postProcessQuery(query.Term.Query); err != nil {
			return err
		}
		if lbl := query.Term.Label; lbl != nil {
			if err := postProcessQuery(lbl.Body); err != nil {
				return err
			}
		}
		for _, suff := range query.Term.SuffixList {
			if bind := suff.Bind; bind != nil {
				if err := postProcessQuery(bind.Body); err != nil {
					return err
				}
			}
		}
	}

	for _, fd := range query.FuncDefs {
		if err := postProcessQuery(fd.Body); err != nil {
			return err
		}
	}
	return nil
}

func gojqParse(src string) (*gojq.Query, error) {
	query, err := gojq.Parse(src)
	if err != nil {
		return nil, err
	}
	if err := postProcessQuery(query); err != nil {
		return nil, err
	}
	return query, nil
}

func gojqCompile(query *gojq.Query, compilerOpts ...gojq.CompilerOption) (*gojq.Code, error) {
	var allOpts []gojq.CompilerOption
	if len(compilerOpts) == 0 {
		allOpts = builtinOpts
	} else {
		allOpts = make([]gojq.CompilerOption, 0, len(compilerOpts)+len(builtinOpts))
		allOpts = append(allOpts, builtinOpts...)
		allOpts = append(allOpts, compilerOpts...)
	}
	return gojq.Compile(query, allOpts...)
}

// Additional built-ins for gojq

func fromYaml(obj interface{}, _ []interface{}) interface{} {
	var bytes []byte
	switch o := obj.(type) {
	case []byte:
		bytes = o
	case string:
		bytes = []byte(o)
	default:
		return errors.Errorf("expected string or bytes as input to fromyaml, got %T", obj)
	}

	var out map[string]interface{}
	if err := yaml.Unmarshal(bytes, &out); err != nil {
		return errors.Wrap(err, "fromyaml")
	}
	return out
}

func toYaml(obj interface{}, _ []interface{}) interface{} {
	bytes, err := yaml.Marshal(obj)
	if err != nil {
		return errors.Wrap(err, "toyaml")
	}
	return string(bytes)
}

func printjq(obj interface{}, _ []interface{}) interface{} {
	fmt.Println("--------DEBUG--------")
	fmt.Println(obj)
	return obj
}

func assertThat(obj interface{}, args []interface{}) interface{} {
	evalResult := args[0]
	if !truthiness(evalResult) {
		return errors.Errorf("%+v failed predicate '%s'", obj, args[1])
	}
	return true
}

func assumeThat(obj interface{}, args []interface{}) interface{} {
	evalResult := args[0]
	if !truthiness(evalResult) {
		return errAssumptionViolation
	}
	return obj
}

func assertNotExist(obj interface{}, _ []interface{}) interface{} {
	return errors.Errorf("object %+v should not exist", obj)
}

var builtinOpts = []gojq.CompilerOption{
	gojq.WithFunction("fromyaml", 0, 0, fromYaml),
	gojq.WithFunction("toyaml", 0, 0, toYaml),
	gojq.WithFunction("assertThat", 2, 2, assertThat),
	gojq.WithFunction("assertNotExist", 0, 0, assertNotExist),
	gojq.WithFunction("assumeThat", 1, 1, assumeThat),
	gojq.WithFunction("print", 0, 0, printjq),
}
