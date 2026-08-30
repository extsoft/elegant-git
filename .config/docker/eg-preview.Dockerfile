FROM debian:bookworm-slim
RUN apt-get update \
  && apt-get install -y --no-install-recommends git ca-certificates \
  && rm -rf /var/lib/apt/lists/* \
  && useradd -m -d /home/dev -s /bin/bash dev
USER dev
ENV HOME=/home/dev
WORKDIR /home/dev
CMD ["bash"]
