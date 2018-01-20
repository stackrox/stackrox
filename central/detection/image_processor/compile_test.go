package imageprocessor

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/central/detection/processors"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestCompileImageNameRuleRegex(t *testing.T) {
	regex, err := compileImageNamePolicyRegex(nil)
	assert.NoError(t, err)
	assert.Nil(t, regex)

	rule := &v1.ImageNamePolicy{
		Registry: ".*regisry.*",
	}

	regex, err = compileImageNamePolicyRegex(rule)
	assert.NoError(t, err)
	assert.NotNil(t, regex)

	rule.Repo = ".*repo.*"
	regex, err = compileImageNamePolicyRegex(rule)
	assert.NoError(t, err)
	assert.NotNil(t, regex)

	rule.Registry = "*"
	regex, err = compileImageNamePolicyRegex(rule)
	assert.Error(t, err)
}

func TestCompileStringRegex(t *testing.T) {
	regex, err := processors.CompileStringRegex("")
	assert.NoError(t, err)
	assert.Nil(t, regex)

	regex, err = processors.CompileStringRegex(".*")
	assert.NoError(t, err)
	assert.NotNil(t, regex)

	// Not a regex
	regex, err = processors.CompileStringRegex("*")
	assert.Error(t, err)
	assert.Nil(t, regex)
}

func TestCompileLineRuleFieldRegex(t *testing.T) {
	regex, err := compileLineRuleFieldRegex(nil)
	assert.NoError(t, err)
	assert.Nil(t, regex)

	// Happy path
	lineRule := &v1.DockerfileLineRuleField{
		Instruction: "CMD",
		Value:       ".*",
	}
	regex, err = compileLineRuleFieldRegex(lineRule)
	assert.NoError(t, err)
	assert.NotNil(t, regex)

	lineRule.Instruction = "BLAH"
	regex, err = compileLineRuleFieldRegex(lineRule)
	assert.Error(t, err)
	assert.Nil(t, regex)

	lineRule.Instruction = "CMD"
	lineRule.Value = ""
	regex, err = compileLineRuleFieldRegex(lineRule)
	assert.Error(t, err)
	assert.Nil(t, regex)
}
