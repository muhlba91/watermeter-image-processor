# Water Meter Image Processor

[![](https://img.shields.io/github/license/muhlba91/watermeter-image-processor?style=for-the-badge)](LICENSE.md)
[![](https://img.shields.io/github/actions/workflow/status/muhlba91/watermeter-image-processor/verify.yml?style=for-the-badge)](https://github.com/muhlba91/watermeter-image-processor/actions/workflows/verify.yml)
[![](https://img.shields.io/coverallsCoverage/github/muhlba91/watermeter-image-processor?style=for-the-badge)](https://github.com/muhlba91/watermeter-image-processor/)
[![](https://api.scorecard.dev/projects/github.com/muhlba91/watermeter-image-processor/badge?style=for-the-badge)](https://scorecard.dev/viewer/?uri=github.com/muhlba91/watermeter-image-processor)
[![](https://img.shields.io/github/release-date/muhlba91/watermeter-image-processor?style=for-the-badge)](https://github.com/muhlba91/watermeter-image-processor/releases)
[![](https://img.shields.io/github/all-contributors/muhlba91/watermeter-image-processor?color=ee8449&style=for-the-badge)](#contributors)
<a href="https://www.buymeacoffee.com/muhlba91" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-orange.png" alt="Buy Me A Coffee" height="28" width="150"></a>

Water Meter Image Processor is a Go-based service designed to process images from water meters (e.g., captured via ESP32-CAM), use Google Gemini AI to read the meter value, and publish the results to MQTT for Home Assistant. It integrates seamlessly with Home Assistant via MQTT Discovery.

---

## Features

- **AI-Powered OCR**: Uses Google Gemini (e.g., `gemini-1.5-flash`) to interpret water meter readings from images.
- **MQTT Integration**: Subscribes to an image topic and publishes the processed readings.
- **Home Assistant Discovery**: Automatically creates a sensor in Home Assistant for easy monitoring.
- **Cloud Storage Backup**: Optionally uploads processed images to Scaleway Object Storage (S3 compatible).
- **Health Monitoring**: Includes a `healthz` server for liveness, readiness, and startup checks.

---

## Configuration

Configure the application using the following environment variables:

| Variable                              | Description                                         | Default                               |
| ------------------------------------- | --------------------------------------------------- | ------------------------------------- |
| `METER_ID`                            | Unique identifier for the meter.                    | `water-meter`                         |
| `METER_NAME`                          | Display name for the meter.                         | `Water Meter`                         |
| `METER_MODEL`                         | Model description of the meter.                     | `ESP32 Water Meter`                   |
| `BROKER_ADDRESS`                      | MQTT broker address (e.g., `tcp://localhost:1883`). | `tcp://localhost:1883`                |
| `BROKER_TOPIC_SUBSCRIPTION_TEMPLATE`  | Template for image subscription topic.              | `tele/%s/image`                       |
| `BROKER_TOPIC_PUBLISH_TEMPLATE`       | Template for usage publication topic.               | `stat/%s/water/usage/state`           |
| `BROKER_CLIENT_ID`                    | MQTT client ID.                                     | *(optional)*                          |
| `BROKER_USERNAME`                     | MQTT username.                                      | *(optional)*                          |
| `BROKER_PASSWORD`                     | MQTT password.                                      | *(optional)*                          |
| `GEMINI_API_KEY`                      | Google Gemini API key.                              | *(required)*                          |
| `GEMINI_MODEL`                        | Gemini model to use.                                | `gemini-3.1-flash-lite-preview`       |
| `SCW_REGION`                          | Scaleway region for S3 backup.                      | `fr-par`                              |
| `SCW_ACCESS_KEY`                      | Scaleway access key.                                | *(optional)*                          |
| `SCW_SECRET_KEY`                      | Scaleway secret key.                                | *(optional)*                          |
| `SCW_BUCKET`                          | Scaleway S3 bucket name.                            | *(optional)*                          |
| `SCW_BUCKET_PATH`                     | Path template within the bucket.                    | `watermeter/%s/`                      |
| `HEALTHZ_HOST`                        | Host for the health server.                         | `0.0.0.0`                             |
| `HEALTHZ_PORT`                        | Port for the health server.                         | `8080`                                |

---

## Deployment

### Docker Run

To run the processor using Docker, you need a Google Gemini API key and an MQTT broker.

```shell
docker run -d \
  --name watermeter-image-processor \
  -e BROKER_ADDRESS="tcp://mqtt-broker:1883" \
  -e GEMINI_API_KEY="your-gemini-api-key" \
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
