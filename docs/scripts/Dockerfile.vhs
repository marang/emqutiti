FROM golang:1.24-bookworm

# Install dependencies, including Chromium and its sandbox helper.
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl git fonts-dejavu-core ffmpeg chromium chromium-sandbox && \
    curl -L https://github.com/tsl0922/ttyd/releases/latest/download/ttyd.x86_64 \
        -o /usr/local/bin/ttyd && \
    chmod +x /usr/local/bin/ttyd && \
    # Ensure the Chromium sandbox helper has the correct permissions when present.
    (chown root:root /usr/lib/chromium/chrome-sandbox 2>/dev/null || true) && \
    (chmod 4755 /usr/lib/chromium/chrome-sandbox 2>/dev/null || true) && \
    rm -rf /var/lib/apt/lists/*

RUN go install github.com/charmbracelet/vhs@latest

ENV PATH="/go/bin:$PATH"

# Use a wrapper that forces flags needed for running in containers.
RUN printf '%s\n' \
    '#!/usr/bin/env bash' \
    'exec /usr/bin/chromium \\' \
    '  --headless=new \\' \
    '  --no-sandbox \\' \
    '  --disable-setuid-sandbox \\' \
    '  --disable-gpu \\' \
    '  --disable-gpu-sandbox \\' \
    '  --disable-dev-shm-usage \\' \
    '  --no-zygote \\' \
    '  --single-process \\' \
    '  "$@"' \
    > /usr/local/bin/chromium-no-sandbox && \
    chmod +x /usr/local/bin/chromium-no-sandbox

# Point Rod (used by VHS) at our wrapper to avoid sandbox issues.
ENV ROD_BROWSER_PATH=/usr/local/bin/chromium-no-sandbox

WORKDIR /work
ENTRYPOINT ["bash"]
