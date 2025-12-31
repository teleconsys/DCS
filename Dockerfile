FROM alpine:latest

# Install dependencies
RUN apk --no-cache add ca-certificates libc6-compat

# Create a non-root user
RUN addgroup -g 1000 dcs && \
    adduser -D -u 1000 -G dcs dcs

# Create working directory
WORKDIR /app

# Copy the executable
COPY bin/linux/DCS-API /app/DCS-API

# Verify that the file exists and set permissions
RUN ls -la /app/DCS-API && \
    chmod +x /app/DCS-API && \
    chown -R dcs:dcs /app

# Pass to the non-root user
USER dcs

# Define port as build argument with default value
ARG PORT=8080

# Set port as environment variable and expose it
ENV PORT=${PORT}
EXPOSE ${PORT}

# Note: Sensitive environment variables (e.g. GC_PRIVATE_KEY, GC_ADDRESS, GC_WALLET_GAS_ID, etc.)
# must be passed at runtime using --env-file or -e flags.

# Entrypoint to execute the application
ENTRYPOINT ["/app/DCS-API"]

