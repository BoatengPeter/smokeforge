package patch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

func patchJSON(data []byte, operation, pathValue string, value any) ([]byte, error) {
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	updated, err := applyTreePatch(document, operation, pathValue, value)
	if err != nil {
		return nil, err
	}

	out, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}

	return append(out, '\n'), nil
}

func patchYAML(data []byte, operation, pathValue string, value any) ([]byte, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	if len(document.Content) == 0 {
		return nil, errors.New("parse yaml: empty document")
	}

	root := document.Content[0]

	segments, err := splitPath(pathValue)
	if err != nil {
		return nil, err
	}

	switch operation {
	case "set":
		err = yamlSet(root, segments, value)
	case "merge":
		err = yamlMerge(root, segments, value)
	case "append":
		err = yamlAppend(root, segments, value)
	case "delete":
		err = yamlDelete(root, segments)
	default:
		err = fmt.Errorf("unknown operation %q", operation)
	}

	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	if err := encoder.Encode(&document); err != nil {
		return nil, fmt.Errorf("encode yaml: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("close yaml encoder: %w", err)
	}

	return buf.Bytes(), nil
}

func patchTOML(data []byte, operation, pathValue string, value any) ([]byte, error) {
	var document map[string]any
	if err := toml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse toml: %w", err)
	}

	updated, err := applyTreePatch(document, operation, pathValue, value)
	if err != nil {
		return nil, err
	}

	mapping, ok := updated.(map[string]any)
	if !ok {
		return nil, errors.New("patch toml: document root must remain a table")
	}

	out, err := toml.Marshal(mapping)
	if err != nil {
		return nil, fmt.Errorf("marshal toml: %w", err)
	}

	var check map[string]any
	if err := toml.Unmarshal(out, &check); err != nil {
		return nil, fmt.Errorf("validate toml: %w", err)
	}

	return out, nil
}

func applyTreePatch(root any, operation, pathValue string, value any) (any, error) {
	segments, err := splitPath(pathValue)
	if err != nil {
		return nil, err
	}

	switch operation {
	case "set":
		return setTree(root, segments, value)
	case "merge":
		return mergeTree(root, segments, value)
	case "append":
		return appendTree(root, segments, value)
	case "delete":
		return deleteTree(root, segments)
	default:
		return nil, fmt.Errorf("unknown operation %q", operation)
	}
}

func splitPath(pathValue string) ([]string, error) {
	if strings.TrimSpace(pathValue) == "" {
		return nil, errors.New("path must not be empty")
	}

	segments := strings.Split(pathValue, ".")
	for _, segment := range segments {
		if strings.TrimSpace(segment) == "" {
			return nil, fmt.Errorf("path %q contains an empty segment", pathValue)
		}
	}

	return segments, nil
}

func setTree(node any, segments []string, value any) (any, error) {
	if len(segments) == 0 {
		return value, nil
	}

	mapping, ok := node.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("path segment %q: parent is not an object", segments[0])
	}

	key := segments[0]
	if len(segments) == 1 {
		mapping[key] = value

		return mapping, nil
	}

	child, exists := mapping[key]
	if !exists {
		child = make(map[string]any)
	}

	updated, err := setTree(child, segments[1:], value)
	if err != nil {
		return nil, err
	}

	mapping[key] = updated

	return mapping, nil
}

func mergeTree(node any, segments []string, value any) (any, error) {
	existing, err := getTree(node, segments)
	if err != nil {
		return nil, err
	}

	existingMap, ok := existing.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("path %q: target is not an object", strings.Join(segments, "."))
	}

	valueMap, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("path %q: merge value is not an object", strings.Join(segments, "."))
	}

	merged := mergeMaps(existingMap, valueMap)

	return setTree(node, segments, merged)
}

func mergeMaps(target, source map[string]any) map[string]any {
	for _, key := range slices.Sorted(maps.Keys(source)) {
		sourceValue := source[key]

		targetValue, exists := target[key]
		targetMap, targetIsMap := targetValue.(map[string]any)

		sourceMap, sourceIsMap := sourceValue.(map[string]any)
		if exists && targetIsMap && sourceIsMap {
			target[key] = mergeMaps(targetMap, sourceMap)

			continue
		}

		target[key] = sourceValue
	}

	return target
}

func appendTree(node any, segments []string, value any) (any, error) {
	existing, err := getTree(node, segments)
	if err != nil {
		return nil, err
	}

	list, ok := existing.([]any)
	if !ok {
		return nil, fmt.Errorf("path %q: target is not an array", strings.Join(segments, "."))
	}

	return setTree(node, segments, append(list, value))
}

