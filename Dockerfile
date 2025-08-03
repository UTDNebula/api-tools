FROM golang:1.23 AS builder

WORKDIR /app
COPY . .

# Run setup and checks
RUN make setup
RUN make check
RUN make build

# Use a lightweight final image
FROM debian:12-slim
WORKDIR /app

# Install gcloud CLI
RUN apt-get update && apt-get install -y wget gnupg apt-transport-https lsb-release ca-certificates
RUN wget -qO - https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor > /usr/share/keyrings/cloud.google.gpg
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
RUN apt-get update && apt-get install -y google-cloud-sdk

# Install chromium
RUN apt-get update && apt-get install -y chromium
ENV CHROMIUM_BIN /usr/bin/chromium
ENV GOOGLE_CHROME_BIN /usr/bin/chromium # Also set this for compatibility

# Copy build file from builder
COPY --from=builder /app/api-tools /app/api-tools
COPY runners /app/runners

RUN chmod +x /app/runners/setup.sh
ENTRYPOINT ["/app/runners/setup.sh"]
