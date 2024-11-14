package token_counter

import (
	"fmt"
	"slices"
	"strings"

	"bigkinds.or.kr/conversation/model"
)

func descriptionToSystemPrompt(description string) string {
	prompt := ""
	description = strings.TrimSpace(description)
	for _, line := range strings.Split(description, "\n") {
		line = strings.TrimSpace(line)
		prompt += "//"
		if len(line) > 0 {
			prompt += fmt.Sprintf(" %s", strings.TrimSpace(line))
		}
		prompt += "\n"
	}
	return prompt
}

func functionParametersToSystemPrompt(parameters map[string]interface{}) string {
	prompt := ""

	t, ok := parameters["type"].(string)
	if ok && t == "object" {
		prompt += "{\n"

		required, ok := parameters["required"].([]string)
		if !ok {
			required = []string{}
		}

		properties, ok := parameters["properties"].(map[string]interface{})
		if ok {
			keys := make([]string, 0, len(properties))
			for k := range properties {
				keys = append(keys, k)
			}
			slices.Sort(keys)

			for _, key := range keys {
				propertyMap, ok := properties[key].(map[string]interface{})
				if !ok {
					continue
				}
				description, ok := propertyMap["description"].(string)
				if ok {
					prompt += descriptionToSystemPrompt(description)
				}
				prompt += key
				if !slices.Contains(required, key) {
					prompt += "?"
				}
				prompt += ": "
				prompt += functionParametersToSystemPrompt(propertyMap)
				prompt += ",\n"
			}
		}

		prompt += "}"
	} else {
		if t == "string" {
			enums, ok := parameters["enum"].([]string)
			if ok {
				for i, e := range enums {
					enums[i] = fmt.Sprintf(`"%s"`, e)
				}
				prompt += strings.Join(enums, " | ")
			} else {
				prompt += "string"
			}
		} else {
			prompt += t
		}
	}

	return prompt
}

func FunctionDefinitionToSystemPrompt(definition model.Function) string {
	prompt := ""

	prompt += descriptionToSystemPrompt(definition.Description)

	prompt += fmt.Sprintf("type %s = (", definition.Name)
	if len(definition.Parameters) > 0 {
		prompt += "_: "
		prompt += functionParametersToSystemPrompt(definition.Parameters)

	}
	prompt += ") => any;"

	return prompt
}

func FunctionCallResponseToGPTRawOutput(name string, arguments string) string {
	return fmt.Sprintf(`functions.%s(%s)`, name, arguments)
}
