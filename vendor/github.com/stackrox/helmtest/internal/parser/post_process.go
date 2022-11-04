package parser

import (
	"github.com/itchyny/gojq"
	"github.com/pkg/errors"
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
