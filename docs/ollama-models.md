# Choosing a local Ollama model for `--polish`

The `--polish` pass sends the rendered prompt to a local Ollama model and asks
it to tighten the wording. It is a **rewrite / instruction-following** task with
light reasoning — not heavy reasoning or code generation. Two practical
consequences:

- **Bigger is barely better here.** A 7B–14B model already captures almost all
  of the achievable quality for this task. The larger tiers below matter mostly
  if you also use the machine for other, heavier LLM work.
- **Smaller is faster and safer on the timeout.** A snappy model comfortably
  finishes within the default `--ollama-timeout 30s`; a model that pushes your
  RAM will be slower and more likely to time out (in which case the unpolished
  prompt is printed and a warning goes to stderr).

## How much memory you actually have

Apple Silicon uses *unified* memory shared between the OS, apps, and the model.
A useful rule of thumb:

    model budget  ≈  total RAM  −  ~5 GB  (OS + apps + the model's KV cache)

Ollama defaults to the `Q4_K_M` quantization. Approximate footprints at Q4:

| Model size | ~VRAM at Q4 |
|------------|-------------|
| 3B         | ~2.5 GB     |
| 7–8B       | ~5 GB       |
| 14B        | ~9 GB       |
| 27–32B     | ~16–19 GB   |
| 30B MoE (a3b) | ~18 GB   |
| 70–72B     | ~42 GB      |

## Recommendations by machine (Apple Silicon, M3 onward)

Each tier lists **two** picks: a **Default** (the best choice for the polish
task at that tier) and a **Push** (uses spare headroom for a little more
quality, or doubles as a general-purpose local model).

| Unified RAM | Typical machine (M3+) | Default pick | Push pick |
|-------------|-----------------------|--------------|-----------|
| **16 GB** | Mac mini (base) | `qwen2.5:7b` | `qwen2.5:14b` *(tight — close other apps)* |
| **24 GB** | Mac mini / MacBook Pro M3 | `qwen2.5:14b` | `qwen3:30b-a3b` *(MoE, fast)* |
| **32–36 GB** | MacBook Pro M3 Pro | `qwen2.5:14b` | `qwen2.5:32b` |
| **48 GB** | Mac mini M3 Pro / MBP | `qwen2.5:32b` | `gemma2:27b` |
| **64 GB** | MacBook Pro M3 Max | `qwen2.5:32b` | `qwen2.5:72b` |
| **96–128 GB** | MBP M3 Max / Mac Studio | `qwen2.5:72b` | `llama3.1:70b` at a higher quant (`:70b-instruct-q8_0`) |

Notes:

- **Base 16 GB Mac mini:** `qwen2.5:7b` is the sweet spot — ~5 GB, fast,
  strong instruction-following. `llama3.1:8b` is an equally fine runner-up and
  matches the `llama3` example in the README. `llama3.2:3b` is a good
  budget/fast option if you run polish interactively a lot.
- The `qwen2.5` family is recommended throughout for its instruction-following
  and writing quality; substitute the `qwen3` or `llama3.x` equivalents if you
  prefer those lines.
- 30B+ MoE and 70B+ dense models **will not load on 16 GB** — don't pick them
  for the base machine.

## Using your chosen model

```bash
ollama pull qwen2.5:7b

./prompt-creator generate --goal "..." --polish --polish-model qwen2.5:7b
```

To pin a specific quantization, pull the tagged variant and pass that tag:

```bash
ollama pull qwen2.5:14b-instruct-q5_K_M
./prompt-creator generate --goal "..." --polish --polish-model qwen2.5:14b-instruct-q5_K_M
```

## Caveats

These picks are a reasoned starting point, not benchmarked on each machine, and
the local-model landscape moves quickly. Treat the table as a default to adjust
once you've tried a model on your own hardware and prompts.
