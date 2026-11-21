FROM python:3.11-slim
COPY --from=ghcr.io/astral-sh/uv:0.13.9 /uv /usr/local/bin/uv
COPY --from=ghcr.io/astral-sh/uv:0.13.9 /uvx /usr/local/bin/uvx
RUN apt-get update && apt-get install -y --no-install-recommends \
    openssh-client \
    git \
    && rm -rf /var/lib/apt/lists/* \
    && uv pip install --system passlib
WORKDIR /workspace
