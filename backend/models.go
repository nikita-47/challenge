package main

const (
	ModelSonnet  = "claude-sonnet-4-6"
	ModelHaiku   = "claude-haiku-4-5"
	ModelOpus    = "claude-opus-4-6"
	DefaultModel = ModelSonnet
)

// ModelPricing holds per-model token pricing (USD per 1M tokens).
type ModelPricing struct {
	CostIn       float64 // input tokens
	CostOut      float64 // output tokens
	CacheWriteIn float64 // prompt caching: write
	CacheReadIn  float64 // prompt caching: read
}

var modelPricing = map[string]ModelPricing{
	ModelSonnet: {CostIn: 3.0, CostOut: 15.0, CacheWriteIn: 3.75, CacheReadIn: 0.30},
	ModelHaiku:  {CostIn: 1.0, CostOut: 5.0, CacheWriteIn: 1.25, CacheReadIn: 0.10},
	ModelOpus:   {CostIn: 5.0, CostOut: 25.0, CacheWriteIn: 6.25, CacheReadIn: 0.50},
}

// PricingFor returns pricing for the given model ID.
// Falls back to DefaultModel pricing for unknown models.
func PricingFor(model string) ModelPricing {
	if p, ok := modelPricing[model]; ok {
		return p
	}
	return modelPricing[DefaultModel]
}
