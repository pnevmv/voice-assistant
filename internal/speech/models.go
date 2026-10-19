package speech

import "fmt"

// deprecatedSTTModels are refused at startup so a stale .env cannot quietly
// run on a model ElevenLabs has deprecated.
var deprecatedSTTModels = map[string]string{
	"scribe_v1": "deprecated by ElevenLabs; use scribe_v3 (batch) or scribe_v3_live (realtime)",
}

// ValidateSTTModel rejects an empty or deprecated STT model id.
func ValidateSTTModel(model string) error {
	if model == "" {
		return fmt.Errorf("ELEVENLABS_STT_MODEL is empty")
	}
	if why, bad := deprecatedSTTModels[model]; bad {
		return fmt.Errorf("ELEVENLABS_STT_MODEL=%s: %s", model, why)
	}
	return nil
}
