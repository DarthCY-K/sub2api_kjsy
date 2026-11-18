//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGPT62OfficialCatalogWorkflowAndInstructions(t *testing.T) {
	for _, model := range []string{"gpt-6.2-sol", "gpt-6.2-astra"} {
		t.Run(model, func(t *testing.T) {
			descriptor := newConfiguredCodexModelDescriptor(model)
			var official map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(descriptor.officialMetadata, &official))
			require.JSONEq(t, `["text","image"]`, string(official["input_modalities"]))
			instructions := gjson.GetBytes(official["model_messages"], "instructions_template").String()
			require.NotEmpty(t, instructions)
			require.Equal(t, instructions, openai.CodexBaseInstructionsForModel(model))
			require.NotEqual(t, openai.CodexBaseInstructionsForModel("gpt-6-astra"), instructions)
			require.Equal(t, model, normalizeKnownOpenAICodexModel("openai/"+model+"-ultra"))
			require.Equal(t, "ultra", normalizeOpenAIReasoningEffortForModel("ULTRA", model))
			require.Equal(t, "max", normalizeOpenAIReasoningEffortForModel("max", model))
			require.Empty(t, normalizeOpenAIReasoningEffortForModel("ultra", "gpt-5.4"))
			req := &apicompat.AnthropicRequest{Model: model + "-ultra"}
			applyOpenAICompatModelNormalization(req)
			require.Equal(t, model, req.Model)
			require.Equal(t, "ultra", req.OutputConfig.Effort)
			converted, err := apicompat.AnthropicToResponses(req)
			require.NoError(t, err)
			require.Equal(t, "ultra", converted.Reasoning.Effort)
			levels := configuredCodexGPTReasoningLevels(model)
			require.Equal(t, "ultra", levels[len(levels)-1].Effort)
		})
	}
}

func TestGPT62OfficialAPIAddsUltrafastToFastOnlySnapshot(t *testing.T) {
	for _, model := range []string{"gpt-6.2-sol", "gpt-6.2-astra"} {
		a := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
		a.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{model: {ID: model, CodexToolCapabilities: map[string]json.RawMessage{"service_tiers": json.RawMessage(`[{"id":"priority","description":"keep upstream fields"}]`)}}}})
		tiers := accountCodexToolCapabilities(a, model)["service_tiers"]
		require.Equal(t, "keep upstream fields", gjson.GetBytes(tiers, `#(id=="priority").description`).String())
		require.True(t, gjson.GetBytes(tiers, `#(id=="ultrafast")`).Exists())
	}
}

func TestGPT62ChatOnlyAccountsRejectToolsAcrossProtocols(t *testing.T) {
	for _, model := range []string{"gpt-6.2-sol", "gpt-6.2-astra"} {
		for _, protocol := range []string{"chat", "responses", "messages"} {
			t.Run(model+"/"+protocol, func(t *testing.T) {
				body := `{"model":"public","messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"test","parameters":{"type":"object"}}}]}`
				if protocol == "responses" {
					body = `{"model":"public","input":"hi","tools":[{"type":"function","name":"test","parameters":{"type":"object"}}]}`
				}
				if protocol == "messages" {
					body = `{"model":"public","max_tokens":100,"messages":[{"role":"user","content":"hi"}],"tools":[{"name":"test","input_schema":{"type":"object"}}]}`
				}
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest("POST", "/v1/"+protocol, strings.NewReader(body))
				account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"model_mapping": map[string]any{"public": model}}}
				svc := &OpenAIGatewayService{cfg: &config.Config{}}
				var err error
				switch protocol {
				case "chat":
					_, err = svc.forwardAsRawChatCompletions(context.Background(), c, account, []byte(body), "")
				case "responses":
					_, err = svc.forwardResponsesViaRawChatCompletions(context.Background(), c, account, []byte(body), time.Now())
				case "messages":
					_, err = svc.forwardAnthropicViaRawChatCompletions(context.Background(), c, account, []byte(body), "")
				}
				require.ErrorContains(t, err, "Responses")
				require.Equal(t, 400, recorder.Code)
			})
		}
	}
}

func TestGPT62APIKeyManifestUsesFullResponses(t *testing.T) {
	for _, model := range []string{"gpt-6.2-sol", "gpt-6.2-astra"} {
		a := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
		body, err := completeAPIKeyCodexModelsManifestMetadata([]byte(`{"models":[{"slug":"`+model+`"}]}`), true, a)
		require.NoError(t, err)
		require.False(t, gjson.GetBytes(body, "models.0.use_responses_lite").Bool())
		require.Equal(t, int64(872000), gjson.GetBytes(body, "models.0.max_context_window").Int())
	}
}

func TestGPT62OfficialTwelveHourCacheWritePrices(t *testing.T) {
	for _, model := range []string{"gpt-6.2-sol", "gpt-6.2-astra"} {
		unit := 5e-6 // Official short-context 12-hour cache write, USD/token.
		if model == "gpt-6.2-astra" {
			unit = 20e-6
		}
		for _, svc := range []*BillingService{newTestBillingService(), NewBillingService(nil, newStubPricingServiceFromJSON(t, `{"other":{"input_cost_per_token":1}}`))} {
			for tier, ratio := range map[string]float64{"": 1, "priority": 2, "ultrafast": 3} {
				cost, err := svc.CalculateCostUnified(CostInput{
					Model: model, Tokens: UsageTokens{CacheCreationTokens: 100, CacheCreation12hTokens: 100},
					RateMultiplier: 1, ServiceTier: tier, PricingAt: time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC),
				})
				require.NoError(t, err)
				require.InDelta(t, 100*unit*ratio, cost.CacheCreationCost, 1e-12, model+"/"+tier)
			}
		}
	}
}
