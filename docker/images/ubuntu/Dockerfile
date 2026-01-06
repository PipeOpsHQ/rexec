FROM ubuntu:22.04

# Avoid prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install essential packages including SSH server
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash \
    curl \
    wget \
    git \
    vim \
    nano \
    htop \
    tree \
    unzip \
    zip \
    sudo \
    openssh-server \
    openssh-client \
    ca-certificates \
    locales \
    build-essential \
    python3 \
    python3-pip \
    nodejs \
    npm \
    net-tools \
    iputils-ping \
    dnsutils \
    procps \
    less \
    man-db \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Set up locale
RUN locale-gen en_US.UTF-8
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# Configure SSH server
RUN mkdir -p /var/run/sshd \
    && mkdir -p /etc/ssh/sshd_config.d \
    && echo "Port 22" > /etc/ssh/sshd_config.d/rexec.conf \
    && echo "PermitRootLogin no" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "PasswordAuthentication no" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "PubkeyAuthentication yes" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "AuthorizedKeysFile .ssh/authorized_keys" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "X11Forwarding no" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "AllowTcpForwarding no" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "PermitTunnel no" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "PrintMotd yes" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "UsePAM yes" >> /etc/ssh/sshd_config.d/rexec.conf \
    && ssh-keygen -A

# Create a non-root user
RUN useradd -m -s /bin/bash -u 1000 user \
    && echo "user ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Set up home directory
WORKDIR /home/user

# Set environment variables
ENV HOME=/home/user
ENV TERM=xterm-256color
ENV PATH="/home/user/.local/bin:${PATH}"

# Create common directories including SSH directory
RUN mkdir -p /home/user/.local/bin \
    && mkdir -p /home/user/workspace \
    && mkdir -p /home/user/.ssh \
    && chmod 700 /home/user/.ssh \
    && touch /home/user/.ssh/authorized_keys \
    && chmod 600 /home/user/.ssh/authorized_keys \
    && chown -R user:user /home/user

# Install rexec CLI
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "amd64" ]; then \
    curl -sSL https://rexec.sh/downloads/rexec-cli-linux-amd64 -o /usr/local/bin/rexec; \
    elif [ "$ARCH" = "arm64" ]; then \
    curl -sSL https://rexec.sh/downloads/rexec-cli-linux-arm64 -o /usr/local/bin/rexec; \
    fi && \
    chmod +x /usr/local/bin/rexec

# Configure bash prompt
RUN echo 'export PS1="\[\033[01;32m\]\u@rexec\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ "' >> /home/user/.bashrc \
    && echo 'alias ll="ls -la"' >> /home/user/.bashrc \
    && echo 'alias la="ls -A"' >> /home/user/.bashrc

# Create MOTD
RUN echo '#!/bin/bash' > /etc/update-motd.d/00-rexec \
    && echo 'echo ""' >> /etc/update-motd.d/00-rexec \
    && echo 'echo "  ██████╗ ███████╗██╗  ██╗███████╗ ██████╗"' >> /etc/update-motd.d/00-rexec \
    && echo 'echo "  ██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝"' >> /etc/update-motd.d/00-rexec \
    && echo 'echo "  ██████╔╝█████╗   ╚███╔╝ █████╗  ██║     "' >> /etc/update-motd.d/00-rexec \
    && echo 'echo "  ██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║     "' >> /etc/update-motd.d/00-rexec \
    && echo 'echo "  ██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗"' >> /etc/update-motd.d/00-rexec \
    && echo 'echo "  ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝"' >> /etc/update-motd.d/00-rexec \
    && echo 'echo ""' >> /etc/update-motd.d/00-rexec \
    && echo 'echo "  Welcome to Rexec Terminal (Ubuntu 22.04)"' >> /etc/update-motd.d/00-rexec \
    && echo 'echo ""' >> /etc/update-motd.d/00-rexec \
    && chmod +x /etc/update-motd.d/00-rexec

# Create entrypoint script that starts SSH and keeps container running
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Switch to non-root user
USER user

# Use entrypoint script
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/bin/bash"]
