# Go + DevContainer + Ollama (GPT-OSS 20B)


This repo lets you build a Go backend that talks to an Ollama model (default: GPT-OSS 20B) inside Docker. Designed for Windows 11 (ARM64 or x86_64) + VS Code Dev Containers.


## Prereqs
- Docker Desktop (enable WSL2 backend on Windows)
- VS Code + Dev Containers extension


## Setup
1. **Clone** this repo and open it in VS Code.
2. Copy `.env.example` to `.env` and set `OLLAMA_MODELS_HOST` to a host directory for models (e.g. `G:\\models` on Windows).
3. Press `F1` → **Dev Containers: Reopen in Container**. This will bring up `ollama` & `app` services.
4. The devcontainer will try to **pull the model** specified by `OLLAMA_MODEL` on first start. You can also pull manually:
```sh
# in another terminal
docker exec -it ollama ollama pull gpt-oss-20b-q4_K_M