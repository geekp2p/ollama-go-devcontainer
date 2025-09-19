# Go + Dev Container + Ollama (GPT-OSS 20B)

This template wires a small Go HTTP service to an [Ollama](https://ollama.com) model (default: `gpt-oss:20b`) using Docker.
It is designed primarily for Windows 11 users running VS Code Dev Containers, but you can also run the Go binary directly on your
machine as long as an Ollama container is available.

---

## Project layout

- `cmd/server` – Go HTTP server that exposes `POST /chat` and `GET /healthz`.
- `internal/ollama` – tiny client for Ollama’s `/api/chat` endpoint.
- `docker-compose.yml` – brings up two services: `ollama` (the model runtime) and `app` (the VS Code dev container).
- `Makefile` – quality of life commands (`make run`, `make test`).

---

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) with the WSL2 backend enabled on Windows.
- [VS Code](https://code.visualstudio.com/) + the **Dev Containers** extension (recommended).
- [Go 1.22+](https://go.dev/dl/) only if you intend to run the API directly on the host instead of inside the dev container.

---

## Quick start

### 1. Clone the repository

```powershell
git clone https://github.com/geekp2p/ollama-go-devcontainer.git
cd ollama-go-devcontainer
```

Create a folder that will be mounted into the Ollama container for model storage (you can skip this if you plan to
set `OLLAMA_MODELS_HOST` to another location in a `.env` file):

```powershell
mkdir models
```

### 2. Start Ollama and download a model

Bring up the Ollama runtime (we only need the `ollama` service at this stage):

```powershell
docker compose up -d ollama
```

Download the default model (change the model name if you need another one):

```powershell
docker exec -it ollama ollama pull gpt-oss:20b
```

> Want Thai-centric models? Pull the new [OpenThaiGPT 1.5 Instruct](https://openthaigpt.aieat.or.th) variants before calling the API:
>
> ```powershell
> docker exec -it ollama ollama pull openthaigpt1.5-7b-instruct
> docker exec -it ollama ollama pull openthaigpt1.5-14b-instruct
> ```

> Exploring multimodal models? Pull the new vision-capable releases to experiment with images alongside text:
>
> ```powershell
> docker exec -it ollama ollama pull llama3.2-vision:11b
> docker exec -it ollama ollama pull internvl2
> docker exec -it ollama ollama pull internvl2.5
> ```

> Models are cached under the path configured by `OLLAMA_MODELS_HOST` (defaults to `./models`).

### 3. Run the Go API

You have two options depending on where you want to run the Go code.

#### Option A – Inside the VS Code dev container

1. Open the folder in VS Code.
2. Press `F1` → **Dev Containers: Reopen in Container**. The `app` service from `docker-compose.yml` is used as the development container.
3. Inside the container terminal, make sure dependencies are tidy (only required the first time):
   ```sh
   go mod tidy
   ```
4. Start the API:
   ```sh
   make run    # or: go run ./cmd/server
   ```

The server listens on `http://localhost:8082` and talks to the Ollama service via the internal hostname `http://ollama:11434`.

#### Option B – Run the binary on the host

@@ -106,51 +106,51 @@ curl -X POST http://localhost:8082/chat \
@@ -91,49 +98,50 @@ The API responds with JSON in the form `{"reply":"..."}`.

When you are done, stop the containers. Adding `-v` removes the model cache as well.

```powershell
docker compose down -v
```

---

## Configuration reference

| Variable | Default | Where it lives | Description |
|----------|---------|----------------|-------------|
| `OLLAMA_URL` | `http://ollama:11434` | `cmd/server/main.go` | Endpoint used by the Go service to talk to Ollama. Override with `http://localhost:11434` if you run the server on the host. |
| `OLLAMA_MODEL` | `gpt-oss:20b` | `docker-compose.yml`, Go server | Model pulled on first start and used for chat requests. |
| `OLLAMA_ALLOWED_MODELS` | _(unset)_ | Go server | Optional comma-separated allow list for `/chat`. The first entry becomes the default when `OLLAMA_MODEL` is not part of the list. |
| `OLLAMA_MODELS_HOST` | `./models` | `docker-compose.yml` | Host path mounted into the Ollama container to store downloaded models. |
| `OLLAMA_TIMEOUT` | `2m` | Go server | Maximum time the server waits for Ollama to answer. Accepts [Go duration strings](https://pkg.go.dev/time#ParseDuration). |

You can add a `.env` file next to `docker-compose.yml` to override any of these variables. Start with the provided
`.env.example`:

```sh
cp .env.example .env
```

List multiple models in `OLLAMA_ALLOWED_MODELS` to restrict `/chat` to known images. The first item in the list becomes the
default when requests omit the `model` field, and other values will be rejected with `400 Bad Request`.

---

## API contract

- `GET /healthz` – returns `200 OK` with body `ok`. Useful for probes.
- `POST /chat` – request body must be JSON with a `prompt` field (non-empty string).␊
  - Optional `model` field lets you override the default per call (e.g. `openthaigpt1.5-7b-instruct`, `llama3.2-vision:11b`, `internvl2.5`). When `OLLAMA_ALLOWED_MODELS` is defined, the value must match one of the configured options.
  - Optional `images` array accepts Base64-encoded image strings for multimodal models such as Llama 3.2 Vision and InternVL.
  - Example request:

    ```json
    {
      "prompt": "ช่วยอธิบายภาพนี้หน่อย",
      "model": "llama3.2-vision:11b",
      "images": ["<base64-encoded-image>"]
    }
    ```
  - Text-only example: `{ "prompt": "สรุป Expected Value ในการลงทุนหน่อย", "model": "openthaigpt1.5-14b-instruct" }`
  - Example response: `{ "reply": "..." }`

The server enforces a two-minute timeout per request. Failed upstream calls return `502 Bad Gateway` with the Ollama error message.

---

## Development workflow

- Run tests: `go test ./...`
- Rebuild binaries: `go build -o bin/server ./cmd/server`
- Format / lint: hook up your favourite tools (see `Makefile` stubs).

---

## Troubleshooting

- `context deadline exceeded` – the Go service could not reach Ollama before the timeout elapsed. Ensure the `ollama` container is running,
  the `OLLAMA_URL` matches your setup, and increase `OLLAMA_TIMEOUT` if the model needs longer than two minutes to answer.
- `model not found` – pull the model first (`docker exec -it ollama ollama pull <model>`), or update `OLLAMA_MODEL` to one that exists locally.
- Permission errors when pulling models on Windows – check that the folder bound via `OLLAMA_MODELS_HOST` is writable from WSL/Docker.

Enjoy hacking! :rocket: