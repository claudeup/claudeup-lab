# ABOUTME: Base dev container image for Claude Code sandbox environments.
# ABOUTME: Provides Node.js, dev tools, Claude Code, claudeup, and Bun.

FROM node:22

ARG TZ=America/Denver
ENV TZ="$TZ"

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl wget ca-certificates gnupg2 \
    git git-lfs gh \
    procps htop \
    jq yq \
    ripgrep fd-find \
    build-essential make \
    python3 \
    nano vim \
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

WORKDIR /workspaces
USER node

ENV NPM_CONFIG_PREFIX=/home/node/.npm-global
ENV PATH=$PATH:/home/node/.npm-global/bin:/home/node/.local/bin:/home/node/.bun/bin

RUN curl -fsSL https://bun.sh/install | bash

RUN git config --global core.excludesfile ~/.gitignore_global && \
    echo ".claude/settings.local.json" > /home/node/.gitignore_global

RUN curl -fsSL https://claude.ai/install.sh | bash

USER root

COPY --chmod=755 init-claude-config.sh /usr/local/bin/
COPY --chmod=755 init-config-repo.sh /usr/local/bin/
COPY --chmod=755 init-claudeup.sh /usr/local/bin/

USER node
