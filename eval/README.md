# Eval harness (Python)

Placeholder for M4. Planned layout:

```
eval/
  pyproject.toml          # uv-managed; ragas, httpx, jiwer, psycopg, pydantic
  unipd_eval/             # package: loader/validator, runners, metrics, reports
  golden/                 # golden set (see ../docs/eval/GOLDEN_SET_FORMAT.md)
  audio/                  # recordings (git-ignored if large)
  reports/                # generated, git-ignored
```

Voice eval (M5): each recording is transcribed with and without `keyterms` (config/keyterms.txt) to measure whether the extra-billed keyterms reduce WER on ISEE, CFU, Uniweb, bando…; time-to-first-audio is compared for `eleven_v4_turbo` and `eleven_flash_v2_5` via the `tts_model` override (run the voice service with `VOICE_ALLOW_OVERRIDES=true`; off by default). Synthetic (TTS-generated) audio flatters WER, so report it separately from real recordings.

Principles: black-box over HTTP (`RAG_URL`, `VOICE_URL`), judge LLM through the gateway (`GATEWAY_URL` as OpenAI `base_url`), deterministic metrics (date exact match, year correctness, refusal) are the hard gates; RAGAS scores are advisory. See ADR-0006.
