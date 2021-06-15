package v1alpha1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing variables API", func() {
	var v *Variable

	Context("string variable values", func() {
		BeforeEach(func() {
			v = &Variable{
				VariablePayload: VariablePayload{
					ID:    "foo_id",
					Type:  "string",
					Value: "foo",
				},
			}
		})

		JustBeforeEach(func() {
			Expect(v.Value).To(BeEquivalentTo("foo"))
		})

		It("sets a non-empty string", func() {
			err := v.SetValue("bar")
			Expect(err).To(BeNil())
			Expect(v.Value).To(BeEquivalentTo("bar"))
		})

		It("denies a non-empty string", func() {
			err := v.SetValue("")
			Expect(err).ToNot(BeNil())
			Expect(v.Value).To(BeEquivalentTo("foo"))
		})
	})

	Context("string variable value selections", func() {
		BeforeEach(func() {
			v = &Variable{
				VariablePayload: VariablePayload{
					ID:    "beatles",
					Type:  "string",
					Value: "john",
					Selections: []ValueSelection{
						{
							"vocals",
							"john",
						},
						{
							"bass",
							"paul",
						},
						{
							"drums",
							"ringo",
						},
						{
							"guitar",
							"george",
						},
					},
				},
			}
		})

		JustBeforeEach(func() {
			Expect(v.Value).To(BeEquivalentTo("john"))
		})

		It("allowed values are used", func() {
			err := v.SetValue("ringo")
			Expect(err).To(BeNil())
			Expect(v.Value).To(BeEquivalentTo("ringo"))
		})

		It("denied values are not used", func() {
			err := v.SetValue("ringo_deathstarr")
			Expect(err).ToNot(BeNil())
			Expect(v.Value).To(BeEquivalentTo("john"))
		})
	})

	Context("bool variable values", func() {
		BeforeEach(func() {
			v = &Variable{
				VariablePayload: VariablePayload{
					ID:    "bool_test",
					Type:  "bool",
					Value: "true",
				},
			}
		})

		JustBeforeEach(func() {
			Expect(v.Value).To(BeEquivalentTo("true"))
		})

		It("true and false values are used", func() {
			err := v.SetValue("false")
			Expect(err).To(BeNil())
			Expect(v.Value).To(BeEquivalentTo("false"))
		})

		It("nonbool values are not used", func() {
			err := v.SetValue("xxx")
			Expect(err).ToNot(BeNil())
			Expect(v.Value).To(BeEquivalentTo("true"))
		})
	})

	Context("number variable value selections", func() {
		BeforeEach(func() {
			v = &Variable{
				VariablePayload: VariablePayload{
					ID:    "number_test",
					Type:  "number",
					Value: "42",
					Selections: []ValueSelection{
						{
							"fourty two",
							"42",
						},
						{
							"fourty two times two",
							"84",
						},
					},
				},
			}
		})

		JustBeforeEach(func() {
			Expect(v.Value).To(BeEquivalentTo("42"))
		})

		It("allowed values are used", func() {
			err := v.SetValue("84")
			Expect(err).To(BeNil())
			Expect(v.Value).To(BeEquivalentTo("84"))
		})

		It("disallowed values are not used", func() {
			err := v.SetValue("123")
			Expect(err).ToNot(BeNil())
			Expect(v.Value).To(BeEquivalentTo("42"))
		})
	})
})
