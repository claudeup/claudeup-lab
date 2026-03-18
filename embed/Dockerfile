# ABOUTME: Base dev container image for Claude Code sandbox environments.
# ABOUTME: Provides Node.js, dev tools, Claude Code, claudeup, and Bun.

FROM node:22

ARG TZ=America/Denver
ENV TZ="$TZ"

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl wget ca-certificates gnupg2 \
    git git-lfs gh \
    procps htop sudo less man-db \
    jq yq \
    ripgrep fd-find fzf \
    build-essential make \
    python3 \
    nano vim neovim unzip \
    zsh \
    iptables ipset iproute2 dnsutils aggregate \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN update-alternatives --install /usr/bin/python python /usr/bin/python3 1

ARG USERNAME=node

RUN mkdir /commandhistory && \
    touch /commandhistory/.bash_history && \
    chown -R $USERNAME /commandhistory && \
    echo 'export PATH=$PATH:/home/node/.npm-global/bin:/home/node/.local/bin:/home/node/.bun/bin' >> /home/node/.bashrc && \
    echo 'export HISTFILE=/commandhistory/.bash_history' >> /home/node/.bashrc

ENV DEVCONTAINER=true

RUN mkdir -p /home/node/.config/gh /home/node/.claude \
    /home/node/.claudeup/profiles /home/node/.claude-mem \
    /home/node/dotfiles /home/node/.npm-global/lib \
    /home/node/.aws /home/node/.local/bin && \
    chown -R node:node /home/node

ARG GIT_DELTA_VERSION=0.18.2
RUN ARCH=$(dpkg --print-architecture) && \
    wget -q "https://github.com/dandavison/delta/releases/download/${GIT_DELTA_VERSION}/git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb" -O /tmp/git-delta.deb && \
    dpkg -i /tmp/git-delta.deb && \
    rm /tmp/git-delta.deb

WORKDIR /workspaces
USER node

ENV NPM_CONFIG_PREFIX=/home/node/.npm-global
ENV PATH=$PATH:/home/node/.npm-global/bin:/home/node/.local/bin:/home/node/.bun/bin
ENV SHELL=/bin/zsh
ENV EDITOR=vim
ENV VISUAL=vim

ARG ZSH_IN_DOCKER_VERSION=1.2.0
RUN sh -c "$(wget -O- https://github.com/deluan/zsh-in-docker/releases/download/v${ZSH_IN_DOCKER_VERSION}/zsh-in-docker.sh)" -- \
    -p git \
    -p fzf \
    -a "source /usr/share/doc/fzf/examples/key-bindings.zsh" \
    -a "source /usr/share/doc/fzf/examples/completion.zsh" \
    -a "export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
    -x

ARG STARSHIP_VERSION=1.24.2
ARG STARSHIP_SHA256_X64=3f12f61883ff324c1dbe7b885fa125d5490960e5cad6a12eeaa34695ec1b5744
ARG STARSHIP_SHA256_ARM64=56b9ff412bbf374d29b99e5ac09a849124cb37a0a13121e8470df32de53c1ea6
RUN ARCH=$(uname -m) \
    && if [ "$ARCH" = "x86_64" ]; then \
        STARSHIP_TARGET="x86_64-unknown-linux-gnu"; STARSHIP_SHA256="${STARSHIP_SHA256_X64}"; \
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then \
        STARSHIP_TARGET="aarch64-unknown-linux-musl"; STARSHIP_SHA256="${STARSHIP_SHA256_ARM64}"; \
    else echo "Unsupported architecture: $ARCH" >&2; exit 1; fi \
    && curl -fsSL -o /tmp/starship.tar.gz \
        "https://github.com/starship/starship/releases/download/v${STARSHIP_VERSION}/starship-${STARSHIP_TARGET}.tar.gz" \
    && echo "${STARSHIP_SHA256}  /tmp/starship.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/starship.tar.gz -C /home/node/.local/bin starship \
    && chmod 755 /home/node/.local/bin/starship \
    && rm /tmp/starship.tar.gz \
    && echo 'eval "$(starship init zsh)"' >> /home/node/.zshrc

COPY --chown=node:node starship.toml /home/node/.config/starship.toml

ARG BUN_VERSION=1.3.10
ARG BUN_SHA256_X64=f57bc0187e39623de716ba3a389fda5486b2d7be7131a980ba54dc7b733d2e08
ARG BUN_SHA256_ARM64=fa5ecb25cafa8e8f5c87a0f833719d46dd0af0a86c7837d806531212d55636d3
RUN ARCH=$(uname -m) \
    && if [ "$ARCH" = "x86_64" ]; then \
        BUN_ARCH="x64"; BUN_SHA256="${BUN_SHA256_X64}"; \
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then \
        BUN_ARCH="aarch64"; BUN_SHA256="${BUN_SHA256_ARM64}"; \
    else echo "Unsupported architecture: $ARCH" >&2; exit 1; fi \
    && curl -fsSL -o /tmp/bun.zip \
        "https://github.com/oven-sh/bun/releases/download/bun-v${BUN_VERSION}/bun-linux-${BUN_ARCH}.zip" \
    && echo "${BUN_SHA256}  /tmp/bun.zip" | sha256sum -c - \
    && mkdir -p /home/node/.bun/bin \
    && unzip -j /tmp/bun.zip "*/bun" -d /home/node/.bun/bin \
    && test -f /home/node/.bun/bin/bun \
    && chmod 755 /home/node/.bun/bin/bun \
    && ln -s /home/node/.bun/bin/bun /home/node/.bun/bin/bunx \
    && rm -f /tmp/bun.zip

RUN git config --global core.excludesfile ~/.gitignore_global && \
    echo ".claude/settings.local.json" > /home/node/.gitignore_global

ARG CLAUDE_VERSION=2.1.76
ARG CLAUDE_SHA256_X64=801a085676c3d54392c42e8e43c44947df7c52132356575f7d9267c4f22d6992
ARG CLAUDE_SHA256_ARM64=40f753c07f070df34ca83e400f746a8279a3fd343967a453d9fbfab2f3ca7acd
RUN ARCH=$(uname -m) \
    && if [ "$ARCH" = "x86_64" ]; then \
        CLAUDE_ARCH="x64"; CLAUDE_SHA256="${CLAUDE_SHA256_X64}"; \
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then \
        CLAUDE_ARCH="arm64"; CLAUDE_SHA256="${CLAUDE_SHA256_ARM64}"; \
    else echo "Unsupported architecture: $ARCH" >&2; exit 1; fi \
    && curl -fsSL -o /tmp/claude \
        "https://storage.googleapis.com/claude-code-dist-86c565f3-f756-42ad-8dfa-d59b1c096819/claude-code-releases/${CLAUDE_VERSION}/linux-${CLAUDE_ARCH}/claude" \
    && echo "${CLAUDE_SHA256}  /tmp/claude" | sha256sum -c - \
    && install -m 755 /tmp/claude /home/node/.local/bin/claude \
    && rm /tmp/claude

USER root

COPY --chmod=755 init-claude-config.sh /usr/local/bin/
COPY --chmod=755 init-config-repo.sh /usr/local/bin/
COPY --chmod=755 init-claudeup.sh /usr/local/bin/
COPY --chmod=755 init-firewall.sh /usr/local/bin/

RUN chsh -s /bin/zsh node

RUN echo "node ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/node-nopasswd && \
    chmod 0440 /etc/sudoers.d/node-nopasswd

USER node