func deleteTree(node any, segments []string) (any, error) {
	mapping, ok := node.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("path segment %q: parent is not an object", segments[0])
	}

	key := segments[0]
	if len(segments) == 1 {
		if _, exists := mapping[key]; !exists {
			return nil, fmt.Errorf("path %q: key does not exist", key)
		}

		delete(mapping, key)

		return mapping, nil
	}

	child, exists := mapping[key]
	if !exists {
		return nil, fmt.Errorf("path %q: key does not exist", key)
	}

	updated, err := deleteTree(child, segments[1:])
	if err != nil {
		return nil, err
	}

	mapping[key] = updated

	return mapping, nil
}

func getTree(node any, segments []string) (any, error) {
	current := node

	for _, segment := range segments {
		mapping, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("path segment %q: parent is not an object", segment)
		}

		value, exists := mapping[segment]
		if !exists {
			return nil, fmt.Errorf("path segment %q does not exist", segment)
		}

		current = value
	}

	return current, nil
}

func yamlSet(mapping *yaml.Node, segments []string, value any) error {
	if len(segments) == 0 {
		return errors.New("path must not be empty")
	}

	if mapping.Kind != yaml.MappingNode {
		return fmt.Errorf("path segment %q: parent is not a mapping", segments[0])
	}

	key := segments[0]
	index := yamlFind(mapping, key)

	if len(segments) == 1 {
		node, err := yamlValue(value)
		if err != nil {
			return err
		}

		if index >= 0 {
			mapping.Content[index+1] = node

			return nil
		}

		mapping.Content = append(mapping.Content, yamlKey(key), node)

		return nil
	}

	var child *yaml.Node
	if index >= 0 {
		child = mapping.Content[index+1]
		if child.Kind != yaml.MappingNode {
			child = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			mapping.Content[index+1] = child
		}
	} else {
		child = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		mapping.Content = append(mapping.Content, yamlKey(key), child)
	}

	return yamlSet(child, segments[1:], value)
}

func yamlMerge(mapping *yaml.Node, segments []string, value any) error {
	valueMap, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("path %q: merge value is not a mapping", strings.Join(segments, "."))
	}

	target := mapping
	for _, segment := range segments {
		index := yamlFind(target, segment)
		if index < 0 {
			return yamlSet(mapping, segments, value)
		}

		target = target.Content[index+1]
		if target.Kind != yaml.MappingNode {
			return fmt.Errorf("path %q: target is not a mapping", strings.Join(segments, "."))
		}
	}

	for _, key := range slices.Sorted(maps.Keys(valueMap)) {
		if err := yamlSet(target, []string{key}, valueMap[key]); err != nil {
			return err
		}
	}

	return nil
}

func yamlAppend(mapping *yaml.Node, segments []string, value any) error {
	target, err := yamlGet(mapping, segments)
	if err != nil {
		return err
	}

	if target.Kind != yaml.SequenceNode {
		return fmt.Errorf("path %q: target is not a sequence", strings.Join(segments, "."))
	}

	node, err := yamlValue(value)
	if err != nil {
		return err
	}

	target.Content = append(target.Content, node)

	return nil
}

func yamlDelete(mapping *yaml.Node, segments []string) error {
	if len(segments) == 0 {
		return errors.New("path must not be empty")
	}

	if mapping.Kind != yaml.MappingNode {
		return fmt.Errorf("path segment %q: parent is not a mapping", segments[0])
	}

	index := yamlFind(mapping, segments[0])
	if index < 0 {
		return fmt.Errorf("path %q: key does not exist", strings.Join(segments, "."))
	}

	if len(segments) == 1 {
		mapping.Content = append(mapping.Content[:index], mapping.Content[index+2:]...)

		return nil
	}

	return yamlDelete(mapping.Content[index+1], segments[1:])
}

func yamlGet(mapping *yaml.Node, segments []string) (*yaml.Node, error) {
	if len(segments) == 0 {
		return nil, errors.New("path must not be empty")
	}

	current := mapping

	for _, segment := range segments {
		if current.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("path segment %q: parent is not a mapping", segment)
		}

		index := yamlFind(current, segment)
		if index < 0 {
			return nil, fmt.Errorf("path segment %q does not exist", segment)
		}

		current = current.Content[index+1]
	}

	return current, nil
}

func yamlFind(mapping *yaml.Node, key string) int {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return index
		}
	}

	return -1
}

func yamlKey(key string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
}

func yamlValue(value any) (*yaml.Node, error) {
	var node yaml.Node
	if err := node.Encode(value); err != nil {
		return nil, fmt.Errorf("encode yaml value: %w", err)
	}

	return &node, nil
}
