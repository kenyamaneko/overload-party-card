package domain_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type openAPIEnumSchema struct {
	Enum []string `yaml:"enum"`
}

type openAPIDocument struct {
	Components struct {
		Schemas map[string]openAPIEnumSchema `yaml:"schemas"`
	} `yaml:"components"`
}

func readOpenAPIEnumValues(t *testing.T, schemaName string) []string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "data", "openapi.yaml"))
	require.NoError(t, err)

	var doc openAPIDocument
	require.NoError(t, yaml.Unmarshal(raw, &doc))

	require.Contains(t, doc.Components.Schemas, schemaName)

	return doc.Components.Schemas[schemaName].Enum
}

func TestEnumDrift(t *testing.T) {
	t.Run("[契約整合性] domain の列挙型定数と openapi.yaml の列挙型スキーマ", func(t *testing.T) {
		cases := []struct {
			caseName     string
			schemaName   string
			domainValues []string
		}{
			{
				caseName:   "PassiveEffectType 系定数のとき、openapi.yaml の PassiveEffectType の列挙値と過不足なく一致する",
				schemaName: "PassiveEffectType",
				domainValues: []string{
					domain.PassiveTPPerBackendDB,
					domain.PassiveTPPerBackendData,
					domain.PassiveTPIfCardTypeOnField,
					domain.PassiveYieldPerOtherDB,
					domain.PassiveYieldIfCardOnField,
					domain.PassiveAVBonus,
					domain.PassiveScaleCostFree,
				},
			},
			{
				caseName:   "PlatformEffectType 系定数のとき、openapi.yaml の PlatformEffectType の列挙値と過不足なく一致する",
				schemaName: "PlatformEffectType",
				domainValues: []string{
					domain.PlatformTPBonus,
					domain.PlatformYieldBonus,
					domain.PlatformAVBonus,
				},
			},
			{
				caseName:   "AttachmentEffectType 系定数のとき、openapi.yaml の AttachmentEffectType の列挙値と過不足なく一致する",
				schemaName: "AttachmentEffectType",
				domainValues: []string{
					domain.AttachmentStatBonus,
				},
			},
			{
				caseName:   "InitiativeKind 系定数のとき、openapi.yaml の InitiativeKind の列挙値と過不足なく一致する",
				schemaName: "InitiativeKind",
				domainValues: []string{
					domain.InitiativeKindRoutine,
					domain.InitiativeKindSpecial,
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.caseName, func(t *testing.T) {
				openAPIValues := readOpenAPIEnumValues(t, tc.schemaName)

				assert.ElementsMatch(t, tc.domainValues, openAPIValues)
			})
		}
	})
}
