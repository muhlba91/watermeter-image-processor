# Water Meter Image Processor

[![](https://img.shields.io/github/license/muhlba91/watermeter-image-processor?style=for-the-badge)](LICENSE.md)
[![](https://img.shields.io/github/actions/workflow/status/muhlba91/watermeter-image-processor/verify.yml?style=for-the-badge)](https://github.com/muhlba91/watermeter-image-processor/actions/workflows/verify.yml)
[![](https://img.shields.io/coverallsCoverage/github/muhlba91/watermeter-image-processor?style=for-the-badge)](https://github.com/muhlba91/watermeter-image-processor/)
[![](https://api.scorecard.dev/projects/github.com/muhlba91/watermeter-image-processor/badge?style=for-the-badge)](https://scorecard.dev/viewer/?uri=github.com/muhlba91/watermeter-image-processor)
[![](https://img.shields.io/github/release-date/muhlba91/watermeter-image-processor?style=for-the-badge)](https://github.com/muhlba91/watermeter-image-processor/releases)
[![](https://img.shields.io/github/all-contributors/muhlba91/watermeter-image-processor?color=ee8449&style=for-the-badge)](#contributors)
<a href="https://www.buymeacoffee.com/muhlba91" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-orange.png" alt="Buy Me A Coffee" height="28" width="150"></a>

Water Meter Image Processor is a Go-based service designed to process images from water meters (e.g., captured via ESP32-CAM), use an AI provider to read the meter value, and publish the results to MQTT for Home Assistant. It integrates seamlessly with Home Assistant via MQTT Discovery.

> [!NOTE]
> This project was co-developed with the help of AI coding assistants. All changes are reviewed and tested by the maintainer.

---

## Features

- **AI-Powered OCR**: Supports multiple AI providers — Google Gemini, OpenAI, Anthropic, Mistral, and any OpenAI-compatible proxy (e.g., LiteLLM, Ollama, vLLM) — to interpret water meter readings from images.
- **Image Preprocessing**: Smooths out JPEG compression artifacts and normalizes the brightness of dim frames, then stacks an enlarged crop of the digit wheels (located by the red decimal wheels' hue) below the full image, so the AI provider sees both the whole meter for context and the digits in detail. Answers with the wrong number of digits are rejected instead of published.
- **MQTT Integration**: Subscribes to an image topic and publishes the processed readings.
- **Home Assistant Discovery**: Automatically creates a sensor in Home Assistant for easy monitoring.
- **Image Storage**: Persists processed images to a local file path (default) or Scaleway Object Storage (S3 compatible).
- **Health Monitoring**: Includes a `healthz` server for liveness, readiness, and startup checks.

---

## Configuration

Configure the application using the following environment variables:

### General

| Variable                              | Description                                                   | Default                         |
| ------------------------------------- | ------------------------------------------------------------- | ------------------------------- |
| `METER_ID`                            | Unique identifier for the meter.                              | `water-meter`                   |
| `METER_NAME`                          | Display name for the meter.                                   | `Water Meter`                   |
| `METER_MODEL`                         | Model description of the meter.                               | `ESP32 Water Meter`             |
| `BROKER_ADDRESS`                      | MQTT broker address (e.g., `tcp://localhost:1883`).           | `tcp://localhost:1883`          |
| `BROKER_TOPIC_SUBSCRIPTION_TEMPLATE`  | Template for image subscription topic.                        | `tele/%s/image`                 |
| `BROKER_TOPIC_PUBLISH_TEMPLATE`       | Template for usage publication topic.                         | `stat/%s/water/usage/state`     |
| `BROKER_CLIENT_ID`                    | MQTT client ID.                                               | *(optional)*                    |
| `BROKER_USERNAME`                     | MQTT username.                                                | *(optional)*                    |
| `BROKER_PASSWORD`                     | MQTT password.                                                | *(optional)*                    |
| `HEALTHZ_HOST`                        | Host for the health server.                                   | `0.0.0.0`                       |
| `HEALTHZ_PORT`                        | Port for the health server.                                   | `8080`                          |

### Image Processing

| Variable                 | Description                                                                                                                                                                     | Default |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `IMAGE_ROTATION_DEGREES` | Counter-clockwise rotation, in degrees, applied to every image to level the digit wheels if the camera is not mounted straight. Tilted digit wheels are misread far more often. | `0`     |

### AI Provider

Use `MODEL_PROVIDER` to select the active provider. Only the variables for the chosen provider need to be set.

| Variable                    | Description                                                                                                                                                                    | Default  |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| `MODEL_PROVIDER`            | AI provider to use: `gemini`, `openai`, `anthropic`, `mistral`, or `openai_compat`.                                                                                            | `gemini` |
| `MODEL_CHECK_CACHE_TTL`     | How long a provider's model-availability check is cached before being re-verified, instead of re-listing models on every processed image.                                      | `5m`     |
| `MODEL_MAX_RESPONSE_TOKENS` | Maximum number of tokens the model may generate per image, including the hidden reasoning of reasoning models. Too low a limit can cut off the answer, which is then rejected. | `4096`   |

#### Google Gemini (`gemini`)

| Variable          | Description                  | Default                  |
| ----------------- | ---------------------------- | ------------------------ |
| `GEMINI_API_KEY`  | Google Gemini API key.       | *(required)*             |
| `GEMINI_MODEL`    | Gemini model to use.         | `gemini-3.5-flash-lite`  |

#### OpenAI (`openai`)

| Variable         | Description             | Default       |
| ---------------- | ----------------------- | ------------- |
| `OPENAI_API_KEY` | OpenAI API key.         | *(required)*  |
| `OPENAI_MODEL`   | OpenAI model to use.    | `gpt-6-luna`  |

#### Anthropic (`anthropic`)

| Variable             | Description                 | Default              |
| -------------------- | --------------------------- | -------------------- |
| `ANTHROPIC_API_KEY`  | Anthropic API key.          | *(required)*         |
| `ANTHROPIC_MODEL`    | Anthropic model to use.     | `claude-haiku-4-5`   |

#### Mistral (`mistral`)

Uses Mistral's OpenAI-compatible chat completions API. Only vision-capable chat models are supported (e.g. `mistral-small-latest`, `pixtral-large-latest`) — the dedicated Document AI/OCR models (`mistral-ocr-*`) use a different, instruction-less API and cannot be used here.

| Variable          | Description           | Default                 |
| ----------------- | --------------------- | ----------------------- |
| `MISTRAL_API_KEY` | Mistral API key.      | *(required)*            |
| `MISTRAL_MODEL`   | Mistral model to use. | `mistral-medium-latest` |

#### OpenAI-Compatible Proxy (`openai_compat`)

Use this provider to connect to any OpenAI API-compatible endpoint such as [LiteLLM](https://github.com/BerriAI/litellm), [Ollama](https://ollama.com/), or [vLLM](https://github.com/vllm-project/vllm).

| Variable                | Description                                          | Default                    |
| ----------------------- | ---------------------------------------------------- | -------------------------- |
| `OPENAI_COMPAT_URL`     | Base URL of the OpenAI-compatible endpoint.          | `http://localhost:4000`    |
| `OPENAI_COMPAT_API_KEY` | API key for the endpoint (if required).              | *(optional)*               |
| `OPENAI_COMPAT_MODEL`   | Model name to use via the compatible endpoint.       | `glm-flash-latest`         |

### Image Storage

Processed images can optionally be persisted for later review. Use `STORAGE_PROVIDER` to select where — exactly one of the two providers is ever active. Storage is a best-effort side effect: a write failure is logged and ignored, and never blocks or fails image processing.

| Variable           | Description                                    | Default |
| ------------------ | ---------------------------------------------- | ------- |
| `STORAGE_PROVIDER` | Storage provider to use: `file` or `scaleway`. | `file`  |

#### Local File (`file`)

| Variable            | Description                                                        | Default                      |
| ------------------- | ------------------------------------------------------------------ | ---------------------------- |
| `FILE_STORAGE_PATH` | Directory images are written to. `%s` is replaced with `METER_ID`. | `/tmp/watermeter-images/%s/` |

#### Scaleway (`scaleway`)

`SCW_REGION`, `SCW_ACCESS_KEY`, `SCW_SECRET_KEY`, and `SCW_BUCKET` must all be set, or startup fails.

| Variable          | Description                      | Default          |
| ----------------- | -------------------------------- | ---------------- |
| `SCW_REGION`      | Scaleway region for S3 backup.   | `fr-par`         |
| `SCW_ACCESS_KEY`  | Scaleway access key.             | *(required)*     |
| `SCW_SECRET_KEY`  | Scaleway secret key.             | *(required)*     |
| `SCW_BUCKET`      | Scaleway S3 bucket name.         | *(required)*     |
| `SCW_BUCKET_PATH` | Path template within the bucket. | `watermeter/%s/` |

---

## Deployment

### Docker Run

#### Google Gemini (default)

```shell
docker run -d \
  --name watermeter-image-processor \
  -e BROKER_ADDRESS="tcp://mqtt-broker:1883" \
  -e GEMINI_API_KEY="your-gemini-api-key" \
  -e METER_ID="my-water-meter" \
  ghcr.io/muhlba91/watermeter-image-processor:latest
```

#### OpenAI

```shell
docker run -d \
  --name watermeter-image-processor \
  -e BROKER_ADDRESS="tcp://mqtt-broker:1883" \
  -e MODEL_PROVIDER="openai" \
  -e OPENAI_API_KEY="your-openai-api-key" \
  -e METER_ID="my-water-meter" \
  ghcr.io/muhlba91/watermeter-image-processor:latest
```

#### Anthropic

```shell
docker run -d \
  --name watermeter-image-processor \
  -e BROKER_ADDRESS="tcp://mqtt-broker:1883" \
  -e MODEL_PROVIDER="anthropic" \
  -e ANTHROPIC_API_KEY="your-anthropic-api-key" \
  -e METER_ID="my-water-meter" \
  ghcr.io/muhlba91/watermeter-image-processor:latest
```

#### Mistral

```shell
docker run -d \
  --name watermeter-image-processor \
  -e BROKER_ADDRESS="tcp://mqtt-broker:1883" \
  -e MODEL_PROVIDER="mistral" \
  -e MISTRAL_API_KEY="your-mistral-api-key" \
  -e METER_ID="my-water-meter" \
  ghcr.io/muhlba91/watermeter-image-processor:latest
```

#### OpenAI-Compatible Proxy (e.g., Ollama)

```shell
docker run -d \
  --name watermeter-image-processor \
  -e BROKER_ADDRESS="tcp://mqtt-broker:1883" \
  -e MODEL_PROVIDER="openai_compat" \
  -e OPENAI_COMPAT_URL="http://ollama:11434/v1" \
  -e OPENAI_COMPAT_MODEL="llava" \
  -e METER_ID="my-water-meter" \
  ghcr.io/muhlba91/watermeter-image-processor:latest
```

#### Scaleway Image Storage

```shell
docker run -d \
  --name watermeter-image-processor \
  -e BROKER_ADDRESS="tcp://mqtt-broker:1883" \
  -e GEMINI_API_KEY="your-gemini-api-key" \
  -e STORAGE_PROVIDER="scaleway" \
  -e SCW_ACCESS_KEY="your-scaleway-access-key" \
  -e SCW_SECRET_KEY="your-scaleway-secret-key" \
  -e SCW_BUCKET="your-scaleway-bucket" \
  -e METER_ID="my-water-meter" \
  ghcr.io/muhlba91/watermeter-image-processor:latest
```

---

## Testing

The repository includes a `testing/` directory to help you verify the setup locally.

### 1. Start Local Infrastructure

Use Docker Compose to start a Mosquitto MQTT broker and MQTT Explorer.

```shell
cd testing
docker-compose up -d
```

- **MQTT Broker**: `localhost:1883`
- **MQTT Explorer**: `http://localhost:3000`

### 2. Ingest a Test Image

You can simulate a water meter sending an image using the provided Python script.

1. Install dependencies:

   ```shell
   pip install -r testing/requirements.txt
   ```

2. Run the ingestion script:

   ```shell
   python3 testing/ingest.py localhost tele/water-meter/image testing/watermeter.jpg
   ```

Replace `tele/water-meter/image` with the topic corresponding to your `METER_ID` (default is `water-meter`).

### 3. Measure Reading Accuracy

`cmd/eval` sends the test images in `testing/images/` through the same preprocessing as the processor to one or more AI providers, repeatedly, and reports how often each provider reads the expected value and how consistent its answers are. It reads the API keys, models, and `IMAGE_ROTATION_DEGREES` from the same environment variables as the processor. Use it to verify any change to the preprocessing, the prompt, or a model before deploying it.

```shell
make eval
make eval EVAL_ARGS="-providers gemini,mistral -images watermeter.jpg -repeats 10 -v"
```

| Flag         | Description                                                                             | Default                           |
| ------------ | --------------------------------------------------------------------------------------- | --------------------------------- |
| `-providers` | Comma-separated providers to evaluate.                                                  | `gemini,openai,anthropic,mistral` |
| `-images`    | Comma-separated substrings; only images whose file name contains one are evaluated.     | *(all)*                           |
| `-repeats`   | Calls per image and provider.                                                           | `5`                               |
| `-parallel`  | Concurrent calls per provider.                                                          | `3`                               |
| `-csv`       | Write the raw results to a CSV file.                                                    | *(none)*                          |
| `-dump`      | Write the preprocessed images, as sent to the providers, to a directory for inspection. | *(none)*                          |
| `-v`         | Log the raw provider responses.                                                         | `false`                           |

The test images, their accepted readings, and what each one tests are listed in `testing/images/manifest.json`:

| Image                | Purpose                                                                                         |
| -------------------- | ----------------------------------------------------------------------------------------------- |
| `../watermeter.jpg`  | Real worst case: level, very dark, heavily compressed ESP32-CAM frame. All providers must pass. |
| `level-lit-q90.jpg`  | Control: an ideal frame. A failure here is a regression.                                        |
| `level-lit-q10.jpg`  | Good lighting, heavy JPEG compression.                                                          |
| `level-dark-q30.jpg` | Under-exposed, moderate JPEG compression.                                                       |
| `level-dark-q10.jpg` | Stretch target, harder than the real worst case: under-exposed and heavily compressed.          |
| `tilted-lit-q30.jpg` | Camera ~26° off level; run with `IMAGE_ROTATION_DEGREES=-26` to verify the rotation setting.    |

All but `watermeter.jpg` are ESP32-CAM-like 800x600 frames generated from a high-quality photo (reading `667.38`), which is not part of the repository. `make eval-images` recreates exactly this set from it; the crops, rotation, and variants are defined in the `Makefile`:

```shell
# convert the DNG to PNG first, e.g. on macOS: sips -s format png photo.dng --out photo.png
make eval-images SRC=photo.png
```

To generate other images, call the generator directly: `-framing name=x,y,w,h[@degrees]` defines a crop of the source photo, optionally rotated counter-clockwise. Each `-variant framing:lighting:quality` writes `<framing>-<lighting>-q<quality>.jpg`, where `lit` keeps the photo's exposure and `dark` scales it to the `-dark-mean` luminance (default `28`, matching `watermeter.jpg`). Regenerating keeps the notes of existing manifest entries.

---

## Contributors

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/docs/en/emoji-key)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tbody>
    <tr>
      <td align="center" valign="top" width="14.28%"><a href="https://muehlbachler.io/"><img src="https://avatars.githubusercontent.com/u/653739?v=4?s=100" width="100px;" alt="Daniel Mühlbachler-Pietrzykowski"/><br /><sub><b>Daniel Mühlbachler-Pietrzykowski</b></sub></a><br /><a href="#maintenance-muhlba91" title="Maintenance">🚧</a> <a href="https://github.com/muhlba91/watermeter-image-processor/commits?author=muhlba91" title="Code">💻</a> <a href="https://github.com/muhlba91/watermeter-image-processor/commits?author=muhlba91" title="Documentation">📖</a></td>
    </tr>
  </tbody>
</table>

<!-- markdownlint-restore -->
<!-- prettier-ignore-end -->

<!-- ALL-CONTRIBUTORS-LIST:END -->

This project follows the [all-contributors](https://github.com/all-contributors/all-contributors) specification. Contributions of any kind welcome!
