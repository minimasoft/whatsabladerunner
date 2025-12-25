# Running whatsabladerunner with Docker

This project is ready to be run in a container. It uses a small Alpine-based image.

## Volume Requirement

The bot stores all its data, configuration, and databases in `/data`. You **must** mount a persistent volume to this path to keep your WhatsApp session and configuration.

## Basic Docker Run

```bash
docker build -t whatsabladerunner .

docker run -it \
  -v $(pwd)/blady_data:/data \
  --name blady \
  whatsabladerunner
```

Upon the first run, the `/data` directory will be populated with default configuration templates. Watch the terminal to scan the WhatsApp QR code.

## Docker Compose

The easiest way to run the bot is using `docker-compose`:

```bash
docker-compose up -d
```

Check logs for the QR code:

```bash
docker-compose logs -f
```
