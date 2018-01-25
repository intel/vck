package reify

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

type reifyTestCase struct {
	description          string
	template             string
	templateValues       interface{}
	globalTemplateValues map[string]string
	expectedError        error
	expectedResult       string
}

func TestReify(t *testing.T) {
	testCases := []reifyTestCase{
		// Valid cases.
		{
			description:          "empty template, no values",
			template:             "",
			templateValues:       nil,
			globalTemplateValues: map[string]string{},
			expectedError:        nil,
			expectedResult:       "null",
		},
		{
			description:          "non-empty template, no values",
			template:             `a: b`,
			templateValues:       nil,
			globalTemplateValues: map[string]string{},
			expectedError:        nil,
			expectedResult:       `{"a":"b"}`,
		},
		{
			description:          "expansion from values object",
			template:             `a: {{.X}}`,
			templateValues:       struct{ X string }{"b"},
			globalTemplateValues: map[string]string{},
			expectedError:        nil,
			expectedResult:       `{"a":"b"}`,
		},
		{
			description:          "expansion from global values object",
			template:             `a: {{ GlobalTemplateValue "X" }}`,
			templateValues:       nil,
			globalTemplateValues: map[string]string{"X": "b"},
			expectedError:        nil,
			expectedResult:       `{"a":"b"}`,
		},
		{
			description:          "expansion of resource string from values object",
			template:             `"amount": "{{ ResourceString .Resource }}"`,
			templateValues:       struct{ Resource resource.Quantity }{resource.MustParse("250m")},
			globalTemplateValues: map[string]string{},
			expectedError:        nil,
			expectedResult:       `{"amount":"250m"}`,
		},

		// Invalid cases.
		{
			description:          "invalid yaml syntax",
			template:             `"a" "b"`,
			templateValues:       nil,
			globalTemplateValues: map[string]string{},
			expectedError:        fmt.Errorf("yaml: did not find expected <document start>"),
			expectedResult:       "",
		},
		{
			description:          "invalid template syntax",
			template:             `a: {{ .X }"`,
			templateValues:       nil,
			globalTemplateValues: map[string]string{},
			expectedError:        fmt.Errorf(`unexpected "}" in operand`),
			expectedResult:       "",
		},
		{
			description:          "invalid template value reference",
			template:             `"a": "{{ .X }}"`,
			templateValues:       struct{}{},
			globalTemplateValues: map[string]string{},
			expectedError:        fmt.Errorf("at <.X>: can't evaluate field X in type struct {}"),
			expectedResult:       "",
		},
	}

	for index, tc := range testCases {
		// Write template data to temporary file.
		templateFile, err := ioutil.TempFile("", fmt.Sprintf("TestReify-%d", index))
		if err != nil {
			t.Fatal(err.Error())
		}
		defer os.Remove(templateFile.Name())
		_, err = templateFile.WriteString(tc.template)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Logf("TestReify: %s", tc.description)
		result, err := Reify(templateFile.Name(), tc.templateValues, tc.globalTemplateValues)
		if err != tc.expectedError {
			if err != nil && tc.expectedError != nil && strings.Contains(err.Error(), tc.expectedError.Error()) {
				// Do nothing; although inequal, the error contains the expected error text.
			} else {
				t.Errorf("expected error [%v] but got [%v]", tc.expectedError, err)
			}
		}

		resultStr := string(result)
		if resultStr != tc.expectedResult {
			t.Errorf("expected result:\n%s\n\nbut got:\n%s\n", tc.expectedResult, resultStr)
		}
	}
}
