package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/rs/zerolog/log"
)

const (
	pkg        = "github.com/digital-foxy/card-client/store/record/erecord/ent"
	src        = "./store/record/erecord/schema"
	dest       = "./store/record/erecord/ent"
	stringType = "string"
)

// Pattern to match type assertion and capture ANY custom type
var incorrectPattern = regexp.MustCompile(`if id, ok := _spec\.ID\.Value\.\(([\w.]+)\); ok \{`)

func main() {
	cfg := &gen.Config{
		Features: []gen.Feature{
			gen.FeatureUpsert,
			gen.FeatureModifier,
			gen.FeatureExecQuery,
		},
		Package: pkg,
		Target:  dest,
	}

	// Load the graph first to inspect schemas
	graph, err := entc.LoadGraph(src, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("error loading ent graph")
	}

	// Generate code using entc.Generate
	err = entc.Generate(src, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("error running ent codegen")
	}

	// Post-process generated *_create.go files to fix custom ID type conversions
	if err := fixCustomIDTypeConversions(dest, graph); err != nil {
		log.Fatal().Err(err).Msg("error fixing custom ID type conversion")
	}
}

func fixCustomIDTypeConversions(entDir string, graph *gen.Graph) error {
	// Build a map of entity names to their ID field info
	stringIDEntities := getStringIDEntities(graph)

	// Find all *_create.go files
	files, err := filepath.Glob(filepath.Join(entDir, "*_create.go"))
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := fixTypeIDConversion(file, stringIDEntities); err != nil {
			return err
		}
	}

	return nil
}

func getStringIDEntities(graph *gen.Graph) map[string]string {
	stringIDEntities := make(map[string]string)

	for _, node := range graph.Nodes {
		fieldInfoType := node.ID.Type
		fieldType := fieldInfoType.Type.String()
		rFieldType := fieldInfoType.RType.String()

		// Check if ID is string-based with a custom GoType
		if fieldType == stringType && rFieldType != stringType {
			stringIDEntities[strings.ToLower(node.Name)] = rFieldType
		}
	}

	return stringIDEntities
}

func fixTypeIDConversion(file string, stringIDEntities map[string]string) error {
	// Extract entity name from filename (e.g., "creatorentity_create.go" -> "creatorentity")
	entityName := strings.TrimSuffix(filepath.Base(file), "_create.go")

	// Check if this entity has a string ID with custom GoType
	customType, ok := stringIDEntities[entityName]
	if !ok {
		return nil
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	originalContent := string(content)
	modifiedContent := replaceIncorrectPatterns(originalContent, customType)

	// Only write if content changed
	if modifiedContent != originalContent {
		if err := os.WriteFile(file, []byte(modifiedContent), 0644); err != nil {
			return fmt.Errorf("writing file %s: %w", file, err)
		}
		log.Info().Msgf("Fixed custom ID type conversion in %s", filepath.Base(file))
	}

	return nil
}

func replaceIncorrectPatterns(originalContent string, customType string) string {
	modifiedContent := originalContent
	newStr := fmt.Sprintf("if id, ok := _spec.ID.Value.(string); ok {\n\t\t\t_node.ID = %s(id)", customType)

	// Find all matches and replace them
	matches := incorrectPattern.FindAllStringSubmatch(originalContent, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		matchedType := match[1]

		// Skip if it's already string
		if matchedType == stringType {
			continue
		}

		// Build the old and new strings
		oldStr := fmt.Sprintf("if id, ok := _spec.ID.Value.(%s); ok {\n\t\t\t_node.ID = id", matchedType)

		modifiedContent = strings.Replace(modifiedContent, oldStr, newStr, 1)
	}

	return modifiedContent
}
